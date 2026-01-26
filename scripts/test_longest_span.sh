#!/bin/bash
# 測試 longest-span API 的邏輯正確性
# 驗證是否總是回傳 root span (parent span)

set -e

# 設定顏色輸出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# API 端點
TRACE_DEMO_URL="${TRACE_DEMO_URL:-http://localhost:8080}"
ANOMALY_SERVICE_URL="${ANOMALY_SERVICE_URL:-http://localhost:8081}"
TEMPO_URL="${TEMPO_URL:-http://localhost:3200}"

echo "========================================="
echo "Longest Span API 邏輯驗證測試"
echo "========================================="
echo ""
echo "TRACE_DEMO_URL: $TRACE_DEMO_URL"
echo "ANOMALY_SERVICE_URL: $ANOMALY_SERVICE_URL"
echo "TEMPO_URL: $TEMPO_URL"
echo ""

# 檢查服務是否運行
echo "檢查服務狀態..."
if ! curl -s "$TRACE_DEMO_URL/healthz" > /dev/null 2>&1; then
    echo -e "${RED}❌ Trace Demo 服務未啟動${NC}"
    echo "請先啟動: cd tempo-otlp-trace-demo && make up"
    exit 1
fi

if ! curl -s "$ANOMALY_SERVICE_URL/healthz" > /dev/null 2>&1; then
    echo -e "${RED}❌ Anomaly Service 未啟動${NC}"
    echo "請先啟動: cd tempo-latency-anomaly-service && make up"
    exit 1
fi

echo -e "${GREEN}✓ 服務運行正常${NC}"
echo ""

# 測試函數
test_longest_span() {
    local endpoint=$1
    local description=$2
    local method=${3:-POST}
    local data=${4:-'{}'}
    
    echo "========================================="
    echo -e "${BLUE}測試場景: $description${NC}"
    echo "端點: $endpoint"
    echo "========================================="
    
    # 步驟 1: 呼叫 trace demo API 產生 trace
    echo ""
    echo "步驟 1: 產生 trace..."
    if [ "$method" = "GET" ]; then
        response=$(curl -s "$TRACE_DEMO_URL$endpoint")
    else
        response=$(curl -s -X POST "$TRACE_DEMO_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    trace_id=$(echo "$response" | jq -r '.trace_id // .traceId // empty')
    
    if [ -z "$trace_id" ]; then
        echo -e "${RED}❌ 無法取得 trace ID${NC}"
        echo "Response: $response"
        return 1
    fi
    
    echo -e "${GREEN}✓ Trace ID: $trace_id${NC}"
    
    # 等待 trace 寫入 Tempo
    echo ""
    echo "步驟 2: 等待 trace 寫入 Tempo (3秒)..."
    sleep 3
    
    # 步驟 2: 從 Tempo 直接查詢該 trace 的所有 spans
    echo ""
    echo "步驟 3: 從 Tempo 查詢完整 trace..."
    tempo_response=$(curl -s "$TEMPO_URL/api/traces/$trace_id")
    
    # 檢查 Tempo 回應
    if echo "$tempo_response" | jq -e '.resourceSpans' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Tempo 查詢成功${NC}"
        
        # 解析所有 spans
        echo ""
        echo "--- Trace 中的所有 Spans ---"
        echo "$tempo_response" | jq -r '
            .resourceSpans[] as $rs |
            ($rs.resource.attributes[] | select(.key == "service.name") | .value.stringValue) as $service |
            $rs.scopeSpans[].spans[] |
            . as $span |
            {
                spanId: .spanId,
                name: .name,
                service: $service,
                parentSpanId: (.parentSpanId // "ROOT"),
                startTime: .startTimeUnixNano,
                endTime: .endTimeUnixNano,
                duration: (((.endTimeUnixNano | tonumber) - (.startTimeUnixNano | tonumber)) / 1000000)
            } |
            "SpanID: \(.spanId) | Parent: \(.parentSpanId) | Duration: \(.duration)ms | Name: \(.name) | Service: \(.service)"
        '
        
        # 計算最長的 span (不考慮 parent 關係)
        echo ""
        echo "--- 分析結果 ---"
        
        # 找出所有 spans 中 duration 最長的
        longest_all=$(echo "$tempo_response" | jq -r '
            .resourceSpans[] as $rs |
            ($rs.resource.attributes[] | select(.key == "service.name") | .value.stringValue) as $service |
            $rs.scopeSpans[].spans[] |
            {
                spanId: .spanId,
                name: .name,
                parentSpanId: (.parentSpanId // "ROOT"),
                duration: (((.endTimeUnixNano | tonumber) - (.startTimeUnixNano | tonumber)) / 1000000)
            } |
            select(.duration != null)
        ' | jq -s 'max_by(.duration)')
        
        # 找出 root span (沒有 parent 的)
        root_span=$(echo "$tempo_response" | jq -r '
            .resourceSpans[] as $rs |
            $rs.scopeSpans[].spans[] |
            select(.parentSpanId == null or .parentSpanId == "") |
            {
                spanId: .spanId,
                name: .name,
                parentSpanId: "ROOT",
                duration: (((.endTimeUnixNano | tonumber) - (.startTimeUnixNano | tonumber)) / 1000000)
            }
        ' | jq -s '.[0]')
        
        # 找出 leaf spans (沒有 children 的)
        all_parent_ids=$(echo "$tempo_response" | jq -r '
            [.resourceSpans[].scopeSpans[].spans[].parentSpanId] | unique | .[]
        ' | grep -v "^$" | sort)
        
        leaf_spans=$(echo "$tempo_response" | jq -r --argjson parents "$(echo "$all_parent_ids" | jq -R . | jq -s .)" '
            .resourceSpans[] as $rs |
            $rs.scopeSpans[].spans[] |
            select([.spanId] | inside($parents) | not) |
            {
                spanId: .spanId,
                name: .name,
                duration: (((.endTimeUnixNano | tonumber) - (.startTimeUnixNano | tonumber)) / 1000000)
            }
        ' | jq -s 'max_by(.duration)')
        
        echo -e "${YELLOW}最長 Span (所有spans):${NC}"
        echo "$longest_all" | jq .
        
        echo -e "${YELLOW}Root Span:${NC}"
        echo "$root_span" | jq .
        
        echo -e "${YELLOW}最長 Leaf Span (沒有children的span):${NC}"
        echo "$leaf_spans" | jq .
        
        # 判斷是否一致
        longest_id=$(echo "$longest_all" | jq -r '.spanId')
        root_id=$(echo "$root_span" | jq -r '.spanId')
        
        echo ""
        if [ "$longest_id" = "$root_id" ]; then
            echo -e "${RED}⚠️  問題確認: 最長 span 就是 root span!${NC}"
            echo -e "${RED}   這驗證了我們的討論 - parent span 總是最長的${NC}"
        else
            echo -e "${GREEN}✓ 最長 span 不是 root span${NC}"
        fi
        
    else
        echo -e "${RED}❌ Tempo 無此 trace 或查詢失敗${NC}"
        echo "Response: $tempo_response"
    fi
    
    # 步驟 3: 呼叫 longest-span API
    echo ""
    echo "步驟 4: 測試 longest-span API..."
    api_response=$(curl -s "$ANOMALY_SERVICE_URL/v1/traces/$trace_id/longest-span")
    
    if echo "$api_response" | jq -e '.longestSpan' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API 呼叫成功${NC}"
        echo ""
        echo "--- API 回傳的最長 Span ---"
        echo "$api_response" | jq '{
            spanId: .longestSpan.spanId,
            name: .longestSpan.name,
            service: .longestSpan.service,
            durationMs: .longestSpan.durationMs,
            parentSpanId: .longestSpan.parentSpanId
        }'
        
        api_span_id=$(echo "$api_response" | jq -r '.longestSpan.spanId')
        api_parent=$(echo "$api_response" | jq -r '.longestSpan.parentSpanId // "ROOT"')
        
        echo ""
        if [ "$api_parent" = "ROOT" ] || [ "$api_parent" = "" ]; then
            echo -e "${RED}⚠️  API 回傳的是 ROOT span${NC}"
        else
            echo -e "${GREEN}✓ API 回傳的不是 ROOT span${NC}"
        fi
        
        # 比較
        if [ "$api_span_id" = "$root_id" ]; then
            echo -e "${RED}⚠️  確認: API 回傳的就是 root span${NC}"
        fi
        
    else
        echo -e "${RED}❌ API 查詢失敗${NC}"
        echo "Response: $api_response"
    fi
    
    echo ""
    echo "========================================="
    echo ""
    sleep 1
}

# 執行多個測試場景
echo ""
echo "開始測試多個場景..."
echo ""

# 測試 1: 訂單建立 (複雜的 trace，10-12 spans)
test_longest_span "/api/order/create" "訂單建立 (複雜流程)" "POST" '{
    "user_id": "test_user_001",
    "product_id": "prod_12345",
    "quantity": 2,
    "price": 299.99
}'

# 測試 2: 使用者資料查詢 (簡單的 trace，4-5 spans)
test_longest_span "/api/user/profile?user_id=test_user_002" "使用者資料查詢 (簡單流程)" "GET"

# 測試 3: 報表生成 (耗時較長的 trace，10-12 spans)
test_longest_span "/api/report/generate" "報表生成 (長時間操作)" "POST" '{
    "report_type": "sales",
    "start_date": "2024-01-01",
    "end_date": "2024-01-31",
    "filters": ["region:TW"]
}'

# 測試 4: 搜尋功能 (中等複雜度，6-7 spans)
test_longest_span "/api/search?query=test&limit=10" "搜尋功能 (中等複雜度)" "GET"

echo ""
echo "========================================="
echo -e "${BLUE}測試總結${NC}"
echo "========================================="
echo ""
echo "從以上測試可以看出:"
echo ""
echo -e "${YELLOW}1. 在多數真實場景中，root span 確實是最長的${NC}"
echo "   因為 parent span 的時間包含了所有 child spans"
echo ""
echo -e "${YELLOW}2. 目前的 longest-span API 邏輯的問題:${NC}"
echo "   - 技術上正確: 確實找出 duration 最大的 span"
echo "   - 實用性低: 總是回傳 root span，無法找出真正的瓶頸"
echo ""
echo -e "${YELLOW}3. 建議的改進方向:${NC}"
echo "   - 只考慮 leaf spans (沒有 children 的 spans)"
echo "   - 或計算 self-time (扣除 children 的時間)"
echo "   - 或提供參數讓使用者選擇行為"
echo ""
echo "========================================="
echo -e "${GREEN}測試完成!${NC}"
echo "========================================="
