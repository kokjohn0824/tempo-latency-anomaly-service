#!/bin/bash

# Test Swagger UI functionality

set -e

API_BASE="http://localhost:8080"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Swagger UI 功能測試${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to print test result
print_result() {
    local test_name=$1
    local result=$2
    local details=$3
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC} - $test_name"
    else
        echo -e "${RED}✗ FAIL${NC} - $test_name"
    fi
    
    if [ -n "$details" ]; then
        echo -e "  ${details}"
    fi
    echo ""
}

# Test 1: Swagger JSON 端點
echo -e "${BLUE}測試 1: Swagger JSON 端點${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/swagger/doc.json")
if [ "$HTTP_CODE" = "200" ]; then
    TITLE=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.info.title')
    print_result "Swagger JSON 可訪問" "PASS" "標題: $TITLE"
else
    print_result "Swagger JSON 可訪問" "FAIL" "HTTP 狀態碼: $HTTP_CODE"
fi

# Test 2: Swagger UI 頁面
echo -e "${BLUE}測試 2: Swagger UI 頁面${NC}"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/swagger/index.html")
if [ "$HTTP_CODE" = "200" ]; then
    print_result "Swagger UI 頁面可訪問" "PASS" "URL: $API_BASE/swagger/index.html"
else
    print_result "Swagger UI 頁面可訪問" "FAIL" "HTTP 狀態碼: $HTTP_CODE"
fi

# Test 3: API 端點定義
echo -e "${BLUE}測試 3: API 端點定義${NC}"
PATHS=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths | keys | length')
if [ "$PATHS" -ge 3 ]; then
    ENDPOINTS=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths | keys | join(", ")')
    print_result "API 端點定義完整" "PASS" "共 $PATHS 個端點: $ENDPOINTS"
else
    print_result "API 端點定義完整" "FAIL" "只找到 $PATHS 個端點"
fi

# Test 4: 模型定義
echo -e "${BLUE}測試 4: 資料模型定義${NC}"
MODELS=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.definitions | keys | length')
if [ "$MODELS" -ge 5 ]; then
    MODEL_LIST=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.definitions | keys | join(", ")')
    print_result "資料模型定義完整" "PASS" "共 $MODELS 個模型"
else
    print_result "資料模型定義完整" "FAIL" "只找到 $MODELS 個模型"
fi

# Test 5: Health 端點文檔
echo -e "${BLUE}測試 5: Health 端點文檔${NC}"
HEALTH_SUMMARY=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths."/healthz".get.summary')
if [ "$HEALTH_SUMMARY" != "null" ]; then
    print_result "Health 端點文檔" "PASS" "摘要: $HEALTH_SUMMARY"
else
    print_result "Health 端點文檔" "FAIL" "未找到文檔"
fi

# Test 6: Anomaly Check 端點文檔
echo -e "${BLUE}測試 6: Anomaly Check 端點文檔${NC}"
CHECK_SUMMARY=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths."/v1/anomaly/check".post.summary')
if [ "$CHECK_SUMMARY" != "null" ]; then
    print_result "Anomaly Check 端點文檔" "PASS" "摘要: $CHECK_SUMMARY"
else
    print_result "Anomaly Check 端點文檔" "FAIL" "未找到文檔"
fi

# Test 7: Baseline 端點文檔
echo -e "${BLUE}測試 7: Baseline 端點文檔${NC}"
BASELINE_SUMMARY=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths."/v1/baseline".get.summary')
if [ "$BASELINE_SUMMARY" != "null" ]; then
    print_result "Baseline 端點文檔" "PASS" "摘要: $BASELINE_SUMMARY"
else
    print_result "Baseline 端點文檔" "FAIL" "未找到文檔"
fi

# Test 8: 請求模型範例
echo -e "${BLUE}測試 8: 請求模型範例${NC}"
REQUEST_EXAMPLE=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.definitions."domain.AnomalyCheckRequest".properties.service.example')
if [ "$REQUEST_EXAMPLE" != "null" ]; then
    print_result "請求模型包含範例" "PASS" "Service 範例: $REQUEST_EXAMPLE"
else
    print_result "請求模型包含範例" "FAIL" "未找到範例"
fi

# Test 9: 回應模型範例
echo -e "${BLUE}測試 9: 回應模型範例${NC}"
RESPONSE_EXAMPLE=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.definitions."domain.AnomalyCheckResponse".properties.isAnomaly.example')
if [ "$RESPONSE_EXAMPLE" != "null" ]; then
    print_result "回應模型包含範例" "PASS" "isAnomaly 範例: $RESPONSE_EXAMPLE"
else
    print_result "回應模型包含範例" "FAIL" "未找到範例"
fi

# Test 10: API 標籤
echo -e "${BLUE}測試 10: API 標籤分組${NC}"
TAGS=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.tags | length')
if [ "$TAGS" -ge 3 ]; then
    TAG_LIST=$(curl -s "$API_BASE/swagger/doc.json" | jq -r '.tags[].name' | tr '\n' ', ')
    print_result "API 標籤分組" "PASS" "共 $TAGS 個標籤: $TAG_LIST"
else
    print_result "API 標籤分組" "FAIL" "只找到 $TAGS 個標籤"
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}測試摘要${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Swagger UI 訪問 URL:"
echo "  ${GREEN}http://localhost:8080/swagger/index.html${NC}"
echo ""
echo "Swagger JSON API:"
echo "  http://localhost:8080/swagger/doc.json"
echo ""
echo "可用的 API 端點:"
curl -s "$API_BASE/swagger/doc.json" | jq -r '.paths | keys[]' | while read -r path; do
    echo "  - $path"
done
echo ""
echo "可用的資料模型:"
curl -s "$API_BASE/swagger/doc.json" | jq -r '.definitions | keys[]' | while read -r model; do
    echo "  - $model"
done
echo ""
echo -e "${GREEN}測試完成!${NC}"
echo ""
echo "請在瀏覽器中打開以下 URL 查看完整的 Swagger UI:"
echo -e "${GREEN}http://localhost:8080/swagger/index.html${NC}"
