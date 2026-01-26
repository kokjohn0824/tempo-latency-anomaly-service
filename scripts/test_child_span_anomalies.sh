#!/bin/bash
# child-span-anomalies API 測試
# 用法: ./test_child_span_anomalies.sh [trace_id] [parent_span_id]
# 若未提供 trace_id，會使用 /v1/available 與 /v1/traces 取得最新 trace

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

ANOMALY_SERVICE_URL="${ANOMALY_SERVICE_URL:-http://localhost:8081}"
LOOKBACK_SECONDS="${LOOKBACK_SECONDS:-900}"

trace_id="$1"
parent_span_id="$2"

if [ -z "$trace_id" ]; then
    echo "未提供 trace_id，將從 /v1/available 與 /v1/traces 取得 trace..."
    echo ""

    available_response=$(curl -s "$ANOMALY_SERVICE_URL/v1/available")
    if ! echo "$available_response" | jq -e '.services and (.services | length > 0)' > /dev/null 2>&1; then
        echo -e "${RED}❌ 無法取得可用服務/端點${NC}"
        echo "Response: $available_response"
        exit 1
    fi

    service=$(echo "$available_response" | jq -r '.services[0].service // empty')
    endpoint=$(echo "$available_response" | jq -r '.services[0].endpoint // empty')
    if [ -z "$service" ] || [ -z "$endpoint" ]; then
        echo -e "${RED}❌ 無法解析可用服務/端點${NC}"
        echo "Response: $available_response"
        exit 1
    fi

    end_ts=$(date +%s)
    start_ts=$((end_ts - LOOKBACK_SECONDS))
    service_q=$(jq -rn --arg v "$service" '$v|@uri')
    endpoint_q=$(jq -rn --arg v "$endpoint" '$v|@uri')
    trace_response=$(curl -s "$ANOMALY_SERVICE_URL/v1/traces?service=$service_q&endpoint=$endpoint_q&start=$start_ts&end=$end_ts&limit=5")

    trace_id=$(echo "$trace_response" | jq -r '.traces | sort_by(.startTimeUnixNano | tonumber) | last | .traceID // .traceId // empty')
    if [ -z "$trace_id" ]; then
        echo -e "${RED}❌ 無法從 /v1/traces 取得 trace${NC}"
        echo "Service: $service"
        echo "Endpoint: $endpoint"
        echo "Response: $trace_response"
        exit 1
    fi

    echo -e "${GREEN}✓ 取得 Trace ID: $trace_id${NC}"
    echo "  Service: $service"
    echo "  Endpoint: $endpoint"
    echo "  查詢時間範圍: $start_ts ~ $end_ts (最近 ${LOOKBACK_SECONDS}s)"
fi

echo ""
echo "========================================="
echo "測試 Trace ID: $trace_id"
echo "========================================="
echo ""

if [ -z "$parent_span_id" ]; then
    echo -e "${YELLOW}未提供 parent_span_id，將由 API 使用 root span${NC}"
fi

echo ""
echo "步驟 1: 呼叫 child-span-anomalies API..."
payload=$(jq -n --arg traceId "$trace_id" --arg parentSpanId "$parent_span_id" '
    if $parentSpanId == "" then {traceId: $traceId} else {traceId: $traceId, parentSpanId: $parentSpanId} end
')
api_response=$(curl -s -X POST -H "Content-Type: application/json" -d "$payload" "$ANOMALY_SERVICE_URL/v1/traces/child-span-anomalies")

if ! echo "$api_response" | jq -e '.children' > /dev/null 2>&1; then
    echo -e "${RED}❌ API 呼叫失敗${NC}"
    echo "Response: $api_response"
    exit 1
fi

child_count=$(echo "$api_response" | jq -r '.childCount')
anomaly_count=$(echo "$api_response" | jq -r '.anomalyCount')

echo -e "${GREEN}✓ API 回傳成功${NC}"
echo "  Child count: $child_count"
echo "  Anomaly count: $anomaly_count"
echo ""

missing_duration=$(echo "$api_response" | jq '[.children[] | select(.span.durationMs == null)] | length')
if [ "$missing_duration" -gt 0 ]; then
    echo -e "${RED}❌ 發現沒有 durationMs 的 child span${NC}"
    exit 1
fi

echo "--- Child Spans ---"
echo "$api_response" | jq -r '.children[] |
    "  \(.span.name) | \(.span.durationMs)ms | anomaly=\(.isAnomaly) | cannotDetermine=\(.cannotDetermine) | baselineSource=\(.baselineSource)"'

echo ""
echo "========================================="
echo -e "${GREEN}測試完成${NC}"
echo "========================================="
