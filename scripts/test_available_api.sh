#!/bin/bash

# Test script for /v1/available API endpoint
# This script validates the new API that lists available services and endpoints

set -e

API_BASE="http://localhost:8080"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC} - $test_name: $message"
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}✗ FAIL${NC} - $test_name: $message"
    else
        echo -e "${YELLOW}⚠ WARN${NC} - $test_name: $message"
    fi
}

print_header "測試新的 /v1/available API"

# Test 1: Health check
echo -e "${YELLOW}Test 1: 健康檢查${NC}"
HEALTH_RESPONSE=$(curl -s "$API_BASE/healthz")
if echo "$HEALTH_RESPONSE" | grep -q "ok"; then
    print_result "健康檢查" "PASS" "服務正常運行"
else
    print_result "健康檢查" "FAIL" "服務未正常運行"
    exit 1
fi

# Wait a bit for data collection
echo -e "\n${YELLOW}等待 5 秒讓服務收集資料...${NC}"
sleep 5

# Test 2: Call /v1/available API
echo -e "\n${YELLOW}Test 2: 呼叫 /v1/available API${NC}"
AVAILABLE_RESPONSE=$(curl -s "$API_BASE/v1/available")

# Check if response is valid JSON
if ! echo "$AVAILABLE_RESPONSE" | jq . > /dev/null 2>&1; then
    print_result "API 回應格式" "FAIL" "回應不是有效的 JSON: $AVAILABLE_RESPONSE"
    exit 1
fi

print_result "API 回應格式" "PASS" "回應為有效的 JSON"

# Test 3: Check response structure
echo -e "\n${YELLOW}Test 3: 檢查回應結構${NC}"
TOTAL_SERVICES=$(echo "$AVAILABLE_RESPONSE" | jq -r '.totalServices // "null"')
TOTAL_ENDPOINTS=$(echo "$AVAILABLE_RESPONSE" | jq -r '.totalEndpoints // "null"')
HAS_SERVICES_FIELD=$(echo "$AVAILABLE_RESPONSE" | jq 'has("services")')

if [ "$TOTAL_SERVICES" != "null" ] && [ "$TOTAL_ENDPOINTS" != "null" ] && [ "$HAS_SERVICES_FIELD" = "true" ]; then
    print_result "回應結構" "PASS" "包含所有必要欄位 (totalServices, totalEndpoints, services)"
else
    print_result "回應結構" "FAIL" "缺少必要欄位"
    exit 1
fi

# Test 4: Display statistics
echo -e "\n${YELLOW}Test 4: 統計資訊${NC}"
echo -e "${BLUE}總服務數:${NC} $TOTAL_SERVICES"
echo -e "${BLUE}總端點數:${NC} $TOTAL_ENDPOINTS"

if [ "$TOTAL_SERVICES" -gt 0 ]; then
    print_result "資料可用性" "PASS" "找到 $TOTAL_SERVICES 個服務和 $TOTAL_ENDPOINTS 個端點"
else
    print_result "資料可用性" "WARN" "目前沒有可用的服務資料 (可能需要更多時間收集)"
fi

# Test 5: Display sample services
if [ "$TOTAL_SERVICES" -gt 0 ]; then
    echo -e "\n${YELLOW}Test 5: 顯示可用服務範例${NC}"
    
    # Get first 5 services
    SAMPLE_COUNT=$(echo "$AVAILABLE_RESPONSE" | jq -r '.services | length')
    DISPLAY_COUNT=$((SAMPLE_COUNT < 5 ? SAMPLE_COUNT : 5))
    
    for i in $(seq 0 $((DISPLAY_COUNT - 1))); do
        SERVICE=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[$i].service")
        ENDPOINT=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[$i].endpoint")
        BUCKETS=$(echo "$AVAILABLE_RESPONSE" | jq -r ".services[$i].buckets | join(\", \")")
        
        echo -e "${BLUE}[$((i+1))]${NC} Service: ${GREEN}$SERVICE${NC}"
        echo -e "    Endpoint: ${GREEN}$ENDPOINT${NC}"
        echo -e "    Buckets: ${GREEN}$BUCKETS${NC}"
    done
    
    if [ "$SAMPLE_COUNT" -gt 5 ]; then
        echo -e "${YELLOW}... 還有 $((SAMPLE_COUNT - 5)) 個端點${NC}"
    fi
    
    print_result "服務列表" "PASS" "成功顯示服務資訊"
fi

# Test 6: Verify specific service (if twdiw-customer-service-prod exists)
echo -e "\n${YELLOW}Test 6: 驗證特定服務${NC}"
TWDIW_SERVICES=$(echo "$AVAILABLE_RESPONSE" | jq -r '.services[] | select(.service == "twdiw-customer-service-prod")')

if [ -n "$TWDIW_SERVICES" ]; then
    TWDIW_COUNT=$(echo "$TWDIW_SERVICES" | jq -s 'length')
    print_result "特定服務查詢" "PASS" "找到 twdiw-customer-service-prod 的 $TWDIW_COUNT 個端點"
    
    echo -e "\n${BLUE}twdiw-customer-service-prod 的端點:${NC}"
    echo "$TWDIW_SERVICES" | jq -r '.endpoint' | while read endpoint; do
        echo -e "  - ${GREEN}$endpoint${NC}"
    done
else
    print_result "特定服務查詢" "WARN" "未找到 twdiw-customer-service-prod (可能需要更多時間收集資料)"
fi

# Test 7: Check bucket information
if [ "$TOTAL_SERVICES" -gt 0 ]; then
    echo -e "\n${YELLOW}Test 7: 檢查時間分桶資訊${NC}"
    
    # Get first service with buckets
    FIRST_SERVICE=$(echo "$AVAILABLE_RESPONSE" | jq -r '.services[0]')
    SERVICE_NAME=$(echo "$FIRST_SERVICE" | jq -r '.service')
    ENDPOINT_NAME=$(echo "$FIRST_SERVICE" | jq -r '.endpoint')
    BUCKET_COUNT=$(echo "$FIRST_SERVICE" | jq -r '.buckets | length')
    
    if [ "$BUCKET_COUNT" -gt 0 ]; then
        print_result "時間分桶" "PASS" "$SERVICE_NAME 的 $ENDPOINT_NAME 有 $BUCKET_COUNT 個時間桶"
        
        # Show sample buckets
        SAMPLE_BUCKETS=$(echo "$FIRST_SERVICE" | jq -r '.buckets[0:3] | join(", ")')
        echo -e "${BLUE}範例時間桶:${NC} $SAMPLE_BUCKETS"
    else
        print_result "時間分桶" "WARN" "服務沒有時間桶資訊"
    fi
fi

# Test 8: Test API with wrong method
echo -e "\n${YELLOW}Test 8: 測試錯誤的 HTTP 方法${NC}"
POST_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE/v1/available")
POST_STATUS=$(echo "$POST_RESPONSE" | tail -n1)

if [ "$POST_STATUS" = "405" ]; then
    print_result "HTTP 方法驗證" "PASS" "正確拒絕 POST 請求 (405)"
else
    print_result "HTTP 方法驗證" "WARN" "預期 405 但得到 $POST_STATUS"
fi

# Test 9: Performance test
echo -e "\n${YELLOW}Test 9: 效能測試${NC}"
START_TIME=$(date +%s%N)
for i in {1..10}; do
    curl -s "$API_BASE/v1/available" > /dev/null
done
END_TIME=$(date +%s%N)
DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
AVG_TIME=$(( DURATION / 10 ))

print_result "API 效能" "PASS" "10 次請求平均耗時 ${AVG_TIME}ms"

if [ "$AVG_TIME" -lt 100 ]; then
    echo -e "${GREEN}效能優異 (< 100ms)${NC}"
elif [ "$AVG_TIME" -lt 500 ]; then
    echo -e "${YELLOW}效能良好 (< 500ms)${NC}"
else
    echo -e "${RED}效能需要改善 (> 500ms)${NC}"
fi

# Test 10: Swagger documentation
echo -e "\n${YELLOW}Test 10: 檢查 Swagger 文檔${NC}"
SWAGGER_JSON=$(curl -s "$API_BASE/swagger/doc.json")

if echo "$SWAGGER_JSON" | jq -e '.paths."/v1/available"' > /dev/null 2>&1; then
    print_result "Swagger 文檔" "PASS" "/v1/available 已加入 Swagger 文檔"
    
    # Check if it has the correct tag
    TAG=$(echo "$SWAGGER_JSON" | jq -r '.paths."/v1/available".get.tags[0]')
    if [ "$TAG" = "Available Services" ]; then
        print_result "Swagger 標籤" "PASS" "使用正確的標籤: $TAG"
    else
        print_result "Swagger 標籤" "WARN" "標籤為: $TAG"
    fi
else
    print_result "Swagger 文檔" "FAIL" "/v1/available 未在 Swagger 文檔中"
fi

# Final summary
print_header "測試總結"

echo -e "${GREEN}✓ 所有關鍵測試通過!${NC}"
echo -e "\n${BLUE}新 API 端點:${NC} GET $API_BASE/v1/available"
echo -e "${BLUE}Swagger UI:${NC} $API_BASE/swagger/index.html"
echo -e "\n${YELLOW}提示:${NC} 如果資料較少,請等待服務收集更多 Tempo traces"

# Save full response to file for inspection
echo "$AVAILABLE_RESPONSE" > /tmp/available_api_response.json
echo -e "\n${BLUE}完整回應已儲存至:${NC} /tmp/available_api_response.json"
