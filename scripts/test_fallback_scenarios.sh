#!/usr/bin/env bash

# 測試 fallback 情境：
# 1) Level 1 精確匹配 (使用當前時間)
# 2) Level 2 相鄰時段 (使用凌晨 03:00 的時間桶)
# 3) Level 3-4 全局 fallback 路徑 (使用不存在的服務，最終應為 unavailable)
# 4) 驗證回應包含 baselineSource, fallbackLevel, sourceDetails 欄位
# 5) 顯示詳細的測試結果與統計

set -euo pipefail

API_BASE="http://localhost:8080"
SERVICE="twdiw-customer-service-prod"
ENDPOINT_HEALTH="GET /actuator/health"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PASSED=0
FAILED=0
WARNED=0

exact_count=0
nearby_count=0
daytype_count=0
global_count=0
unavailable_count=0

print_header() {
  echo -e "${BLUE}========================================${NC}"
  echo -e "${BLUE}$1${NC}"
  echo -e "${BLUE}========================================${NC}"
}

pass() { echo -e "${GREEN}✓ PASS${NC} - $1"; PASSED=$((PASSED+1)); }
fail() { echo -e "${RED}✗ FAIL${NC} - $1"; FAILED=$((FAILED+1)); }
warn() { echo -e "${YELLOW}⚠ WARN${NC} - $1"; WARNED=$((WARNED+1)); }

have_cmd() { command -v "$1" >/dev/null 2>&1; }

require_tools() {
  for c in curl jq; do
    if ! have_cmd "$c"; then
      echo "Missing required tool: $c" >&2
      exit 1
    fi
  done
}

field_stats() {
  local src="$1"
  case "$src" in
    exact) exact_count=$((exact_count+1));;
    nearby) nearby_count=$((nearby_count+1));;
    daytype) daytype_count=$((daytype_count+1));;
    global) global_count=$((global_count+1));;
    unavailable) unavailable_count=$((unavailable_count+1));;
  esac
}

get_now_nano() { date +%s000000000; }

# 產生一個其在 Asia/Taipei 時區的「小時 = 3」的時間戳 (奈秒)
get_3am_taipei_nano() {
  # 以目前時間為基準，調整到台北時間的 3 點整所在的小時桶
  local now_s=$(date +%s)
  local cur_hour=$(TZ=Asia/Taipei date +%H | sed 's/^0//')
  # 目標小時 3，計算需要回退/前進的小時差，使得 (cur_hour -> 3)
  local delta=$((cur_hour - 3))
  local ts=$((now_s - delta*3600))
  echo "${ts}000000000"
}

validate_fields() {
  local resp="$1"
  local ok=true
  local src=$(echo "$resp" | jq -r '.baselineSource // empty')
  local lvl=$(echo "$resp" | jq -r '.fallbackLevel // empty')
  local det=$(echo "$resp" | jq -r '.sourceDetails // empty')

  if [ -z "$src" ] || [ "$src" = "null" ]; then ok=false; fi
  if [ -z "$lvl" ] || [ "$lvl" = "null" ]; then ok=false; fi
  if [ -z "$det" ]; then ok=false; fi

  echo "$src" # also return source via stdout for stats
  $ok && return 0 || return 1
}

print_header "Fallback 測試：$SERVICE"
require_tools

echo -e "${CYAN}環境檢查:${NC} $API_BASE/healthz"
if curl -s "$API_BASE/healthz" | grep -q '"status":"ok"'; then
  echo "服務健康檢查 OK"
else
  echo "警告: 健康檢查未通過或服務未啟動，仍嘗試進行測試..." >&2
fi

# ---------- 測試 1：Level 1 精確匹配 (當前時間) ----------
echo -e "\n${BLUE}測試 1: Level 1 精確匹配 (當前時間)${NC}"
NOW_NANO=$(get_now_nano)
REQ_1=$(cat <<JSON
{
  "service": "$SERVICE",
  "endpoint": "$ENDPOINT_HEALTH",
  "timestampNano": $NOW_NANO,
  "durationMs": 300
}
JSON
)
RESP_1=$(curl -s -X POST "$API_BASE/v1/anomaly/check" -H 'Content-Type: application/json' -d "$REQ_1")
SRC_1=$(validate_fields "$RESP_1" && echo ok || echo fail)
FIELDS_OK=true; if [ "$SRC_1" = "fail" ]; then FIELDS_OK=false; fi
SRC_VALUE=$(echo "$RESP_1" | jq -r '.baselineSource // ""')
LVL_VALUE=$(echo "$RESP_1" | jq -r '.fallbackLevel // 0')
DET_VALUE=$(echo "$RESP_1" | jq -r '.sourceDetails // ""')
echo "Response: $RESP_1"
field_stats "$SRC_VALUE"
if $FIELDS_OK; then
  if [ "$SRC_VALUE" = "exact" ] && [ "$LVL_VALUE" = "1" ]; then
    pass "Level 1 命中 (baselineSource=exact, level=1)"
  else
    warn "未命中 Level 1，實際 baselineSource=$SRC_VALUE, level=$LVL_VALUE"
  fi
else
  fail "回應缺少必要欄位 (baselineSource/fallbackLevel/sourceDetails)"
fi

# ---------- 測試 2：Level 2 相鄰時段 (凌晨 03:00) ----------
echo -e "\n${BLUE}測試 2: Level 2 相鄰時段 (目標 03:00 時間桶)${NC}"
TS_3AM=$(get_3am_taipei_nano)
REQ_2=$(cat <<JSON
{
  "service": "$SERVICE",
  "endpoint": "$ENDPOINT_HEALTH",
  "timestampNano": $TS_3AM,
  "durationMs": 300
}
JSON
)
RESP_2=$(curl -s -X POST "$API_BASE/v1/anomaly/check" -H 'Content-Type: application/json' -d "$REQ_2")
SRC_2=$(validate_fields "$RESP_2" && echo ok || echo fail)
FIELDS_OK=true; if [ "$SRC_2" = "fail" ]; then FIELDS_OK=false; fi
SRC_VALUE=$(echo "$RESP_2" | jq -r '.baselineSource // ""')
LVL_VALUE=$(echo "$RESP_2" | jq -r '.fallbackLevel // 0')
DET_VALUE=$(echo "$RESP_2" | jq -r '.sourceDetails // ""')
echo "Response: $RESP_2"
field_stats "$SRC_VALUE"
if $FIELDS_OK; then
  if [ "$SRC_VALUE" = "nearby" ] && [ "$LVL_VALUE" = "2" ]; then
    pass "Level 2 相鄰時段命中 (baselineSource=nearby, level=2)"
  elif [ "$SRC_VALUE" = "exact" ] && [ "$LVL_VALUE" = "1" ]; then
    warn "03:00 仍有精確 baseline，未觸發相鄰時段 fallback"
  elif [ "$SRC_VALUE" = "daytype" ] || [ "$SRC_VALUE" = "global" ]; then
    warn "未命中 L2，但落在更廣域的 fallback (source=$SRC_VALUE, level=$LVL_VALUE)"
  else
    fail "無法判定相鄰時段 fallback 行為 (source=$SRC_VALUE, level=$LVL_VALUE)"
  fi
else
  fail "回應缺少必要欄位 (baselineSource/fallbackLevel/sourceDetails)"
fi

# ---------- 測試 3：Level 3-4 全局 fallback 路徑 (不存在的服務) ----------
echo -e "\n${BLUE}測試 3: Level 3-4 全局 fallback 路徑 (不存在的服務)${NC}"
NONEXIST_SERVICE="${SERVICE}-nonexistent"
REQ_3=$(cat <<JSON
{
  "service": "$NONEXIST_SERVICE",
  "endpoint": "/does/not/exist",
  "timestampNano": $(get_now_nano),
  "durationMs": 100
}
JSON
)
RESP_3=$(curl -s -X POST "$API_BASE/v1/anomaly/check" -H 'Content-Type: application/json' -d "$REQ_3")
SRC_3=$(validate_fields "$RESP_3" && echo ok || echo fail)
FIELDS_OK=true; if [ "$SRC_3" = "fail" ]; then FIELDS_OK=false; fi
SRC_VALUE=$(echo "$RESP_3" | jq -r '.baselineSource // ""')
LVL_VALUE=$(echo "$RESP_3" | jq -r '.fallbackLevel // 0')
DET_VALUE=$(echo "$RESP_3" | jq -r '.sourceDetails // ""')
CANNOT=$(echo "$RESP_3" | jq -r '.cannotDetermine // false')
echo "Response: $RESP_3"
field_stats "$SRC_VALUE"
if $FIELDS_OK; then
  if [ "$SRC_VALUE" = "unavailable" ] && [ "$LVL_VALUE" = "5" ] && [ "$CANNOT" = "true" ]; then
    pass "無資料時最終返回 unavailable (已嘗試 L1→L4)"
  else
    warn "期望 unavailable/5/cannotDetermine=true，但得到 source=$SRC_VALUE, level=$LVL_VALUE, cannot=$CANNOT"
  fi
else
  fail "回應缺少必要欄位 (baselineSource/fallbackLevel/sourceDetails)"
fi

# ---------- 摘要 ----------
print_header "測試摘要"
echo "Service: $SERVICE"
echo "Endpoint(示例): $ENDPOINT_HEALTH"
echo "API: $API_BASE"
echo ""
echo "結果統計:"
echo -e "  ${GREEN}PASS${NC}: $PASSED"
echo -e "  ${YELLOW}WARN${NC}: $WARNED"
echo -e "  ${RED}FAIL${NC}: $FAILED"
echo "  Total: $((PASSED + WARNED + FAILED))"
echo ""
echo "baselineSource 分佈:"
echo "  exact:        $exact_count"
echo "  nearby:       $nearby_count"
echo "  daytype:      $daytype_count"
echo "  global:       $global_count"
echo "  unavailable:  $unavailable_count"
echo ""

if [ "$FAILED" -eq 0 ]; then
  echo -e "${GREEN}完成: 無失敗案例${NC}"
  exit 0
else
  echo -e "${RED}有失敗案例，請檢視上方細節${NC}"
  exit 1
fi

