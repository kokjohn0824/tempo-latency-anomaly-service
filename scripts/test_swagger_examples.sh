#!/bin/bash
set -e

echo "============================================"
echo "Swagger UI 範例值測試腳本"
echo "============================================"
echo ""

BASE_URL="http://localhost:8080"

# 顏色定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "1. 檢查服務健康狀態..."
HEALTH=$(curl -s ${BASE_URL}/healthz)
if echo "$HEALTH" | jq -e '.status == "ok"' > /dev/null; then
    echo -e "${GREEN}✓ 服務運行正常${NC}"
else
    echo -e "${RED}✗ 服務未運行${NC}"
    exit 1
fi
echo ""

echo "2. 等待 backfill 完成 (30秒)..."
sleep 30
echo -e "${GREEN}✓ 等待完成${NC}"
echo ""

echo "3. 測試 /v1/available API (檢查可用的服務)..."
AVAILABLE=$(curl -s ${BASE_URL}/v1/available)
TOTAL_SERVICES=$(echo "$AVAILABLE" | jq -r '.totalServices')
TOTAL_ENDPOINTS=$(echo "$AVAILABLE" | jq -r '.totalEndpoints')

echo "   總服務數: ${TOTAL_SERVICES}"
echo "   總端點數: ${TOTAL_ENDPOINTS}"

if [ "$TOTAL_SERVICES" -gt 0 ]; then
    echo -e "${GREEN}✓ 有可用的服務資料${NC}"
    echo ""
    echo "   twdiw-customer-service-prod 的端點:"
    echo "$AVAILABLE" | jq -r '.services[] | select(.service == "twdiw-customer-service-prod") | "   - \(.endpoint) [\(.buckets | join(", "))]"' | head -5
else
    echo -e "${YELLOW}⚠ 尚無可用服務資料 (backfill 可能還在進行中)${NC}"
fi
echo ""

echo "4. 測試 Swagger UI 範例請求..."
echo "   使用範例值進行異常檢測:"
echo "   - Service: twdiw-customer-service-prod"
echo "   - Endpoint: AiPromptSyncScheduler.syncAiPromptsToDify"
echo "   - Timestamp: 1737000000000000000 (2025-01-16 09:20:00 +0800)"
echo "   - Duration: 5ms"
echo ""

RESPONSE=$(curl -s -X POST ${BASE_URL}/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "twdiw-customer-service-prod",
    "endpoint": "AiPromptSyncScheduler.syncAiPromptsToDify",
    "timestampNano": 1737000000000000000,
    "durationMs": 5
  }')

echo "   回應結果:"
echo "$RESPONSE" | jq '.'
echo ""

IS_ANOMALY=$(echo "$RESPONSE" | jq -r '.isAnomaly')
BASELINE_SOURCE=$(echo "$RESPONSE" | jq -r '.baselineSource')
FALLBACK_LEVEL=$(echo "$RESPONSE" | jq -r '.fallbackLevel')
EXPLANATION=$(echo "$RESPONSE" | jq -r '.explanation')

echo "   解析:"
echo "   - 是否異常: ${IS_ANOMALY}"
echo "   - Baseline 來源: ${BASELINE_SOURCE} (Level ${FALLBACK_LEVEL})"
echo "   - 說明: ${EXPLANATION}"

if [ "$IS_ANOMALY" == "null" ]; then
    echo -e "${RED}✗ API 回應格式錯誤${NC}"
    exit 1
elif [ "$BASELINE_SOURCE" == "unavailable" ]; then
    echo -e "${YELLOW}⚠ 尚無足夠 baseline 資料 (需等待更多資料收集)${NC}"
else
    echo -e "${GREEN}✓ API 正常運作,使用 ${BASELINE_SOURCE} baseline${NC}"
fi
echo ""

echo "5. 測試異常情況 (高延遲 1000ms)..."
ANOMALY_RESPONSE=$(curl -s -X POST ${BASE_URL}/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "twdiw-customer-service-prod",
    "endpoint": "AiPromptSyncScheduler.syncAiPromptsToDify",
    "timestampNano": 1737000000000000000,
    "durationMs": 1000
  }')

ANOMALY_DETECTED=$(echo "$ANOMALY_RESPONSE" | jq -r '.isAnomaly')
ANOMALY_EXPLANATION=$(echo "$ANOMALY_RESPONSE" | jq -r '.explanation')

echo "   結果: ${ANOMALY_DETECTED}"
echo "   說明: ${ANOMALY_EXPLANATION}"

if [ "$ANOMALY_DETECTED" == "true" ]; then
    echo -e "${GREEN}✓ 成功偵測異常${NC}"
elif [ "$ANOMALY_DETECTED" == "false" ]; then
    echo -e "${YELLOW}⚠ 未偵測為異常 (可能閾值較高)${NC}"
else
    echo -e "${RED}✗ 無法判斷${NC}"
fi
echo ""

echo "6. 檢查 Swagger UI 可訪問性..."
SWAGGER_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" ${BASE_URL}/swagger/index.html)
if [ "$SWAGGER_RESPONSE" == "200" ]; then
    echo -e "${GREEN}✓ Swagger UI 可訪問: ${BASE_URL}/swagger/index.html${NC}"
else
    echo -e "${RED}✗ Swagger UI 無法訪問 (HTTP ${SWAGGER_RESPONSE})${NC}"
fi
echo ""

echo "============================================"
echo "測試完成!"
echo "============================================"
echo ""
echo "📋 範例值摘要 (基於實際測試資料):"
echo ""
echo "   服務: twdiw-customer-service-prod"
echo "   端點: AiPromptSyncScheduler.syncAiPromptsToDify"
echo "   時段: 09:00 weekday (資料量最多: 188 samples)"
echo "   延遲特性:"
echo "   - P50: ~1ms"
echo "   - P95: ~2ms"
echo "   - MAD: ~0ms (穩定)"
echo ""
echo "   其他可用端點:"
echo "   - customer_service (7 時段, 514 samples)"
echo "   - DatasetIndexingStatusScheduler.checkIndexingStatus (7 時段, 540 samples)"
echo "   - AiCategoryRetryScheduler.processCategories (5 時段, 471 samples)"
echo ""
echo "💡 使用 Swagger UI 測試:"
echo "   開啟: ${BASE_URL}/swagger/index.html"
echo "   選擇: POST /v1/anomaly/check"
echo "   點擊: Try it out"
echo "   使用預設範例值即可進行測試!"
echo ""
