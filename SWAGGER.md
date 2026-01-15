# Swagger UI 使用指南

## 概述

本服務已整合 Swagger UI,提供互動式 API 文檔和測試介面。

## 訪問 Swagger UI

啟動服務後,可以通過以下 URL 訪問 Swagger UI:

```
http://localhost:8080/swagger/index.html
```

## API 端點

Swagger UI 提供以下 API 端點的完整文檔:

### 1. Health Check
- **路徑**: `GET /healthz`
- **描述**: 檢查服務是否正常運行
- **回應**: `{"status": "ok"}`

### 2. Anomaly Detection
- **路徑**: `POST /v1/anomaly/check`
- **描述**: 評估給定的請求延遲是否為異常
- **請求體**:
  ```json
  {
    "service": "twdiw-customer-service-prod",
    "endpoint": "GET /actuator/health",
    "timestampNano": 1673000000000000000,
    "durationMs": 250
  }
  ```
- **回應**:
  ```json
  {
    "isAnomaly": false,
    "bucket": {
      "hour": 16,
      "dayType": "weekday"
    },
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

### 3. Baseline Query
- **路徑**: `GET /v1/baseline`
- **描述**: 查詢特定 service、endpoint、hour 和 dayType 的 baseline 統計資料
- **查詢參數**:
  - `service`: 服務名稱 (例如: "twdiw-customer-service-prod")
  - `endpoint`: 端點名稱 (例如: "GET /actuator/health")
  - `hour`: 小時 (0-23)
  - `dayType`: 日期類型 ("weekday" 或 "weekend")
- **範例**:
  ```
  GET /v1/baseline?service=twdiw-customer-service-prod&endpoint=GET%20/actuator/health&hour=16&dayType=weekday
  ```
- **回應**:
  ```json
  {
    "p50": 233.5,
    "p95": 562.0,
    "mad": 43.0,
    "sampleCount": 50,
    "updatedAt": "2026-01-15T08:00:00Z"
  }
  ```

## 使用 Swagger UI 測試 API

1. **打開 Swagger UI**: 在瀏覽器中訪問 `http://localhost:8080/swagger/index.html`

2. **選擇 API 端點**: 點擊要測試的 API 端點展開詳細資訊

3. **點擊 "Try it out"**: 啟用測試模式

4. **填寫參數**: 
   - 對於 GET 請求,填寫查詢參數
   - 對於 POST 請求,編輯請求體 JSON

5. **點擊 "Execute"**: 執行 API 請求

6. **查看回應**: Swagger UI 會顯示:
   - HTTP 狀態碼
   - 回應標頭
   - 回應體
   - 請求的 curl 命令

## Swagger 文檔文件

Swagger 文檔以多種格式提供:

- **JSON 格式**: `http://localhost:8080/swagger/doc.json`
- **YAML 格式**: 位於 `docs/swagger.yaml`
- **Go 代碼**: 位於 `docs/docs.go`

## 本地開發

### 重新生成 Swagger 文檔

當修改 API 註解後,需要重新生成 Swagger 文檔:

```bash
# 安裝 swag CLI (如果尚未安裝)
go install github.com/swaggo/swag/cmd/swag@latest

# 生成 Swagger 文檔
swag init -g cmd/server/main.go -o ./docs
```

### 添加新的 API 端點

1. 在 handler 函數上方添加 Swagger 註解:

```go
// MyNewEndpoint godoc
// @Summary 端點摘要
// @Description 詳細描述
// @Tags 標籤名稱
// @Accept json
// @Produce json
// @Param request body MyRequestType true "請求描述"
// @Success 200 {object} MyResponseType
// @Failure 400 {object} map[string]string
// @Router /v1/my-endpoint [post]
func MyNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // 實作...
}
```

2. 為請求/回應類型添加 JSON 標籤和範例:

```go
type MyRequestType struct {
    Field1 string `json:"field1" example:"value1"`
    Field2 int    `json:"field2" example:"42"`
}
```

3. 重新生成 Swagger 文檔 (見上方)

4. 重新構建並重啟服務

## Docker 部署

Dockerfile 已配置為在構建時自動生成 Swagger 文檔:

```dockerfile
# 安裝 swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

# 生成 Swagger 文檔
RUN /go/bin/swag init -g cmd/server/main.go -o ./docs

# 構建應用程式
RUN go build ...
```

## 故障排除

### Swagger UI 無法訪問

1. 確認服務正在運行:
   ```bash
   curl http://localhost:8080/healthz
   ```

2. 檢查 Swagger JSON 是否可訪問:
   ```bash
   curl http://localhost:8080/swagger/doc.json
   ```

3. 查看服務日誌:
   ```bash
   docker logs tempo-anomaly-service
   ```

### Swagger 文檔未更新

1. 確認已重新生成文檔:
   ```bash
   swag init -g cmd/server/main.go -o ./docs
   ```

2. 重新構建 Docker 鏡像:
   ```bash
   docker compose -f docker/compose.yml build service
   ```

3. 重啟服務:
   ```bash
   docker compose -f docker/compose.yml restart service
   ```

## 參考資源

- [Swaggo 官方文檔](https://github.com/swaggo/swag)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)
- [OpenAPI Specification](https://swagger.io/specification/)

## 範例截圖

訪問 `http://localhost:8080/swagger/index.html` 後,你會看到:

1. **API 概覽**: 顯示所有可用的 API 端點,按標籤分組
2. **端點詳情**: 每個端點的詳細資訊,包括參數、請求體、回應格式
3. **模型定義**: 所有資料模型的結構定義
4. **互動測試**: 可以直接在瀏覽器中測試 API

## 安全性注意事項

在生產環境中,建議:

1. **限制訪問**: 使用認證中介軟體保護 Swagger UI
2. **環境變數控制**: 通過環境變數控制是否啟用 Swagger UI
3. **移除敏感資訊**: 確保 Swagger 註解中不包含敏感資訊

範例 (可選實作):

```go
// 僅在開發環境啟用 Swagger
if os.Getenv("ENABLE_SWAGGER") == "true" {
    mux.HandleFunc("/swagger/", httpSwagger.Handler(
        httpSwagger.URL("/swagger/doc.json"),
    ))
}
```
