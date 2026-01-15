#!/bin/bash

# Demo script for /v1/available API and anomaly detection workflow
# This script demonstrates how to use the new API to discover available services
# and then perform anomaly detection on them

set -e

API_BASE="http://localhost:8080"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_step() {
    echo -e "${CYAN}➜ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_header "Demo: 使用 /v1/available API 發現服務並進行異常檢測"

# Step 1: Query available services
print_step "步驟 1: 查詢所有可用的服務和端點"
AVAILABLE_RESPONSE=$(curl -s "$API_BASE/v1/available")

TOTAL_SERVICES=$(echo "$AVAILABLE_RESPONSE" | jq -r '.totalServices')
TOTAL_ENDPOINTS=$(echo "$AVAILABLE_RESPONSE" | jq -r '.totalEndpoints')

echo -e "${BLUE}發現:${NC}"
echo -e "  - 總服務數: ${GREEN}$TOTAL_SERVICES${NC}"
echo -e "  - 總端點數: ${GREEN}$TOTAL_ENDPOINTS${NC}"

if [ "$TOTAL_SERVICES" -eq 0 ]; then
    print_warning "目前沒有可用的服務資料,請等待系統收集更多 traces"
    exit 0
fi

# Step 2: List all available services
print_step "步驟 2: 列出所有可用的服務"
echo "$AVAILABLE_RESPONSE" | jq -r '.services[] | "\(.service)"' | sort -u | while read service; do
    ENDPOINT_COUNT=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[] | select(.service == \"$service\") | .endpoint" | wc -l)
    echo -e "  ${GREEN}●${NC} $service (${ENDPOINT_COUNT} 個端點)"
done

# Step 3: Focus on a specific service
TARGET_SERVICE="twdiw-customer-service-prod"
print_step "步驟 3: 查看 $TARGET_SERVICE 的詳細資訊"

SERVICE_ENDPOINTS=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[] | select(.service == \"$TARGET_SERVICE\")")

if [ -z "$SERVICE_ENDPOINTS" ]; then
    print_warning "服務 $TARGET_SERVICE 目前沒有可用資料"
    # Try to use the first available service instead
    TARGET_SERVICE=$(echo "$AVAILABLE_RESPONSE" | jq -r '.services[0].service')
    print_step "改用第一個可用服務: $TARGET_SERVICE"
    SERVICE_ENDPOINTS=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[] | select(.service == \"$TARGET_SERVICE\")")
fi

echo "$SERVICE_ENDPOINTS" | jq -r '. | "  Endpoint: \(.endpoint)\n  時間桶: \(.buckets | join(", "))\n"'

# Step 4: Get current timestamp
print_step "步驟 4: 取得當前時間資訊"
CURRENT_TIMESTAMP=$(TZ=Asia/Taipei date +%s%N | awk '{printf "%.0f\n", $1}')
CURRENT_HOUR=$(TZ=Asia/Taipei date +%H | sed 's/^0//')
CURRENT_DAY=$(TZ=Asia/Taipei date +%A)
DAY_OF_WEEK=$(TZ=Asia/Taipei date +%u)

if [ "$DAY_OF_WEEK" -ge 6 ]; then
    DAY_TYPE="weekend"
else
    DAY_TYPE="weekday"
fi

echo -e "  當前時間: ${GREEN}$(TZ=Asia/Taipei date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "  時間桶: ${GREEN}${CURRENT_HOUR}|${DAY_TYPE}${NC}"

# Step 5: Find an endpoint with matching time bucket
print_step "步驟 5: 尋找符合當前時間桶的端點"
MATCHING_ENDPOINT=$(echo "$SERVICE_ENDPOINTS" | jq -r "select(.buckets[] | contains(\"$CURRENT_HOUR|$DAY_TYPE\")) | .endpoint" | head -1)

if [ -z "$MATCHING_ENDPOINT" ]; then
    print_warning "沒有端點符合當前時間桶 ($CURRENT_HOUR|$DAY_TYPE)"
    echo -e "\n${YELLOW}可用的時間桶:${NC}"
    echo "$SERVICE_ENDPOINTS" | jq -r '.buckets[]' | sort -u
    
    # Use the first available endpoint anyway
    MATCHING_ENDPOINT=$(echo "$SERVICE_ENDPOINTS" | jq -r '.endpoint' | head -1)
    AVAILABLE_BUCKET=$(echo "$SERVICE_ENDPOINTS" | jq -r "select(.endpoint == \"$MATCHING_ENDPOINT\") | .buckets[0]")
    print_warning "使用第一個可用端點進行示範: $MATCHING_ENDPOINT (時間桶: $AVAILABLE_BUCKET)"
else
    print_success "找到符合的端點: $MATCHING_ENDPOINT"
fi

# Step 6: Test anomaly detection with normal latency
print_step "步驟 6: 測試正常延遲的異常檢測"
NORMAL_DURATION=250

echo -e "${BLUE}請求參數:${NC}"
echo -e "  Service: ${GREEN}$TARGET_SERVICE${NC}"
echo -e "  Endpoint: ${GREEN}$MATCHING_ENDPOINT${NC}"
echo -e "  Duration: ${GREEN}${NORMAL_DURATION}ms${NC}"

NORMAL_RESPONSE=$(curl -s -X 'POST' \
  "$API_BASE/v1/anomaly/check" \
  -H 'Content-Type: application/json' \
  -d "{
  \"durationMs\": $NORMAL_DURATION,
  \"endpoint\": \"$MATCHING_ENDPOINT\",
  \"service\": \"$TARGET_SERVICE\",
  \"timestampNano\": $CURRENT_TIMESTAMP
}")

IS_ANOMALY=$(echo "$NORMAL_RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$NORMAL_RESPONSE" | jq -r '.explanation')

echo -e "\n${BLUE}檢測結果:${NC}"
if [ "$IS_ANOMALY" = "true" ]; then
    print_error "檢測到異常!"
else
    print_success "正常範圍內"
fi
echo -e "  說明: $EXPLANATION"

# Display baseline info if available
BASELINE=$(echo "$NORMAL_RESPONSE" | jq -r '.baseline')
if [ "$BASELINE" != "null" ]; then
    P50=$(echo "$NORMAL_RESPONSE" | jq -r '.baseline.p50')
    P95=$(echo "$NORMAL_RESPONSE" | jq -r '.baseline.p95')
    SAMPLES=$(echo "$NORMAL_RESPONSE" | jq -r '.baseline.sampleCount')
    echo -e "\n${BLUE}Baseline 統計:${NC}"
    echo -e "  P50: ${GREEN}${P50}ms${NC}"
    echo -e "  P95: ${GREEN}${P95}ms${NC}"
    echo -e "  樣本數: ${GREEN}${SAMPLES}${NC}"
fi

# Step 7: Test anomaly detection with high latency
print_step "步驟 7: 測試異常延遲的檢測"
ANOMALY_DURATION=10000

echo -e "${BLUE}請求參數:${NC}"
echo -e "  Service: ${GREEN}$TARGET_SERVICE${NC}"
echo -e "  Endpoint: ${GREEN}$MATCHING_ENDPOINT${NC}"
echo -e "  Duration: ${RED}${ANOMALY_DURATION}ms${NC} (故意設定很高)"

ANOMALY_RESPONSE=$(curl -s -X 'POST' \
  "$API_BASE/v1/anomaly/check" \
  -H 'Content-Type: application/json' \
  -d "{
  \"durationMs\": $ANOMALY_DURATION,
  \"endpoint\": \"$MATCHING_ENDPOINT\",
  \"service\": \"$TARGET_SERVICE\",
  \"timestampNano\": $CURRENT_TIMESTAMP
}")

IS_ANOMALY=$(echo "$ANOMALY_RESPONSE" | jq -r '.isAnomaly')
EXPLANATION=$(echo "$ANOMALY_RESPONSE" | jq -r '.explanation')

echo -e "\n${BLUE}檢測結果:${NC}"
if [ "$IS_ANOMALY" = "true" ]; then
    print_error "檢測到異常! (符合預期)"
else
    print_warning "未檢測到異常 (可能閾值設定較寬鬆)"
fi
echo -e "  說明: $EXPLANATION"

# Step 8: Query baseline directly
print_step "步驟 8: 直接查詢 Baseline 統計資料"

BASELINE_RESPONSE=$(curl -s "$API_BASE/v1/baseline?service=$TARGET_SERVICE&endpoint=$(echo "$MATCHING_ENDPOINT" | jq -sRr @uri)&hour=$CURRENT_HOUR&dayType=$DAY_TYPE")

if echo "$BASELINE_RESPONSE" | jq -e '.p50' > /dev/null 2>&1; then
    echo -e "${BLUE}Baseline 詳細資訊:${NC}"
    echo "$BASELINE_RESPONSE" | jq .
else
    print_warning "無法取得 baseline 資料: $BASELINE_RESPONSE"
fi

# Summary
print_header "總結"

echo -e "${GREEN}✓${NC} 成功使用 /v1/available API 發現可用服務"
echo -e "${GREEN}✓${NC} 成功對 $TARGET_SERVICE 進行異常檢測"
echo -e "${GREEN}✓${NC} 驗證了正常和異常兩種情況"

echo -e "\n${BLUE}相關連結:${NC}"
echo -e "  API 文檔: ${CYAN}http://localhost:8080/swagger/index.html${NC}"
echo -e "  可用服務: ${CYAN}curl http://localhost:8080/v1/available${NC}"
echo -e "  健康檢查: ${CYAN}curl http://localhost:8080/healthz${NC}"

echo -e "\n${YELLOW}提示:${NC} 使用 /v1/available API 可以:"
echo -e "  • 發現哪些服務已準備好進行異常檢測"
echo -e "  • 查看每個端點的可用時間桶"
echo -e "  • 整合到監控系統中自動發現服務"
echo -e "  • 在進行異常檢測前驗證資料可用性"
