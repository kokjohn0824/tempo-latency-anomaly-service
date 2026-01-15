#!/bin/bash

# Test scenarios for twdiw-customer-service-prod
# This script tests various endpoints and scenarios for the customer service

set -e

API_BASE="http://localhost:8080"
SERVICE="twdiw-customer-service-prod"
CURRENT_TIME=$(date +%s)000000000  # Current time in nanoseconds

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}測試 twdiw-customer-service-prod${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to print test result
print_result() {
    local test_name=$1
    local result=$2
    local details=$3
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC} - $test_name"
    elif [ "$result" = "FAIL" ]; then
        echo -e "${RED}✗ FAIL${NC} - $test_name"
    else
        echo -e "${YELLOW}⚠ WARN${NC} - $test_name"
    fi
    
    if [ -n "$details" ]; then
        echo -e "  ${details}"
    fi
    echo ""
}

# Test 1: Health Check Endpoint - Normal Latency
echo -e "${BLUE}測試 1: GET /actuator/health - 正常延遲${NC}"
echo "Baseline: p50=233ms, p95=562ms, mad=43"
echo "測試值: 250ms (正常範圍內)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 250
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "false" ]; then
    print_result "Health endpoint 正常延遲" "PASS" "說明: $EXPLANATION"
else
    print_result "Health endpoint 正常延遲" "FAIL" "預期非異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 2: Health Check Endpoint - High Latency (Anomaly)
echo -e "${BLUE}測試 2: GET /actuator/health - 高延遲異常${NC}"
echo "Baseline: p50=233ms, p95=562ms, mad=43"
echo "測試值: 1500ms (遠超過 p95)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 1500
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "true" ]; then
    print_result "Health endpoint 高延遲檢測" "PASS" "說明: $EXPLANATION"
else
    print_result "Health endpoint 高延遲檢測" "FAIL" "預期異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 3: Auth Refresh Endpoint - Normal Latency
echo -e "${BLUE}測試 3: POST /api/auth/refresh - 正常延遲${NC}"
echo "Baseline: p50=5ms, p95=69ms, mad=1"
echo "測試值: 10ms (正常範圍內)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"POST /api/auth/refresh\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 10
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "false" ]; then
    print_result "Auth refresh 正常延遲" "PASS" "說明: $EXPLANATION"
else
    print_result "Auth refresh 正常延遲" "FAIL" "預期非異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 4: Auth Refresh Endpoint - High Latency (Anomaly)
echo -e "${BLUE}測試 4: POST /api/auth/refresh - 高延遲異常${NC}"
echo "Baseline: p50=5ms, p95=69ms, mad=1"
echo "測試值: 500ms (遠超過 p95)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"POST /api/auth/refresh\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 500
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "true" ]; then
    print_result "Auth refresh 高延遲檢測" "PASS" "說明: $EXPLANATION"
else
    print_result "Auth refresh 高延遲檢測" "FAIL" "預期異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 5: Notifications Stream - Normal Latency
echo -e "${BLUE}測試 5: GET /api/notifications/stream - 正常延遲${NC}"
echo "Baseline: p50=1800465ms, p95=1800958ms, mad=188.5"
echo "測試值: 1800500ms (正常範圍內,這是長連接 SSE endpoint)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /api/notifications/stream\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 1800500
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "false" ]; then
    print_result "Notifications stream 正常延遲" "PASS" "說明: $EXPLANATION"
else
    print_result "Notifications stream 正常延遲" "FAIL" "預期非異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 6: Notifications Stream - Abnormally Short (Anomaly)
echo -e "${BLUE}測試 6: GET /api/notifications/stream - 異常短延遲${NC}"
echo "Baseline: p50=1800465ms, p95=1800958ms, mad=188.5"
echo "測試值: 100ms (異常短,可能是連接過早斷開)"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /api/notifications/stream\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 100
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "true" ]; then
    print_result "Notifications stream 異常短延遲檢測" "PASS" "說明: $EXPLANATION"
else
    print_result "Notifications stream 異常短延遲檢測" "FAIL" "預期異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 7: Scheduler Endpoints - Normal Latency
echo -e "${BLUE}測試 7: AiReplyRetryScheduler.processAiReplies - 正常延遲${NC}"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"AiReplyRetryScheduler.processAiReplies\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 3
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "false" ]; then
    print_result "Scheduler 正常延遲" "PASS" "說明: $EXPLANATION"
else
    print_result "Scheduler 正常延遲" "FAIL" "預期非異常,實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 8: Non-existent Endpoint (No Baseline)
echo -e "${BLUE}測試 8: 不存在的 endpoint - 無 baseline${NC}"

RESPONSE=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /api/nonexistent/endpoint\",
    \"timestampNano\": $CURRENT_TIME,
    \"durationMs\": 100
  }")

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

if [ "$IS_ANOMALY" = "false" ] && echo "$EXPLANATION" | grep -q "no baseline"; then
    print_result "無 baseline 處理" "PASS" "說明: $EXPLANATION"
else
    print_result "無 baseline 處理" "FAIL" "預期非異常且說明包含 'no baseline',實際: $IS_ANOMALY\n  說明: $EXPLANATION"
fi

# Test 9: Query Baseline API
echo -e "${BLUE}測試 9: 查詢 Baseline API${NC}"

# Get current hour in Asia/Taipei timezone
CURRENT_HOUR=$(TZ=Asia/Taipei date +%H | sed 's/^0//')
DAY_OF_WEEK=$(TZ=Asia/Taipei date +%u)
if [ "$DAY_OF_WEEK" -ge 6 ]; then
    DAY_TYPE="weekend"
else
    DAY_TYPE="weekday"
fi

RESPONSE=$(curl -s "$API_BASE/v1/baseline?service=$SERVICE&endpoint=GET%20/actuator/health&hour=$CURRENT_HOUR&dayType=$DAY_TYPE")
P50=$(echo "$RESPONSE" | jq -r '.p50')
P95=$(echo "$RESPONSE" | jq -r '.p95')
SAMPLE_COUNT=$(echo "$RESPONSE" | jq -r '.sampleCount')

if [ "$P50" != "null" ] && [ "$P95" != "null" ]; then
    print_result "Baseline API 查詢" "PASS" "p50=$P50, p95=$P95, samples=$SAMPLE_COUNT (hour=$CURRENT_HOUR, dayType=$DAY_TYPE)"
else
    ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error // empty')
    if [ -n "$ERROR_MSG" ]; then
        print_result "Baseline API 查詢" "WARN" "尚無 baseline 資料: $ERROR_MSG"
    else
        print_result "Baseline API 查詢" "FAIL" "無法取得 baseline 資料: $RESPONSE"
    fi
fi

# Test 10: Time Bucketing - Different Hours
echo -e "${BLUE}測試 10: 時間分桶 - 不同小時${NC}"

# Current hour (16:00 in Asia/Taipei)
CURRENT_HOUR_TIME=$CURRENT_TIME

# Different hour (10:00 in Asia/Taipei) - 6 hours ago
SIX_HOURS_AGO=$((CURRENT_TIME - 6*3600*1000000000))

RESPONSE_CURRENT=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $CURRENT_HOUR_TIME,
    \"durationMs\": 250
  }")

RESPONSE_PAST=$(curl -s -X POST "$API_BASE/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $SIX_HOURS_AGO,
    \"durationMs\": 250
  }")

BUCKET_CURRENT=$(echo "$RESPONSE_CURRENT" | jq -r '.bucket.hourOfDay')
BUCKET_PAST=$(echo "$RESPONSE_PAST" | jq -r '.bucket.hourOfDay')

if [ "$BUCKET_CURRENT" != "$BUCKET_PAST" ]; then
    print_result "時間分桶驗證" "PASS" "當前小時: $BUCKET_CURRENT, 6小時前: $BUCKET_PAST"
else
    print_result "時間分桶驗證" "FAIL" "時間分桶未正確區分不同小時"
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}測試摘要${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "測試的 Service: $SERVICE"
echo "測試的 Endpoints:"
echo "  - GET /actuator/health (health check)"
echo "  - POST /api/auth/refresh (authentication)"
echo "  - GET /api/notifications/stream (SSE long-polling)"
echo "  - AiReplyRetryScheduler.processAiReplies (background job)"
echo ""
echo "測試情境:"
echo "  ✓ 正常延遲檢測"
echo "  ✓ 高延遲異常檢測"
echo "  ✓ 異常短延遲檢測 (針對長連接)"
echo "  ✓ 無 baseline 處理"
echo "  ✓ Baseline API 查詢"
echo "  ✓ 時間分桶驗證"
echo ""

# Additional Redis Data Inspection
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Redis 資料檢查${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo "Duration Keys 數量:"
docker exec tempo-anomaly-redis redis-cli --scan --pattern "dur:$SERVICE:*" | wc -l

echo ""
echo "Baseline Keys 數量:"
docker exec tempo-anomaly-redis redis-cli --scan --pattern "base:$SERVICE:*" | wc -l

echo ""
echo "所有 Endpoints (前 20 個):"
docker exec tempo-anomaly-redis redis-cli --scan --pattern "base:$SERVICE:*" | \
  sed "s|base:$SERVICE:\([^|]*\)|\1|" | \
  cut -d'|' -f1 | \
  sort | uniq | head -20

echo ""
echo -e "${GREEN}測試完成!${NC}"
