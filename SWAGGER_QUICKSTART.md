# Swagger UI 快速入門

## 5 分鐘快速上手

### 步驟 1: 啟動服務

```bash
cd /path/to/tempo-latency-anomaly-service
docker compose -f docker/compose.yml up -d
```

等待服務啟動 (約 10 秒)。

### 步驟 2: 訪問 Swagger UI

在瀏覽器中打開:

```
http://localhost:8080/swagger/index.html
```

你會看到一個互動式的 API 文檔介面。

### 步驟 3: 測試 Health Check API

1. 找到 **Health** 標籤,點擊展開
2. 點擊 `GET /healthz` 端點
3. 點擊右側的 **"Try it out"** 按鈕
4. 點擊 **"Execute"** 按鈕
5. 查看回應:
   ```json
   {
     "status": "ok"
   }
   ```

### 步驟 4: 測試 Anomaly Detection API

1. 找到 **Anomaly Detection** 標籤,點擊展開
2. 點擊 `POST /v1/anomaly/check` 端點
3. 點擊 **"Try it out"** 按鈕
4. 編輯請求體 (或使用預設值):
   ```json
   {
     "service": "twdiw-customer-service-prod",
     "endpoint": "GET /actuator/health",
     "timestampNano": 1673000000000000000,
     "durationMs": 250
   }
   ```
5. 點擊 **"Execute"** 按鈕
6. 查看回應,了解是否為異常

### 步驟 5: 測試 Baseline Query API

1. 找到 **Baseline** 標籤,點擊展開
2. 點擊 `GET /v1/baseline` 端點
3. 點擊 **"Try it out"** 按鈕
4. 填寫查詢參數:
   - `service`: `twdiw-customer-service-prod`
   - `endpoint`: `GET /actuator/health`
   - `hour`: `16`
   - `dayType`: `weekday`
5. 點擊 **"Execute"** 按鈕
6. 查看 baseline 統計資料 (如果有足夠的樣本)

## 常見使用場景

### 場景 1: 檢查 API 是否正常

使用 Health Check API:
- 端點: `GET /healthz`
- 預期回應: `{"status": "ok"}`

### 場景 2: 檢測延遲異常

使用 Anomaly Detection API:
- 端點: `POST /v1/anomaly/check`
- 提供: service, endpoint, timestamp, duration
- 獲得: 是否異常 + 解釋

### 場景 3: 查看 Baseline 資料

使用 Baseline Query API:
- 端點: `GET /v1/baseline`
- 提供: service, endpoint, hour, dayType
- 獲得: P50, P95, MAD 統計資料

## Swagger UI 功能

### 1. 互動式測試
- 直接在瀏覽器中測試 API
- 無需使用 curl 或 Postman
- 即時查看請求和回應

### 2. 自動生成的 curl 命令
- 每次執行後,Swagger UI 會顯示對應的 curl 命令
- 可以複製並在終端中使用

### 3. 模型定義
- 點擊 **"Schemas"** (頁面底部) 查看所有資料模型
- 了解每個欄位的類型和範例值

### 4. 請求/回應範例
- 每個端點都有預設的請求範例
- 可以直接使用或修改

## 故障排除

### 問題 1: 無法訪問 Swagger UI

**檢查服務是否運行**:
```bash
curl http://localhost:8080/healthz
```

如果失敗,重啟服務:
```bash
docker compose -f docker/compose.yml restart service
```

### 問題 2: API 回應 404

**確認端點路徑正確**:
- Health: `/healthz`
- Anomaly Check: `/v1/anomaly/check`
- Baseline: `/v1/baseline`

### 問題 3: Baseline 查詢回應 "not found"

**原因**: 尚未收集足夠的樣本 (需要 >= 30 個樣本)

**解決方案**: 等待 Tempo poller 收集更多資料,或降低 `min_samples` 配置。

## 進階使用

### 使用 curl 測試 (從 Swagger UI 複製)

Swagger UI 執行後會顯示 curl 命令,例如:

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

### 下載 Swagger 定義

- **JSON 格式**: `http://localhost:8080/swagger/doc.json`
- **YAML 格式**: 位於 `docs/swagger.yaml`

可以導入到其他工具 (如 Postman, Insomnia)。

## 更多資訊

- 詳細使用指南: `SWAGGER.md`
- 實作細節: `SWAGGER_IMPLEMENTATION.md`
- 測試腳本: `scripts/test_swagger.sh`

## 快速連結

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Swagger JSON**: http://localhost:8080/swagger/doc.json
- **Health Check**: http://localhost:8080/healthz

---

**提示**: 建議將 Swagger UI URL 加入書籤,方便隨時訪問!
