#!/bin/bash
# 測試情境腳本 - 驗證 Tempo Latency Anomaly Service 功能

set -e

BASE_URL="http://localhost:8080"
REDIS_CLI="docker exec tempo-anomaly-redis redis-cli"

echo "========================================="
echo "Tempo Latency Anomaly Service 測試"
echo "========================================="
echo ""

# 顏色定義
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 測試結果計數
PASSED=0
FAILED=0

# 測試函數
test_case() {
    local name="$1"
    local description="$2"
    echo ""
    echo "📋 測試案例: $name"
    echo "   描述: $description"
}

pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAILED++))
}

info() {
    echo -e "${YELLOW}ℹ️  INFO${NC}: $1"
}

# ========================================
# 測試 1: 健康檢查
# ========================================
test_case "Test 1" "健康檢查端點"
response=$(curl -s "$BASE_URL/healthz")
if echo "$response" | grep -q '"status":"ok"'; then
    pass "健康檢查返回 OK"
else
    fail "健康檢查失敗: $response"
fi

# ========================================
# 測試 2: 檢查 Redis 資料
# ========================================
test_case "Test 2" "驗證 Redis 中有 trace 資料"

dur_count=$($REDIS_CLI KEYS "dur:*" | wc -l | tr -d ' ')
base_count=$($REDIS_CLI KEYS "base:*" | wc -l | tr -d ' ')

info "Duration keys: $dur_count"
info "Baseline keys: $base_count"

if [ "$dur_count" -gt 0 ]; then
    pass "Redis 中有 $dur_count 個 duration keys"
else
    fail "Redis 中沒有 duration keys"
fi

if [ "$base_count" -gt 0 ]; then
    pass "Redis 中有 $base_count 個 baseline keys"
else
    fail "Redis 中沒有 baseline keys"
fi

# ========================================
# 測試 3: 查詢 Baseline 統計
# ========================================
test_case "Test 3" "查詢特定服務的 baseline 統計"

# 從 Redis 中獲取一個實際存在的 baseline key
sample_key=$($REDIS_CLI KEYS "base:*" | head -1)
if [ -n "$sample_key" ]; then
    # 解析 key 格式: base:service|endpoint|hour|dayType
    service=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f1)
    endpoint=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f2)
    hour=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f3)
    dayType=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f4)
    
    info "測試 key: $sample_key"
    info "Service: $service"
    info "Endpoint: $endpoint"
    info "Hour: $hour"
    info "DayType: $dayType"
    
    # 查詢 baseline
    baseline_url="$BASE_URL/v1/baseline?service=$(echo $service | jq -sRr @uri)&endpoint=$(echo $endpoint | jq -sRr @uri)&hour=$hour&dayType=$dayType"
    response=$(curl -s "$baseline_url")
    
    if echo "$response" | jq -e '.p50' > /dev/null 2>&1; then
        p50=$(echo "$response" | jq -r '.p50')
        p95=$(echo "$response" | jq -r '.p95')
        mad=$(echo "$response" | jq -r '.mad')
        count=$(echo "$response" | jq -r '.sampleCount')
        
        pass "成功查詢 baseline - P50: ${p50}ms, P95: ${p95}ms, MAD: ${mad}ms, Samples: $count"
    else
        fail "Baseline 查詢失敗或格式錯誤: $response"
    fi
else
    fail "Redis 中沒有 baseline keys 可供測試"
fi

# ========================================
# 測試 4: 正常請求 (不應該是異常)
# ========================================
test_case "Test 4" "檢測正常延遲的請求 (應該不是異常)"

if [ -n "$sample_key" ]; then
    # 使用 p50 作為正常請求的延遲
    baseline_data=$($REDIS_CLI HGETALL "$sample_key")
    p50=$(echo "$baseline_data" | grep -A1 "^p50$" | tail -1)
    
    if [ -n "$p50" ] && [ "$p50" != "0" ]; then
        info "使用 P50 延遲: ${p50}ms"
        
        # 構造檢查請求
        check_payload=$(cat <<EOF
{
  "service": "$service",
  "endpoint": "$endpoint",
  "timestampNano": $(date +%s)000000000,
  "durationMs": $p50
}
EOF
)
        
        response=$(curl -s -X POST "$BASE_URL/v1/anomaly/check" \
            -H "Content-Type: application/json" \
            -d "$check_payload")
        
        is_anomaly=$(echo "$response" | jq -r '.isAnomaly')
        explanation=$(echo "$response" | jq -r '.explanation')
        
        info "Response: $response"
        
        if [ "$is_anomaly" = "false" ]; then
            pass "正常請求正確判定為非異常"
        else
            fail "正常請求被誤判為異常: $explanation"
        fi
    else
        info "跳過測試 - 無法獲取 p50 值"
    fi
fi

# ========================================
# 測試 5: 異常請求 (高延遲)
# ========================================
test_case "Test 5" "檢測高延遲的異常請求"

if [ -n "$sample_key" ]; then
    baseline_data=$($REDIS_CLI HGETALL "$sample_key")
    p95=$(echo "$baseline_data" | grep -A1 "^p95$" | tail -1)
    mad=$(echo "$baseline_data" | grep -A1 "^mad$" | tail -1)
    
    if [ -n "$p95" ] && [ "$p95" != "0" ] && [ -n "$mad" ]; then
        # 計算異常閾值: p95 + 3*MAD
        threshold=$(echo "$p95 + 3 * $mad" | bc)
        anomaly_duration=$(echo "$threshold * 1.5" | bc | cut -d. -f1)
        
        info "P95: ${p95}ms, MAD: ${mad}ms"
        info "異常閾值: ${threshold}ms"
        info "測試延遲: ${anomaly_duration}ms"
        
        check_payload=$(cat <<EOF
{
  "service": "$service",
  "endpoint": "$endpoint",
  "timestampNano": $(date +%s)000000000,
  "durationMs": $anomaly_duration
}
EOF
)
        
        response=$(curl -s -X POST "$BASE_URL/v1/anomaly/check" \
            -H "Content-Type: application/json" \
            -d "$check_payload")
        
        is_anomaly=$(echo "$response" | jq -r '.isAnomaly')
        explanation=$(echo "$response" | jq -r '.explanation')
        
        info "Response: $response"
        
        if [ "$is_anomaly" = "true" ]; then
            pass "高延遲請求正確判定為異常: $explanation"
        else
            fail "高延遲請求未被判定為異常"
        fi
    else
        info "跳過測試 - 無法獲取 p95/mad 值"
    fi
fi

# ========================================
# 測試 6: 無 Baseline 的新服務
# ========================================
test_case "Test 6" "檢測沒有 baseline 的新服務"

check_payload=$(cat <<EOF
{
  "service": "test-new-service",
  "endpoint": "/test/endpoint",
  "timestampNano": $(date +%s)000000000,
  "durationMs": 100
}
EOF
)

response=$(curl -s -X POST "$BASE_URL/v1/anomaly/check" \
    -H "Content-Type: application/json" \
    -d "$check_payload")

is_anomaly=$(echo "$response" | jq -r '.isAnomaly')
explanation=$(echo "$response" | jq -r '.explanation')

info "Response: $response"

if [ "$is_anomaly" = "false" ] && echo "$explanation" | grep -q "no baseline"; then
    pass "新服務正確返回無 baseline 狀態"
else
    fail "新服務處理不符合預期"
fi

# ========================================
# 測試 7: 時間分桶驗證
# ========================================
test_case "Test 7" "驗證時間分桶邏輯 (不同小時應該有不同的 baseline)"

# 檢查是否有不同小時的 baseline
hour_keys=$($REDIS_CLI KEYS "base:*" | head -20)
unique_hours=$(echo "$hour_keys" | cut -d'|' -f3 | sort -u | wc -l | tr -d ' ')

info "發現 $unique_hours 個不同的小時分桶"

if [ "$unique_hours" -gt 1 ]; then
    pass "時間分桶正常工作,有多個小時的 baseline"
else
    info "目前只有單一小時的資料 (可能需要更長時間收集)"
fi

# ========================================
# 測試 8: 工作日/週末分類
# ========================================
test_case "Test 8" "驗證工作日/週末分類"

weekday_count=$($REDIS_CLI KEYS "base:*weekday" | wc -l | tr -d ' ')
weekend_count=$($REDIS_CLI KEYS "base:*weekend" | wc -l | tr -d ' ')

info "Weekday baselines: $weekday_count"
info "Weekend baselines: $weekend_count"

current_day=$(date +%u)  # 1=Monday, 7=Sunday
if [ "$current_day" -ge 1 ] && [ "$current_day" -le 5 ]; then
    expected_type="weekday"
else
    expected_type="weekend"
fi

info "今天應該是: $expected_type"

if [ "$expected_type" = "weekday" ] && [ "$weekday_count" -gt 0 ]; then
    pass "工作日分類正確"
elif [ "$expected_type" = "weekend" ] && [ "$weekend_count" -gt 0 ]; then
    pass "週末分類正確"
else
    info "需要更多時間收集不同日期類型的資料"
fi

# ========================================
# 測試 9: Metrics 端點
# ========================================
test_case "Test 9" "驗證 Prometheus metrics 端點"

response=$(curl -s "$BASE_URL/metrics")
if echo "$response" | grep -q "go_"; then
    pass "Metrics 端點正常運作"
else
    fail "Metrics 端點返回異常: $response"
fi

# ========================================
# 測試 10: 持續拉取驗證
# ========================================
test_case "Test 10" "驗證 Tempo 持續拉取"

initial_count=$($REDIS_CLI KEYS "dur:*" | wc -l | tr -d ' ')
info "初始 duration keys: $initial_count"
info "等待 20 秒讓 poller 再次執行..."

sleep 20

final_count=$($REDIS_CLI KEYS "dur:*" | wc -l | tr -d ' ')
info "最終 duration keys: $final_count"

if [ "$final_count" -ge "$initial_count" ]; then
    pass "Tempo poller 持續運作中"
else
    fail "Duration keys 數量減少,可能有問題"
fi

# ========================================
# 測試總結
# ========================================
echo ""
echo "========================================="
echo "測試總結"
echo "========================================="
echo -e "${GREEN}通過: $PASSED${NC}"
echo -e "${RED}失敗: $FAILED${NC}"
echo "總計: $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 所有測試通過!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  有 $FAILED 個測試失敗${NC}"
    exit 1
fi
