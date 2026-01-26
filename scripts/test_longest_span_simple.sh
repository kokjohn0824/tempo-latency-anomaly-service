#!/bin/bash
# 簡化版 longest-span API 測試
# 用法: ./test_longest_span_simple.sh [trace_id]

set -e

# 設定顏色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TEMPO_URL="${TEMPO_URL:-http://localhost:3200}"
ANOMALY_SERVICE_URL="${ANOMALY_SERVICE_URL:-http://localhost:8081}"
TRACE_DEMO_URL="${TRACE_DEMO_URL:-http://localhost:8080}"

# 如果沒有提供 trace_id，則產生一個新的
if [ -z "$1" ]; then
    echo "未提供 trace_id，將產生新的 trace..."
    echo ""
    
    # 使用 simulate API 產生一個有層級結構的 trace
    # depth=3, breadth=2: 會產生多層 spans
    response=$(curl -s "$TRACE_DEMO_URL/api/simulate?depth=3&breadth=2&duration=100&variance=0.5")
    trace_id=$(echo "$response" | jq -r '.traceId // .trace_id // empty')
    
    if [ -z "$trace_id" ]; then
        echo -e "${RED}❌ 無法產生 trace${NC}"
        echo "Response: $response"
        exit 1
    fi
    
    span_count=$(echo "$response" | jq -r '.spanCount // 0')
    echo -e "${GREEN}✓ 產生 Trace ID: $trace_id${NC}"
    echo "  Span 數量: $span_count"
    echo "  等待 3 秒讓 trace 寫入 Tempo..."
    sleep 3
else
    trace_id=$1
fi

echo ""
echo "========================================="
echo "測試 Trace ID: $trace_id"
echo "========================================="
echo ""

# 從 Tempo 查詢
echo "步驟 1: 查詢 Tempo..."
tempo_response=$(curl -s "$TEMPO_URL/api/traces/$trace_id")

if ! echo "$tempo_response" | jq -e '.resourceSpans // .batches' > /dev/null 2>&1; then
    echo -e "${RED}❌ Tempo 查詢失敗或 trace 不存在${NC}"
    echo "Tempo URL: $TEMPO_URL/api/traces/$trace_id"
    exit 1
fi

echo -e "${GREEN}✓ Tempo 查詢成功${NC}"
echo ""

# 解析並顯示所有 spans (支持 resourceSpans 和 batches 兩種格式)
echo "--- Trace 中的所有 Spans ---"
spans_info=$(echo "$tempo_response" | jq -r '
    (.resourceSpans // .batches)[] as $rs |
    ($rs.resource.attributes[] | select(.key == "service.name") | .value.stringValue) as $service |
    $rs.scopeSpans[].spans[] |
    {
        spanId: .spanId,
        name: .name,
        service: $service,
        parentSpanId: (.parentSpanId // "ROOT"),
        duration: (((.endTimeUnixNano | tonumber) - (.startTimeUnixNano | tonumber)) / 1000000)
    }
' | jq -s .)

echo "$spans_info" | jq -r '.[] | 
    "  SpanID: \(.spanId[0:12])... | Parent: \(.parentSpanId[0:12] // "ROOT")... | \(.duration)ms | \(.name)"
'

total_spans=$(echo "$spans_info" | jq 'length')
echo ""
echo "總共 $total_spans 個 spans"
echo ""

# 找出 root span
root_span=$(echo "$spans_info" | jq '.[] | select(.parentSpanId == "ROOT") | {spanId, name, duration}' | jq -s '.[0]')
root_id=$(echo "$root_span" | jq -r '.spanId')
root_duration=$(echo "$root_span" | jq -r '.duration')

echo -e "${YELLOW}Root Span:${NC}"
echo "  SpanID: ${root_id:0:12}..."
echo "  Name: $(echo "$root_span" | jq -r '.name')"
echo "  Duration: ${root_duration}ms"
echo ""

# 找出 duration 最長的 span
longest_span=$(echo "$spans_info" | jq 'max_by(.duration)')
longest_id=$(echo "$longest_span" | jq -r '.spanId')
longest_duration=$(echo "$longest_span" | jq -r '.duration')

echo -e "${YELLOW}Duration 最長的 Span:${NC}"
echo "  SpanID: ${longest_id:0:12}..."
echo "  Name: $(echo "$longest_span" | jq -r '.name')"
echo "  Duration: ${longest_duration}ms"
echo "  Parent: $(echo "$longest_span" | jq -r '.parentSpanId[0:12] // "ROOT"')..."
echo ""

# 找出最長的 leaf span
all_span_ids=$(echo "$spans_info" | jq -r '.[].spanId')
parent_ids=$(echo "$spans_info" | jq -r '.[] | select(.parentSpanId != "ROOT") | .parentSpanId')
leaf_span_ids=$(comm -23 <(echo "$all_span_ids" | sort) <(echo "$parent_ids" | sort))
leaf_spans=$(echo "$spans_info" | jq --argjson leaf_ids "$(echo "$leaf_span_ids" | jq -R . | jq -s .)" '.[] | select([.spanId] | inside($leaf_ids))')
longest_leaf=$(echo "$leaf_spans" | jq -s 'max_by(.duration)')

if [ "$longest_leaf" != "null" ]; then
    longest_leaf_id=$(echo "$longest_leaf" | jq -r '.spanId')
    longest_leaf_duration=$(echo "$longest_leaf" | jq -r '.duration')
    
    echo -e "${YELLOW}最長的 Leaf Span (沒有 children):${NC}"
    echo "  SpanID: ${longest_leaf_id:0:12}..."
    echo "  Name: $(echo "$longest_leaf" | jq -r '.name')"
    echo "  Duration: ${longest_leaf_duration}ms"
    echo ""
fi

# 判斷
echo "--- 分析結果 ---"
if [ "$longest_id" = "$root_id" ]; then
    echo -e "${RED}⚠️  最長 span 就是 root span${NC}"
    echo -e "${RED}   這證明了: parent span 通常是最長的${NC}"
else
    echo -e "${GREEN}✓ 最長 span 不是 root span${NC}"
fi
echo ""

# 測試 API
echo "步驟 2: 測試 longest-span API..."
api_response=$(curl -s "$ANOMALY_SERVICE_URL/v1/traces/$trace_id/longest-span")

if echo "$api_response" | jq -e '.longestSpan' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API 呼叫成功${NC}"
    echo ""
    
    api_span_id=$(echo "$api_response" | jq -r '.longestSpan.spanId')
    api_span_name=$(echo "$api_response" | jq -r '.longestSpan.name')
    api_duration=$(echo "$api_response" | jq -r '.longestSpan.durationMs')
    api_parent=$(echo "$api_response" | jq -r '.longestSpan.parentSpanId // "ROOT"')
    
    echo -e "${YELLOW}API 回傳的 Span:${NC}"
    echo "  SpanID: ${api_span_id:0:12}..."
    echo "  Name: $api_span_name"
    echo "  Duration: ${api_duration}ms"
    echo "  Parent: ${api_parent:0:12}..."
    echo ""
    
    # 驗證
    echo "--- 驗證結果 ---"
    if [ "$api_span_id" = "$longest_id" ]; then
        echo -e "${GREEN}✓ API 回傳的確實是 duration 最長的 span${NC}"
    else
        echo -e "${RED}❌ API 回傳的不是 duration 最長的 span${NC}"
    fi
    
    if [ "$api_span_id" = "$root_id" ]; then
        echo -e "${RED}⚠️  API 回傳的是 root span${NC}"
        echo -e "${RED}   問題: 在實際應用中，這個資訊價值有限${NC}"
        
        if [ "$longest_leaf" != "null" ]; then
            echo ""
            echo -e "${YELLOW}建議: 應該回傳最長的 leaf span:${NC}"
            echo "  Name: $(echo "$longest_leaf" | jq -r '.name')"
            echo "  Duration: ${longest_leaf_duration}ms"
            echo "  這才是真正的性能瓶頸點"
        fi
    else
        echo -e "${GREEN}✓ API 回傳的不是 root span${NC}"
    fi
    
else
    echo -e "${RED}❌ API 呼叫失敗${NC}"
    echo "Response: $api_response"
fi

echo ""
echo "========================================="
echo -e "${GREEN}測試完成${NC}"
echo "========================================="
