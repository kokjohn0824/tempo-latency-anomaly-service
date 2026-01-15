# Tempo Latency Anomaly Service - 任務規劃

## 專案概述
建立一個基於 Grafana Tempo 的時間感知 API 延遲異常檢測服務,具備以下特性:
- 可解釋性(非黑盒 ML)
- 時間感知(特定時段的延遲可能是正常的)
- 低延遲檢查路徑
- 部署後全自動運行

---

## 階段性任務分解

### 第一階段:基礎設施建立 (Tasks 1-4)

#### Task 1: 專案初始化 ✅
**目標**: 建立 Go module 和完整目錄結構
**產出**:
- `go.mod` 初始化
- 完整目錄結構符合 task.md 規範
- 空的檔案佔位符

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 1: 初始化 Go 專案結構,建立所有目錄和空檔案佔位符" --full-auto
```

---

#### Task 2: 配置管理 ✅
**目標**: 實作配置載入和預設值
**產出**:
- `internal/config/config.go` - YAML/ENV 載入
- `internal/config/defaults.go` - 預設參數
- `configs/config.example.yaml` - 範例配置
- `configs/config.dev.yaml` - 開發環境配置

**配置項目**:
- Redis 連線設定
- Tempo 端點和認證
- 時區設定 (Asia/Taipei)
- 統計參數 (factor=2.0, k=10, minSamples=50)
- 輪詢間隔 (tempoInterval=15s, baselineInterval=30s)
- 滾動窗口大小 (windowSize=1000)

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 2: 實作 config 模組,支援 YAML 和環境變數" --full-auto
```

---

#### Task 3: Redis 儲存層 ✅
**目標**: 實作所有 Redis 操作
**產出**:
- `internal/store/redis/client.go` - Redis 連線和初始化
- `internal/store/redis/durations.go` - 滾動樣本操作 (LPUSH/LTRIM/LRANGE)
- `internal/store/redis/baseline.go` - Baseline 快取 (HGETALL/HSET)
- `internal/store/redis/dedup.go` - TraceID 去重 (SET with TTL)
- `internal/store/redis/dirty.go` - Dirty keys 追蹤 (SADD/SPOP)
- `internal/store/store.go` - 介面定義

**Key Schema**:
- `dur:{service}|{endpoint}|{hour}|{dayType}` → LIST
- `base:{service}|{endpoint}|{hour}|{dayType}` → HASH
- `seen:{traceID}` → STRING with TTL
- `dirtyKeys` → SET

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 3: 實作 Redis 儲存層所有操作" --full-auto
```

---

#### Task 4: Domain 層 ✅
**目標**: 定義核心資料模型
**產出**:
- `internal/domain/model.go` - 資料結構
  - `TraceEvent` - Tempo trace 資料
  - `TimeBucket` - 時間桶 (hour, dayType)
  - `BaselineStats` - 統計數據 (p50, p95, mad, count)
  - `AnomalyCheckRequest` - 檢查請求
  - `AnomalyCheckResponse` - 檢查回應
- `internal/domain/key.go` - Key 生成和時間轉換
  - 將 unixNano 轉為 Asia/Taipei 時區
  - 判斷 weekday/weekend
  - 生成 baseline key

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 4: 實作 domain 模型和 key 生成邏輯" --full-auto
```

---

### 第二階段:核心運算邏輯 (Tasks 5-7)

#### Task 5: 統計計算 ✅
**目標**: 實作統計演算法
**產出**:
- `internal/stats/percentile.go` - P50/P95 計算
- `internal/stats/mad.go` - MAD (Median Absolute Deviation) 計算
- `internal/stats/calculator.go` - 整合計算器

**演算法要求**:
- 使用快速選擇或排序計算百分位數
- MAD 公式: `median(|x - p50|)`
- 處理邊界情況 (樣本不足、全為零等)

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 5: 實作統計計算模組 (p50/p95/MAD)" --full-auto
```

---

#### Task 6: Tempo 客戶端 ✅
**目標**: 實作 Tempo API 整合
**產出**:
- `internal/tempo/client.go` - HTTP 客戶端、重試機制
- `internal/tempo/query.go` - 查詢參數建構
- `internal/tempo/types.go` - Tempo 回應結構體

**功能**:
- 查詢最近 N 秒的 traces
- 解析 Tempo API 回應
- 錯誤處理和重試
- 認證 header 支援

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 6: 實作 Tempo HTTP 客戶端和查詢邏輯" --full-auto
```

---

#### Task 7: Service 層 ✅
**目標**: 實作核心業務邏輯
**產出**:
- `internal/service/ingest.go` - Trace 攝取邏輯
  - 解析 trace
  - 去重檢查
  - 儲存 duration
  - 標記 dirty
- `internal/service/check.go` - 異常檢測邏輯
  - 取得 baseline
  - 應用規則: `durationMs > max(p95 * factor, p50 + k * MAD)`
  - 生成可解釋的原因
- `internal/service/baseline.go` - Baseline 重新計算
  - 從 Redis 取得樣本
  - 呼叫統計計算器
  - 儲存結果
  - 處理樣本不足情況

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 7: 實作 service 層業務邏輯" --full-auto
```

---

### 第三階段:API 和介面 (Tasks 8-9)

#### Task 8: API Handlers ✅
**目標**: 實作 HTTP API
**產出**:
- `internal/api/router.go` - Chi router 設定
- `internal/api/middleware.go` - 中介層 (logging, requestID, recover)
- `internal/api/handlers/healthz.go` - 健康檢查
- `internal/api/handlers/check.go` - `POST /v1/anomaly/check`
- `internal/api/handlers/baseline.go` - `GET /v1/baseline`

**API 規格**:
1. `POST /v1/anomaly/check`
   - Input: `{rootServiceName, rootTraceName, startTimeUnixNano, durationMs}`
   - Output: `{isAnomaly, bucket, baseline, reason}`
   
2. `GET /v1/baseline?service=X&endpoint=Y&hour=H&dayType=D`
   - 返回特定 key 的 baseline 資訊

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 8: 實作 HTTP API handlers 和路由" --full-auto
```

---

#### Task 9: 背景任務 ✅
**目標**: 實作自動化背景 jobs
**產出**:
- `internal/jobs/tempo_poller.go` - Tempo 輪詢器
  - 每 15 秒執行
  - 回查 120 秒
  - 呼叫 ingest service
- `internal/jobs/baseline_recompute.go` - Baseline 重算器
  - 每 30 秒執行
  - 處理 dirty keys
  - 批次處理避免阻塞

**並發控制**:
- 使用 context 控制生命週期
- Graceful shutdown 支援
- 錯誤恢復機制

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 9: 實作背景任務輪詢和重算邏輯" --full-auto
```

---

### 第四階段:應用程式整合 (Tasks 10-12)

#### Task 10: 應用程式層 ✅
**目標**: 整合所有元件
**產出**:
- `internal/app/app.go` - 依賴注入和 wiring
- `internal/app/lifecycle.go` - 啟動/關閉邏輯
- `internal/observability/logger.go` - 日誌設定
- `internal/observability/metrics.go` - Prometheus metrics
- `cmd/server/main.go` - 主程式入口

**整合內容**:
- 載入配置
- 初始化 Redis
- 初始化 Tempo 客戶端
- 建立 services
- 啟動 HTTP server
- 啟動背景 jobs
- 信號處理和 graceful shutdown

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 10: 實作應用程式層 wiring 和 lifecycle" --full-auto
```

---

#### Task 11: Docker 和部署 ✅
**目標**: 容器化和編排
**產出**:
- `docker/Dockerfile` - Multi-stage build
- `docker/compose.yml` - 服務編排
  - tempo-anomaly-service
  - redis
  - (可選) grafana/tempo for testing
- `scripts/dev.sh` - 開發輔助腳本

**Docker 設定**:
- 使用官方 golang:1.21 Alpine
- 多階段建構優化大小
- 健康檢查配置
- Volume 掛載配置檔

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 11: 建立 Dockerfile 和 docker-compose 配置" --full-auto
```

---

#### Task 12: 文件和測試資料 ✅
**目標**: 完成專案文件
**產出**:
- `README.md` - 完整專案說明
  - 架構圖
  - 快速開始
  - API 文件
  - 配置說明
- `testdata/tempo_response.json` - 測試用 Tempo 回應
- `ARCHITECTURE.md` - 架構設計文件

**README 包含**:
- 專案簡介
- 設計原則
- 快速啟動指南
- API 使用範例
- 配置參數說明
- 故障排除

**Codex 指令**:
```bash
export TERM=xterm && codex exec "Task 12: 完成 README 和測試資料" --full-auto
```

---

## 執行策略

### 每個 Task 的執行流程:
1. **開啟新的 Task Tool (agent)**
2. **在專案目錄執行 Codex 指令**
3. **驗證產出**
4. **更新 TODO 狀態**
5. **繼續下一個 Task**

### Codex 執行模板:
```bash
export TERM=xterm && codex exec "continue to next task" --full-auto
```

---

## 關鍵設計決策記錄

### 1. 時間桶設計
- **時區**: Asia/Taipei
- **維度**: hourOfDay (0-23) + dayType (weekday/weekend)
- **原因**: 平衡粒度和樣本數量

### 2. 異常檢測公式
```
durationMs > max(p95 * 2.0, p50 + 10 * MAD)
```
- **Hybrid approach**: 結合相對閾值和絕對偏差
- **可調參數**: factor=2.0, k=10
- **最小樣本**: 50

### 3. Storage Schema
- **Rolling Window**: Redis LIST, 最多 1000 筆
- **Baseline Cache**: Redis HASH, 快速讀取
- **Dirty Tracking**: Redis SET, 避免重複計算

### 4. 效能考量
- ✅ 檢查 API 必須是 O(1) - 只讀取 cache
- ✅ 計算在背景執行 - 不阻塞 API
- ✅ Dirty tracking - 只重算需要更新的 key
- ✅ Deduplication - 避免重複攝取

---

## 驗證清單

每個 Task 完成後檢查:
- [ ] 程式碼符合 Go 慣例
- [ ] 錯誤處理完善
- [ ] 日誌輸出適當
- [ ] 配置可調整
- [ ] 無阻塞操作在 critical path
- [ ] Context 正確傳遞
- [ ] Resource cleanup 正確

---

## 下一步行動

**立即執行**: Task 1 - 專案初始化

使用以下指令開始:
```bash
export TERM=xterm && codex exec "Task 1: 初始化 Go 專案結構,建立所有目錄和空檔案佔位符" --full-auto
```
