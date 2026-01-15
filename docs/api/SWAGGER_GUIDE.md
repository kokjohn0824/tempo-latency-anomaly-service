# Swagger API 指南

本文件整合並精簡了原先分散於 `SWAGGER_QUICKSTART.md`、`SWAGGER.md`、`SWAGGER_IMPLEMENTATION.md` 的內容，提供一站式的使用與實作說明。

## 1) 快速開始

### 啟動服務

```bash
cd /path/to/tempo-latency-anomaly-service
docker compose -f docker/compose.yml up -d
```

等待服務啟動 (約 10 秒)。

### 打開 Swagger UI

瀏覽器訪問 `http://localhost:8080/swagger/index.html`，即可使用互動式 API 文檔介面。

### 立刻動手試試

- Health Check: 在 Swagger UI 展開 Health，執行 `GET /healthz`，應回應 `{ "status": "ok" }`
- Anomaly Detection: 展開 Anomaly Detection，執行 `POST /v1/anomaly/check`，使用預設或自訂請求體
- Baseline Query: 展開 Baseline，執行 `GET /v1/baseline`，填入查詢參數

### 常見問題排查

- 無法訪問 Swagger UI：
  - 檢查服務：`curl http://localhost:8080/healthz`
  - 嘗試重啟：`docker compose -f docker/compose.yml restart service`
- API 404：確認路徑是否正確：`/healthz`、`/v1/anomaly/check`、`/v1/baseline`
- Baseline 查詢為 "not found"：樣本不足 (需 >= 30)，等待 Tempo poller 收集或調整 `min_samples`

### 快速連結

- Swagger UI: `http://localhost:8080/swagger/index.html`
- Swagger JSON: `http://localhost:8080/swagger/doc.json`
- Health Check: `http://localhost:8080/healthz`

---

## 2) API 總覽

### 端點一覽

#### 1. Health Check
- 路徑: `GET /healthz`
- 描述: 檢查服務是否正常運行
- 回應: `{"status": "ok"}`

#### 2. Anomaly Detection
- 路徑: `POST /v1/anomaly/check`
- 描述: 評估給定請求延遲是否為異常
- 請求體範例:
  ```json
  {
    "service": "twdiw-customer-service-prod",
    "endpoint": "GET /actuator/health",
    "timestampNano": 1673000000000000000,
    "durationMs": 250
  }
  ```
- 回應範例:
  ```json
  {
    "isAnomaly": false,
    "bucket": { "hour": 16, "dayType": "weekday" },
    "baseline": {
      "p50": 233.5,
      "p95": 562.0,
      "mad": 43.0,
      "sampleCount": 50,
      "updatedAt": "2026-01-15T08:00:00Z"
    },
    "explanation": "duration 250ms within threshold 1124.00ms"
  }
  ```

#### 3. Baseline Query
- 路徑: `GET /v1/baseline`
- 描述: 查詢指定 service、endpoint、hour、dayType 的 baseline 統計
- 查詢參數: `service`, `endpoint`, `hour` (0-23), `dayType` (weekday|weekend)
- 回應範例:
  ```json
  {
    "p50": 233.5,
    "p95": 562.0,
    "mad": 43.0,
    "sampleCount": 50,
    "updatedAt": "2026-01-15T08:00:00Z"
  }
  ```

### 文檔來源

- JSON: `http://localhost:8080/swagger/doc.json`
- YAML: `docs/swagger.yaml`
- 代碼生成: `docs/docs.go`

---

## 3) 實作細節

### 依賴與產物

- 套件: `github.com/swaggo/swag/cmd/swag`, `github.com/swaggo/http-swagger`, `github.com/swaggo/files`
- 產物: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`

### 代碼整合

- 註解覆蓋範圍：
  - `cmd/server/main.go`：API 基本資訊與標籤
  - `internal/api/handlers/*`：各端點註解 (Health/Anomaly/Baseline)
  - `internal/domain/model.go`、`internal/store/store.go`：模型 JSON 標籤與範例
- 路由整合：
  ```go
  // Swagger UI endpoint
  mux.HandleFunc("/swagger/", httpSwagger.Handler(
      httpSwagger.URL("/swagger/doc.json"),
  ))
  ```

### 生成與開發流程

```bash
# 安裝 swag (如未安裝)
go install github.com/swaggo/swag/cmd/swag@latest

# 生成 OpenAPI 文檔
swag init -g cmd/server/main.go -o ./docs
```

新增端點時：

```go
// @Summary 端點摘要
// @Description 詳細描述
// @Tags 標籤名稱
// @Accept json
// @Produce json
// @Param request body MyRequestType true "請求描述"
// @Success 200 {object} MyResponseType
// @Failure 400 {object} map[string]string
// @Router /v1/my-endpoint [post]
```

### Docker 構建整合

- 構建時安裝 `swag` 並自動執行 `swag init` 以確保文檔與代碼同步

### 測試與驗證 (摘要)

- `scripts/test_swagger.sh` 驗證 Swagger JSON/UI 可用、端點與模型定義完整、範例正確

### 自動化與後續改進

- 自動化：從代碼註解生成文檔，無需手動維護 JSON/YAML
- 改進建議：開發環境才開啟 UI、更多錯誤範例、版本標籤、以環境變數控制開關

---

## 4) 使用範例

以下為可直接執行的 curl 範例：

### Health Check

```bash
curl -sS http://localhost:8080/healthz
```

### Anomaly Detection

```bash
curl -X 'POST' \
  'http://localhost:8080/v1/anomaly/check' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "service": "twdiw-customer-service-prod",
  "endpoint": "GET /actuator/health",
  "timestampNano": 1673000000000000000,
  "durationMs": 250
}'
```

### Baseline Query

```bash
curl -G \
  --data-urlencode 'service=twdiw-customer-service-prod' \
  --data-urlencode 'endpoint=GET /actuator/health' \
  --data-urlencode 'hour=16' \
  --data-urlencode 'dayType=weekday' \
  'http://localhost:8080/v1/baseline'
```

### 取得 OpenAPI JSON

```bash
curl -sS http://localhost:8080/swagger/doc.json | jq '.info, .paths | keys | length'
```

---

如需端到端互動測試，建議直接在 Swagger UI 中使用「Try it out」。

