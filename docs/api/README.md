# API 文檔總覽

本目錄提供 Tempo Latency Anomaly Service 的 API 使用與實作文檔。

## 快速入口

- Swagger UI: `http://localhost:8080/swagger/index.html`
- OpenAPI JSON: `http://localhost:8080/swagger/doc.json`
- 詳細指南: `docs/api/SWAGGER_GUIDE.md`

## 可用端點

- `GET /healthz`: 健康檢查，回應 `{"status":"ok"}`
- `POST /v1/anomaly/check`: 判斷請求延遲是否為異常，回傳 baseline 與說明
- `GET /v1/baseline`: 以 service/endpoint/hour/dayType 查詢 baseline 統計資料

## 文檔來源

- YAML: `docs/swagger.yaml`
- JSON: `docs/swagger.json`
- 代碼生成: `docs/docs.go`

如需新增端點、重新生成文檔或了解整合細節，請參考 `docs/api/SWAGGER_GUIDE.md` 的「實作細節」。

