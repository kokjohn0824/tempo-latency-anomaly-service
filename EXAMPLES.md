# API 使用範例

本文件提供 Tempo Latency Anomaly Service 的實際使用範例。

## 目錄
- [健康檢查](#健康檢查)
- [異常檢測](#異常檢測)
- [查詢 Baseline](#查詢-baseline)
- [測試情境](#測試情境)
- [Backfill 日誌範例](#backfill-日誌範例)

---

## 健康檢查

檢查服務是否正常運行。

```bash
curl http://localhost:8080/healthz
```

**回應**:
```json
{
  "status": "ok"
}
```

---

## 異常檢測

### 情境 1: 正常延遲的請求

檢測一個延遲在正常範圍內的請求。

```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "user-service",
    "endpoint": "GET /api/users",
    "timestampNano": 1768463900000000000,
    "durationMs": 50
  }'
```

**回應**:
```json
{
  "isAnomaly": false,
  "bucket": {
    "hour": 15,
    "dayType": "weekday"
  },
  "baseline": {
    "p50": 45,
    "p95": 120,
    "mad": 10,
    "sampleCount": 150,
    "updatedAt": "2026-01-15T08:00:00Z"
  },
  "explanation": "duration 50ms within threshold 180.00ms (p50=45.00, p95=120.00, MAD=10.00, factor=1.50, k=3)"
}
```

### 情境 2: 異常高延遲的請求

檢測一個延遲異常高的請求。

```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "user-service",
    "endpoint": "GET /api/users",
    "timestampNano": 1768463900000000000,
    "durationMs": 500
  }'
```

**回應**:
```json
{
  "isAnomaly": true,
  "bucket": {
    "hour": 15,
    "dayType": "weekday"
  },
  "baseline": {
    "p50": 45,
    "p95": 120,
    "mad": 10,
    "sampleCount": 150,
    "updatedAt": "2026-01-15T08:00:00Z"
  },
  "explanation": "duration 500ms exceeds threshold 180.00ms (p50=45.00, p95=120.00, MAD=10.00, factor=1.50, k=3)"
}
```

### 情境 3: 新服務 (無 Baseline)

檢測一個還沒有建立 baseline 的新服務。

```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "new-service",
    "endpoint": "POST /api/data",
    "timestampNano": 1768463900000000000,
    "durationMs": 1000
  }'
```

**回應**:
```json
{
  "isAnomaly": false,
  "bucket": {
    "hour": 15,
    "dayType": "weekday"
  },
  "explanation": "no baseline available or insufficient samples (have 0, need >= 50)"
}
```

### 情境 4: 不同時段的請求

同一個端點在不同時段可能有不同的 baseline。

**早上 9 點 (weekday)**:
```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "order-service",
    "endpoint": "POST /api/orders",
    "timestampNano": 1768420800000000000,
    "durationMs": 200
  }'
```

可能的回應 (早上流量高,baseline 較高):
```json
{
  "isAnomaly": false,
  "explanation": "duration 200ms within threshold 350.00ms (p50=180.00, p95=300.00, MAD=20.00, factor=1.50, k=3)"
}
```

**凌晨 2 點 (weekday)**:
```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "order-service",
    "endpoint": "POST /api/orders",
    "timestampNano": 1768395600000000000,
    "durationMs": 200
  }'
```

可能的回應 (凌晨流量低,baseline 較低,同樣延遲可能被判定為異常):
```json
{
  "isAnomaly": true,
  "explanation": "duration 200ms exceeds threshold 120.00ms (p50=50.00, p95=80.00, MAD=10.00, factor=1.50, k=3)"
}
```

---

## 查詢 Baseline

### 查詢特定服務端點的 Baseline

```bash
curl "http://localhost:8080/v1/baseline?service=user-service&endpoint=GET%20%2Fapi%2Fusers&hour=15&dayType=weekday"
```

**回應**:
```json
{
  "P50": 45,
  "P95": 120,
  "MAD": 10,
  "SampleCount": 150,
  "UpdatedAt": "2026-01-15T08:00:00Z"
}
```

### 查詢不存在的 Baseline

```bash
curl "http://localhost:8080/v1/baseline?service=nonexistent&endpoint=test&hour=10&dayType=weekday"
```

**回應**:
```json
{
  "error": "baseline not found"
}
```

---

## 測試情境

### 完整測試流程

以下是一個完整的測試流程,模擬真實使用情境:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"
SERVICE="payment-service"
ENDPOINT="POST /api/payments"

echo "=== 測試情境: Payment Service 延遲監控 ==="
echo ""

# 1. 檢查服務健康
echo "1. 健康檢查..."
curl -s "$BASE_URL/healthz" | jq .
echo ""

# 2. 模擬正常支付請求 (50ms)
echo "2. 正常支付請求 (50ms)..."
curl -s -X POST "$BASE_URL/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"$ENDPOINT\",
    \"timestampNano\": $(date +%s)000000000,
    \"durationMs\": 50
  }" | jq .
echo ""

# 3. 模擬慢速支付請求 (500ms)
echo "3. 慢速支付請求 (500ms)..."
curl -s -X POST "$BASE_URL/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"$ENDPOINT\",
    \"timestampNano\": $(date +%s)000000000,
    \"durationMs\": 500
  }" | jq .
echo ""

# 4. 模擬超慢支付請求 (2000ms)
echo "4. 超慢支付請求 (2000ms)..."
curl -s -X POST "$BASE_URL/v1/anomaly/check" \
  -H "Content-Type: application/json" \
  -d "{
    \"service\": \"$SERVICE\",
    \"endpoint\": \"$ENDPOINT\",
    \"timestampNano\": $(date +%s)000000000,
    \"durationMs\": 2000
  }" | jq .
echo ""

# 5. 查詢當前 baseline
echo "5. 查詢當前 baseline..."
HOUR=$(date +%H | sed 's/^0//')
DAY_TYPE="weekday"  # 簡化,實際應根據日期判斷

SERVICE_ENC=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$SERVICE'))")
ENDPOINT_ENC=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$ENDPOINT'))")

curl -s "$BASE_URL/v1/baseline?service=$SERVICE_ENC&endpoint=$ENDPOINT_ENC&hour=$HOUR&dayType=$DAY_TYPE" | jq .
echo ""

echo "=== 測試完成 ==="
```

### 批次檢測範例

檢測多個請求:

```bash
#!/bin/bash

# 批次檢測範例
requests=(
  '{"service":"api-gateway","endpoint":"GET /health","timestampNano":1768463900000000000,"durationMs":5}'
  '{"service":"api-gateway","endpoint":"GET /health","timestampNano":1768463900000000000,"durationMs":50}'
  '{"service":"user-service","endpoint":"GET /api/users","timestampNano":1768463900000000000,"durationMs":100}'
  '{"service":"user-service","endpoint":"POST /api/users","timestampNano":1768463900000000000,"durationMs":200}'
  '{"service":"order-service","endpoint":"GET /api/orders","timestampNano":1768463900000000000,"durationMs":1000}'
)

for req in "${requests[@]}"; do
  echo "檢測: $req"
  curl -s -X POST http://localhost:8080/v1/anomaly/check \
    -H "Content-Type: application/json" \
    -d "$req" | jq -c '{service:.bucket, isAnomaly:.isAnomaly, explanation:.explanation}'
  echo ""
done
```

---

## 與 Grafana 整合

### 使用 Grafana 查詢異常

可以在 Grafana 中使用 Infinity 或 JSON API 資料源來查詢異常:

```javascript
// Grafana Infinity Data Source 範例
const response = await fetch('http://tempo-anomaly-service:8080/v1/anomaly/check', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({
    service: '$service',
    endpoint: '$endpoint',
    timestampNano: Date.now() * 1000000,
    durationMs: $duration
  })
});

const data = await response.json();
return data.isAnomaly ? 1 : 0;  // 轉換為數值用於告警
```

---

## 程式化使用

### Python 範例

```python
import requests
import time

class AnomalyDetector:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def check_anomaly(self, service, endpoint, duration_ms):
        """檢測請求是否異常"""
        payload = {
            "service": service,
            "endpoint": endpoint,
            "timestampNano": int(time.time() * 1e9),
            "durationMs": duration_ms
        }
        
        response = requests.post(
            f"{self.base_url}/v1/anomaly/check",
            json=payload
        )
        response.raise_for_status()
        return response.json()
    
    def get_baseline(self, service, endpoint, hour, day_type):
        """查詢 baseline 統計"""
        params = {
            "service": service,
            "endpoint": endpoint,
            "hour": hour,
            "dayType": day_type
        }
        
        response = requests.get(
            f"{self.base_url}/v1/baseline",
            params=params
        )
        response.raise_for_status()
        return response.json()

# 使用範例
detector = AnomalyDetector()

# 檢測異常
result = detector.check_anomaly(
    service="user-service",
    endpoint="GET /api/users",
    duration_ms=150
)

if result['isAnomaly']:
    print(f"⚠️  異常檢測: {result['explanation']}")
else:
    print(f"✅ 正常: {result['explanation']}")

# 查詢 baseline
baseline = detector.get_baseline(
    service="user-service",
    endpoint="GET /api/users",
    hour=15,
    day_type="weekday"
)
print(f"P50: {baseline['P50']}ms, P95: {baseline['P95']}ms")
```

### Go 範例

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type AnomalyCheckRequest struct {
    Service       string `json:"service"`
    Endpoint      string `json:"endpoint"`
    TimestampNano int64  `json:"timestampNano"`
    DurationMs    int64  `json:"durationMs"`
}

type AnomalyCheckResponse struct {
    IsAnomaly   bool   `json:"isAnomaly"`
    Explanation string `json:"explanation"`
}

func checkAnomaly(service, endpoint string, durationMs int64) (*AnomalyCheckResponse, error) {
    req := AnomalyCheckRequest{
        Service:       service,
        Endpoint:      endpoint,
        TimestampNano: time.Now().UnixNano(),
        DurationMs:    durationMs,
    }
    
    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8080/v1/anomaly/check",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result AnomalyCheckResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

func main() {
    result, err := checkAnomaly("user-service", "GET /api/users", 150)
    if err != nil {
        panic(err)
    }
    
    if result.IsAnomaly {
        fmt.Printf("⚠️  異常: %s\n", result.Explanation)
    } else {
        fmt.Printf("✅ 正常: %s\n", result.Explanation)
    }
}
```

---

## Backfill 日誌範例

以下為服務啟動後的回填(backfill)與正常輪詢的日誌片段,可用來觀察回填進度與是否接近 Tempo 查詢上限。

```
2026/01/15 10:00:00 tempo backfill: starting 2026-01-08T10:00:00Z to 2026-01-15T09:58:00Z (batch 1h)
2026/01/15 10:00:00 tempo backfill: querying window 2026-01-08T10:00:00Z to 2026-01-08T11:00:00Z (lookback ~604800s)
2026/01/15 10:00:02 tempo backfill: received 432 traces, filtered 427, ingested 427 for 2026-01-08T10:00:00Z to 2026-01-08T11:00:00Z
2026/01/15 10:00:03 tempo backfill: querying window 2026-01-08T11:00:00Z to 2026-01-08T12:00:00Z (lookback ~604400s)
2026/01/15 10:00:05 tempo backfill: received 498 traces, filtered 492, ingested 492 for 2026-01-08T11:00:00Z to 2026-01-08T12:00:00Z
2026/01/15 10:00:05 tempo backfill WARNING: batch query results (498) close to limit (500). Consider increasing limit or reducing batch size.
...
2026/01/15 10:45:12 tempo backfill: completed

# 回填完成後,立即執行一次輪詢並進入固定間隔輪詢
2026/01/15 10:45:12 tempo poller: querying last 120 seconds
2026/01/15 10:45:12 tempo poller: received 85 traces
2026/01/15 10:45:13 tempo poller: ingested 85 traces

# 輪詢期間若接近上限會有警告
2026/01/15 10:50:27 tempo poller: querying last 120 seconds
2026/01/15 10:50:27 tempo poller: received 471 traces
2026/01/15 10:50:27 tempo poller WARNING: query results (471) close to limit (500). Consider increasing limit or reducing lookback to avoid drops.
```

提示:
- 出現 `completed` 代表回填階段結束
- 若經常出現 WARNING,可考慮「調整 `internal/tempo/client.go` 的 limit」或「縮小 `polling.backfill_batch`/`polling.tempo_lookback`」

---

## 故障排除

### 問題: 總是返回 "no baseline available"

**原因**: 系統還沒有收集足夠的樣本

**解決方案**:
1. 等待至少 5-10 分鐘讓系統收集資料
2. 檢查 Tempo 連接是否正常
3. 檢查 Redis 中是否有資料: `docker exec tempo-anomaly-redis redis-cli KEYS "base:*"`

### 問題: 正常請求被判定為異常

**原因**: 閾值設定過於嚴格

**解決方案**:
調整配置檔案中的參數:
```yaml
stats:
  factor: 2.0  # 增加 (預設 1.5)
  k: 5         # 增加 (預設 3)
```

### 問題: 異常請求未被檢測

**原因**: 閾值設定過於寬鬆

**解決方案**:
調整配置檔案中的參數:
```yaml
stats:
  factor: 1.2  # 減少 (預設 1.5)
  k: 2         # 減少 (預設 3)
```

---

**更多資訊請參考**: [README.md](./README.md) | [TEST_REPORT.md](./TEST_REPORT.md)
