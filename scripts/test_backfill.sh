#!/usr/bin/env bash

# 測試 Backfill 與可用性/回退行為的綜合腳本
# 目標:
# 1) 檢查服務日誌是否包含 'tempo backfill: completed'
# 2) 使用 /v1/available API 查看各時段的資料分布
# 3) 對比 backfill 前後的資料覆蓋率
# 4) 測試凌晨時段 (如 03:00) 是否能透過 fallback 得到判斷
# 5) 統計各 fallback level 的使用分布
#
# 使用方式 (建議在服務啟動後立即執行，以利取得 backfill 前/後對照):
#   API_BASE=http://localhost:8080 \
#   LOG_FILE=/path/to/service.log \
#   ./scripts/test_backfill.sh
#
# 也支援其他日誌來源 (擇一設定):
#   - DOCKER_CONTAINER=my-service-container  (使用 docker logs)
#   - K8S_POD=my-pod -n my-ns               (使用 kubectl logs)
# 若未設定任何日誌來源，僅執行 API 相關測試並警告略過日誌檢查。

set -euo pipefail

# ------------------------- 基本設定 -------------------------
API_BASE=${API_BASE:-"http://localhost:8080"}

# 日誌來源 (擇一):
LOG_FILE=${LOG_FILE:-""}             # 直接檔案: /var/log/xxx.log
DOCKER_CONTAINER=${DOCKER_CONTAINER:-""} # docker container 名稱
K8S_POD=${K8S_POD:-""}               # kubectl 目標 pod 名稱
K8S_NAMESPACE=${K8S_NAMESPACE:-""}    # kubectl 命名空間 (可選)

# 3:00 測試服務/端點 (若可從 /v1/available 擷取會覆寫)
DEFAULT_SERVICE=${DEFAULT_SERVICE:-"twdiw-customer-service-prod"}
DEFAULT_ENDPOINT=${DEFAULT_ENDPOINT:-"GET /actuator/health"}

# 等待 backfill 完成的最長秒數 (若需要等待)
BACKFILL_WAIT_TIMEOUT=${BACKFILL_WAIT_TIMEOUT:-900}

# 取樣數量: 回退分布統計時要送出的請求數 (越大越準確，越久)
FALLBACK_SAMPLES=${FALLBACK_SAMPLES:-40}

# 暫存檔案
OUT_DIR=${OUT_DIR:-"/tmp"}
PRE_FILE="$OUT_DIR/available_pre.json"
POST_FILE="$OUT_DIR/available_post.json"

# ------------------------- 輸出樣式 -------------------------
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PASSED=0
FAILED=0
WARNED=0

pass() { echo -e "${GREEN}✓ PASS${NC} - $1"; PASSED=$((PASSED+1)); }
fail() { echo -e "${RED}✗ FAIL${NC} - $1"; FAILED=$((FAILED+1)); }
warn() { echo -e "${YELLOW}⚠ WARN${NC} - $1"; WARNED=$((WARNED+1)); }

print_header() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}========================================${NC}\n"
}

have_cmd() { command -v "$1" >/dev/null 2>&1; }

require_tools() {
  for c in curl jq; do
    if ! have_cmd "$c"; then
      echo "Missing required tool: $c" >&2
      exit 1
    fi
  done
}

# 取得目前時間 (奈秒)
now_nano() { date +%s000000000; }

# 取得台北時區 03:00 整點所在小時桶的時間 (奈秒)
ts_3am_taipei_nano() {
  local now_s=$(date +%s)
  local cur_hour=$(TZ=Asia/Taipei date +%H | sed 's/^0//')
  local delta=$((cur_hour - 3))
  local ts=$((now_s - delta*3600))
  echo "${ts}000000000"
}

# ------------------------- 日誌讀取 -------------------------
logs_grep() {
  local pattern="$1"
  # 檔案優先
  if [ -n "$LOG_FILE" ] && [ -f "$LOG_FILE" ]; then
    if grep -q "$pattern" "$LOG_FILE"; then return 0; else return 1; fi
  fi
  # docker logs
  if [ -n "$DOCKER_CONTAINER" ] && have_cmd docker; then
    if docker logs "$DOCKER_CONTAINER" 2>&1 | grep -q "$pattern"; then return 0; else return 1; fi
  fi
  # kubectl logs
  if [ -n "$K8S_POD" ] && have_cmd kubectl; then
    local ns_args=()
    [ -n "$K8S_NAMESPACE" ] && ns_args=( -n "$K8S_NAMESPACE" )
    if kubectl logs "${ns_args[@]}" "$K8S_POD" 2>&1 | grep -q "$pattern"; then return 0; else return 1; fi
  fi
  return 2  # 無日誌來源
}

logs_collect() {
  # 回傳完整日誌文字 (若可得)
  if [ -n "$LOG_FILE" ] && [ -f "$LOG_FILE" ]; then
    cat "$LOG_FILE"
    return 0
  fi
  if [ -n "$DOCKER_CONTAINER" ] && have_cmd docker; then
    docker logs "$DOCKER_CONTAINER" 2>&1
    return 0
  fi
  if [ -n "$K8S_POD" ] && have_cmd kubectl; then
    local ns_args=()
    [ -n "$K8S_NAMESPACE" ] && ns_args=( -n "$K8S_NAMESPACE" )
    kubectl logs "${ns_args[@]}" "$K8S_POD" 2>&1
    return 0
  fi
  return 2
}

wait_for_backfill_complete() {
  local timeout=${1:-$BACKFILL_WAIT_TIMEOUT}
  local waited=0
  echo -n "等待 backfill 完成中"
  while [ $waited -lt $timeout ]; do
    if logs_grep "tempo backfill: completed"; then
      echo ""
      return 0
    fi
    echo -n "."; sleep 3; waited=$((waited+3))
  done
  echo ""
  return 1
}

# ------------------------- 可用性與覆蓋率 -------------------------
fetch_available() {
  curl -s "$API_BASE/v1/available"
}

validate_json() {
  echo "$1" | jq . >/dev/null 2>&1
}

coverage_metrics() {
  # 輸入: /v1/available JSON。輸出: 三個欄位，以空白分隔
  # 1) endpoints 總數 2) 全部 buckets 的總數 3) 每 endpoint 平均 buckets 數
  local json="$1"
  local endpoints=$(echo "$json" | jq -r '.services | length')
  local buckets_total=$(echo "$json" | jq -r '[.services[].buckets | length] | add // 0')
  local avg=0
  if [ "$endpoints" -gt 0 ]; then
    avg=$(echo "scale=2; $buckets_total / $endpoints" | bc 2>/dev/null || echo 0)
  fi
  echo "$endpoints $buckets_total $avg"
}

print_bucket_distribution() {
  # 顯示各小時的分佈情況 (0..23) 與 dayType
  local json="$1"
  echo -e "${CYAN}各時段 bucket 分佈 (樣本為所有 service/endpoint 的 buckets 彙整):${NC}"
  for h in $(seq 0 23); do
    local hour_key="${h}|weekday"
    local hour_key_wend="${h}|weekend"
    local c1=$(echo "$json" | jq -r --arg k "$hour_key" '[.services[].buckets[] | select(. == $k)] | length')
    local c2=$(echo "$json" | jq -r --arg k "$hour_key_wend" '[.services[].buckets[] | select(. == $k)] | length')
    printf "  %02d: weekday=%-4s weekend=%-4s\n" "$h" "$c1" "$c2"
  done
}

# ------------------------- 回退檢查 -------------------------
validate_fields() {
  local resp="$1"
  local src=$(echo "$resp" | jq -r '.baselineSource // empty')
  local lvl=$(echo "$resp" | jq -r '.fallbackLevel // empty')
  local det=$(echo "$resp" | jq -r '.sourceDetails // empty')
  [ -n "$src" ] && [ -n "$lvl" ] && [ -n "$det" ]
}

post_check() {
  local service="$1"; shift
  local endpoint="$1"; shift
  local ts_nano="$1"; shift
  local duration_ms=${1:-300}
  curl -s -X POST "$API_BASE/v1/anomaly/check" -H 'Content-Type: application/json' -d @- <<JSON
{
  "service": "$service",
  "endpoint": "$endpoint",
  "timestampNano": $ts_nano,
  "durationMs": $duration_ms
}
JSON
}

collect_fallback_stats() {
  local samples=$1
  local available_json="$2"
  local exact=0 nearby=0 daytype=0 global=0 unavailable=0

  local total_endpoints=$(echo "$available_json" | jq -r '.services | length')
  if [ "$total_endpoints" -eq 0 ]; then
    echo "$exact $nearby $daytype $global $unavailable"
    return 0
  fi

  for i in $(seq 1 "$samples"); do
    # 隨機挑一個 endpoint 與一個小時 (0..23)
    local idx=$((RANDOM % total_endpoints))
    local svc=$(echo "$available_json" | jq -r ".services[$idx].service")
    local ep=$(echo "$available_json" | jq -r ".services[$idx].endpoint")
    local hr=$((RANDOM % 24))
    # 合成一個接近該小時桶的時間戳：取 UTC 今日該小時
    local today=$(date -u +%Y-%m-%d)
    local ts=$(date -u -j -f "%Y-%m-%d %H:%M:%S" "$today $(printf "%02d" "$hr"):00:00" +%s 2>/dev/null || date -u -d "$today $(printf "%02d" "$hr"):00:00" +%s)
    local ts_nano=$(printf "%d000000000" "$ts")
    local resp=$(post_check "$svc" "$ep" "$ts_nano" 300)
    local src=$(echo "$resp" | jq -r '.baselineSource // ""')
    case "$src" in
      exact) exact=$((exact+1));;
      nearby) nearby=$((nearby+1));;
      daytype) daytype=$((daytype+1));;
      global) global=$((global+1));;
      unavailable) unavailable=$((unavailable+1));;
    esac
  done
  echo "$exact $nearby $daytype $global $unavailable"
}

# ------------------------- 主流程 -------------------------
require_tools

print_header "測試說明"
cat <<EOF
此腳本會：
- 檢查服務日誌是否出現 'tempo backfill: completed' 以確認回填完成。
- 呼叫 /v1/available 統計可用服務/端點與各小時桶的分佈。
- 嘗試在 backfill 完成前後各抓一次 /v1/available 以比較覆蓋率。
- 以台北時區 03:00 送出 anomaly/check，驗證 fallback 能取得判斷。
- 取樣多次 anomaly/check，統計 fallback level 的使用分布。
EOF

print_header "健康檢查"
if curl -s "$API_BASE/healthz" | grep -q '"status":"ok"'; then
  pass "健康檢查成功 ($API_BASE/healthz)"
else
  warn "健康檢查未通過，仍嘗試繼續測試 (請確保服務已啟動)"
fi

print_header "日誌檢查：Backfill 完成訊息"
if logs_grep "tempo backfill: completed"; then
  pass "發現 'tempo backfill: completed' 日誌，系統已完成回填"
  BACKFILL_ALREADY_DONE=1
else
  if [ $? -eq 2 ]; then
    warn "未設定有效日誌來源，略過日誌檢查 (可設定 LOG_FILE/DOCKER_CONTAINER/K8S_POD)"
    BACKFILL_ALREADY_DONE=0
  else
    warn "尚未見到完成訊息，將先擷取 pre 快照，並在背景等待完成"
    BACKFILL_ALREADY_DONE=0
  fi
fi

print_header "擷取 /v1/available (pre)"
PRE_JSON=$(fetch_available)
if ! validate_json "$PRE_JSON"; then
  fail "/v1/available 回應非 JSON：$PRE_JSON"
  exit 1
fi
echo "$PRE_JSON" > "$PRE_FILE"
echo "pre 快照已儲存: $PRE_FILE"

# 若尚未完成，嘗試等待完成訊息後再擷取 post
if [ "${BACKFILL_ALREADY_DONE:-0}" -eq 0 ]; then
  if wait_for_backfill_complete "$BACKFILL_WAIT_TIMEOUT"; then
    pass "偵測到 backfill 完成訊息"
  else
    warn "在 ${BACKFILL_WAIT_TIMEOUT}s 內未偵測到 backfill 完成訊息，仍進行後續分析 (post 可能與 pre 相近)"
  fi
fi

print_header "擷取 /v1/available (post)"
POST_JSON=$(fetch_available)
if ! validate_json "$POST_JSON"; then
  fail "/v1/available 回應非 JSON：$POST_JSON"
  exit 1
fi
echo "$POST_JSON" > "$POST_FILE"
echo "post 快照已儲存: $POST_FILE"

print_header "分佈與覆蓋率統計"
TOTAL_SERV=$(echo "$POST_JSON" | jq -r '.totalServices // 0')
TOTAL_EP=$(echo "$POST_JSON" | jq -r '.totalEndpoints // 0')
echo "目前可用：services=$TOTAL_SERV, endpoints=$TOTAL_EP"
print_bucket_distribution "$POST_JSON"

read -r PRE_EP PRE_BUCKETS PRE_AVG <<<"$(coverage_metrics "$PRE_JSON")"
read -r POST_EP POST_BUCKETS POST_AVG <<<"$(coverage_metrics "$POST_JSON")"

echo ""
echo "覆蓋率 (endpoint 數、總 buckets、平均每 endpoint buckets)"
printf "  pre:  endpoints=%-5s buckets=%-6s avg=%s\n" "$PRE_EP" "$PRE_BUCKETS" "$PRE_AVG"
printf "  post: endpoints=%-5s buckets=%-6s avg=%s\n" "$POST_EP" "$POST_BUCKETS" "$POST_AVG"

if [ "$POST_BUCKETS" -gt "$PRE_BUCKETS" ]; then
  pass "回填後 buckets 總數增加 (+$((POST_BUCKETS-PRE_BUCKETS)))"
else
  warn "回填後 buckets 總數未見明顯增加 (可能已完成或資料有限)"
fi

# 取得用於 03:00 測試的示例 service/endpoint
SVC_FOR_3AM="$DEFAULT_SERVICE"
EP_FOR_3AM="$DEFAULT_ENDPOINT"
if [ "$(echo "$POST_JSON" | jq -r '.services | length')" -gt 0 ]; then
  SVC_FOR_3AM=$(echo "$POST_JSON" | jq -r '.services[0].service')
  EP_FOR_3AM=$(echo "$POST_JSON" | jq -r '.services[0].endpoint')
fi

print_header "凌晨 03:00 回退檢查 (fallback 能否判斷)"
TS_3AM=$(ts_3am_taipei_nano)
RESP_3AM=$(post_check "$SVC_FOR_3AM" "$EP_FOR_3AM" "$TS_3AM" 300)
echo "Response: $RESP_3AM"
if validate_fields "$RESP_3AM"; then
  SRC=$(echo "$RESP_3AM" | jq -r '.baselineSource')
  LVL=$(echo "$RESP_3AM" | jq -r '.fallbackLevel')
  if [ "$SRC" != "unavailable" ] && [ "$LVL" -ge 1 ] && [ "$LVL" -le 4 ]; then
    pass "03:00 透過 fallback 取得判斷 (source=$SRC, level=$LVL)"
  else
    fail "03:00 無法取得有效判斷 (source=$SRC, level=$LVL)"
  fi
else
  fail "03:00 回應缺少必要欄位 (baselineSource/fallbackLevel/sourceDetails)"
fi

print_header "統計 fallback level 使用分布 (取樣 $FALLBACK_SAMPLES 次)"
read -r C_EX C_NEAR C_DAY C_GLOB C_UNAVAIL <<<"$(collect_fallback_stats "$FALLBACK_SAMPLES" "$POST_JSON")"
TOTAL=$((C_EX + C_NEAR + C_DAY + C_GLOB + C_UNAVAIL))
PCT() { local n=$1; [ $TOTAL -gt 0 ] && echo "scale=2; 100*$n/$TOTAL" | bc || echo 0; }
printf "  exact       : %3d (%5.2f%%)\n" "$C_EX" "$(PCT "$C_EX")"
printf "  nearby      : %3d (%5.2f%%)\n" "$C_NEAR" "$(PCT "$C_NEAR")"
printf "  daytype     : %3d (%5.2f%%)\n" "$C_DAY" "$(PCT "$C_DAY")"
printf "  global      : %3d (%5.2f%%)\n" "$C_GLOB" "$(PCT "$C_GLOB")"
printf "  unavailable : %3d (%5.2f%%)\n" "$C_UNAVAIL" "$(PCT "$C_UNAVAIL")"

if [ $C_UNAVAIL -eq 0 ]; then
  pass "所有取樣皆能回退取得某種基準 (unavailable=0)"
else
  warn "有 $C_UNAVAIL 筆取樣為 unavailable，可能資料仍不足或尚未完全回填"
fi

print_header "測試總結"
echo "PASS=$PASSED, WARN=$WARNED, FAIL=$FAILED"
echo "pre 快照:  $PRE_FILE"
echo "post 快照: $POST_FILE"
echo "API_BASE:   $API_BASE"
if [ -n "$LOG_FILE" ]; then echo "LOG_FILE:  $LOG_FILE"; fi
if [ -n "$DOCKER_CONTAINER" ]; then echo "DOCKER:    $DOCKER_CONTAINER"; fi
if [ -n "$K8S_POD" ]; then echo "K8S POD:   $K8S_POD (ns=$K8S_NAMESPACE)"; fi

if [ $FAILED -eq 0 ]; then
  exit 0
else
  exit 1
fi

