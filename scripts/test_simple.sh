#!/bin/bash
# 簡化版測試腳本

BASE_URL="http://localhost:8080"

echo "========================================="
echo "Tempo Latency Anomaly Service 測試"
echo "========================================="
echo ""

# 測試 1: 健康檢查
echo "Test 1: 健康檢查"
curl -s "$BASE_URL/healthz" | jq .
echo ""

# 測試 2: 檢查 Redis 資料
echo "Test 2: Redis 資料統計"
echo "Duration keys:"
docker exec tempo-anomaly-redis redis-cli KEYS "dur:*" | wc -l
echo "Baseline keys:"
docker exec tempo-anomaly-redis redis-cli KEYS "base:*" | wc -l
echo "Dirty keys:"
docker exec tempo-anomaly-redis redis-cli SCARD dirtyKeys
echo ""

# 測試 3: 查詢一個 baseline
echo "Test 3: 查詢 Baseline"
sample_key=$(docker exec tempo-anomaly-redis redis-cli KEYS "base:*" | head -1)
echo "Sample key: $sample_key"

if [ -n "$sample_key" ]; then
    service=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f1)
    endpoint=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f2)
    hour=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f3)
    dayType=$(echo "$sample_key" | cut -d: -f2 | cut -d'|' -f4)
    
    echo "Service: $service"
    echo "Endpoint: $endpoint"
    echo "Hour: $hour"
    echo "DayType: $dayType"
    echo ""
    
    # URL encode
    service_enc=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$service'))")
    endpoint_enc=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$endpoint'))")
    
    echo "Querying baseline..."
    curl -s "$BASE_URL/v1/baseline?service=$service_enc&endpoint=$endpoint_enc&hour=$hour&dayType=$dayType" | jq .
    echo ""
    
    # 測試 4: 異常檢測 - 正常請求
    echo "Test 4: 異常檢測 - 正常延遲"
    baseline_data=$(docker exec tempo-anomaly-redis redis-cli HGETALL "$sample_key")
    p50=$(echo "$baseline_data" | grep -A1 "^p50$" | tail -1)
    
    if [ -n "$p50" ] && [ "$p50" != "0" ]; then
        echo "使用 P50: ${p50}ms"
        timestamp=$(date +%s)000000000
        
        curl -s -X POST "$BASE_URL/v1/anomaly/check" \
            -H "Content-Type: application/json" \
            -d "{\"service\":\"$service\",\"endpoint\":\"$endpoint\",\"timestampNano\":$timestamp,\"durationMs\":$p50}" | jq .
        echo ""
    fi
    
    # 測試 5: 異常檢測 - 高延遲
    echo "Test 5: 異常檢測 - 高延遲 (異常)"
    p95=$(echo "$baseline_data" | grep -A1 "^p95$" | tail -1)
    mad=$(echo "$baseline_data" | grep -A1 "^mad$" | tail -1)
    
    if [ -n "$p95" ] && [ "$p95" != "0" ] && [ -n "$mad" ]; then
        # 計算異常延遲: p95 + 5*MAD
        anomaly_duration=$(python3 -c "print(int($p95 + 5 * $mad))")
        echo "P95: ${p95}ms, MAD: ${mad}ms"
        echo "測試延遲: ${anomaly_duration}ms"
        
        timestamp=$(date +%s)000000000
        curl -s -X POST "$BASE_URL/v1/anomaly/check" \
            -H "Content-Type: application/json" \
            -d "{\"service\":\"$service\",\"endpoint\":\"$endpoint\",\"timestampNano\":$timestamp,\"durationMs\":$anomaly_duration}" | jq .
        echo ""
    fi
fi

# 測試 6: 無 baseline 的新服務
echo "Test 6: 新服務 (無 baseline)"
timestamp=$(date +%s)000000000
curl -s -X POST "$BASE_URL/v1/anomaly/check" \
    -H "Content-Type: application/json" \
    -d "{\"service\":\"test-new-service\",\"endpoint\":\"/test/endpoint\",\"timestampNano\":$timestamp,\"durationMs\":100}" | jq .
echo ""

# 測試 7: Metrics
echo "Test 7: Prometheus Metrics"
curl -s "$BASE_URL/metrics" | grep "^go_" | head -5
echo ""

echo "========================================="
echo "測試完成"
echo "========================================="
