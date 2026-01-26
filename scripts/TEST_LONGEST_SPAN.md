# Longest Span API 測試腳本

這些測試腳本用於驗證 longest-span API 的邏輯正確性，特別是驗證我們討論的問題：**在真實場景中，API 是否總是回傳 root span (parent span)**。

## 問題背景

在分散式追蹤系統中，parent span 的 duration 通常會包含所有 child spans 的執行時間，因此：

- **Parent span duration** = 自己的邏輯時間 + 所有 child spans 的時間
- **Child span duration** = 只有自己的執行時間

這導致在大多數情況下，**root span (最上層的 parent) 總是 duration 最長的**。

目前的 `longest-span` API 實作會遍歷 trace 中的所有 spans，並回傳 duration 數值最大的那個。雖然技術上正確，但在實際應用中：

❌ **問題**: 幾乎總是回傳 root span  
❌ **影響**: 無法找出真正的性能瓶頸點  
✅ **建議**: 應該回傳 leaf span (實際執行工作的最底層 span)

## 測試腳本

### 1. `test_longest_span_simple.sh` - 簡化版測試

快速測試單個 trace，驗證 API 行為。

**用法**:

```bash
# 自動產生一個新的 trace 並測試
./scripts/test_longest_span_simple.sh

# 測試指定的 trace ID
./scripts/test_longest_span_simple.sh <trace_id>
```

**輸出內容**:
- Trace 中所有 spans 的列表
- Root span 資訊
- Duration 最長的 span
- 最長的 leaf span (沒有 children)
- API 回傳結果
- 驗證分析

**範例**:

```bash
cd tempo-latency-anomaly-service
./scripts/test_longest_span_simple.sh
```

### 2. `test_longest_span.sh` - 完整測試套件

測試多個不同場景，全面驗證 API 行為。

**用法**:

```bash
./scripts/test_longest_span.sh
```

**測試場景**:
1. 訂單建立 (複雜流程，10-12 spans)
2. 使用者資料查詢 (簡單流程，4-5 spans)
3. 報表生成 (長時間操作，10-12 spans)
4. 搜尋功能 (中等複雜度，6-7 spans)

**輸出內容**:
- 每個場景的完整分析
- Tempo 原始資料對比
- API 回傳結果對比
- 問題總結和改進建議

## 前置需求

### 1. 啟動服務

需要同時啟動兩個服務：

#### Tempo OTLP Trace Demo (端口 8080)

```bash
cd tempo-otlp-trace-demo
make up
```

驗證: `curl http://localhost:8080/healthz`

#### Tempo Latency Anomaly Service (端口 8081)

```bash
cd tempo-latency-anomaly-service
make up
```

驗證: `curl http://localhost:8081/healthz`

### 2. 檢查 Tempo

確保 Tempo 正常運行 (端口 3200):

```bash
curl http://localhost:3200/api/search
```

### 3. 安裝依賴

測試腳本需要以下工具：

- `curl` - HTTP 請求
- `jq` - JSON 解析

在 macOS 上安裝:

```bash
brew install jq
```

## 環境變數

可以透過環境變數自訂端點：

```bash
export TRACE_DEMO_URL=http://localhost:8080
export ANOMALY_SERVICE_URL=http://localhost:8081
export TEMPO_URL=http://localhost:3200
```

## 預期結果

根據我們的分析，測試應該會顯示：

### ⚠️ 問題確認

```
最長 Span (所有spans): span-root (1500ms)
Root Span: span-root (1500ms)
最長 Leaf Span: span-db-query (800ms)

⚠️ 問題確認: 最長 span 就是 root span!
   這驗證了我們的討論 - parent span 總是最長的

⚠️ API 回傳的是 ROOT span
   問題: 在實際應用中，這個資訊價值有限
```

### ✅ 應該改成什麼

建議的改進方向：

1. **選項 A: 只回傳 leaf spans**
   - 只考慮沒有 children 的 spans
   - 這些是實際執行工作的 spans
   - 更能反映真正的性能瓶頸

2. **選項 B: 計算 self-time**
   - Parent span self-time = total time - children time
   - 找出「自身時間」最長的 span
   - 代表該 span 本身做了最多工作

3. **選項 C: 提供參數選擇**
   - `?mode=all` - 所有 spans (目前行為)
   - `?mode=leaf` - 只考慮 leaf spans
   - `?mode=direct_children` - 只考慮 root 的直接子節點

## 快速開始

```bash
# 1. 啟動所有服務
cd tempo-otlp-trace-demo && make up &
cd tempo-latency-anomaly-service && make up &

# 2. 等待服務啟動 (約 10 秒)
sleep 10

# 3. 執行簡單測試
cd tempo-latency-anomaly-service
./scripts/test_longest_span_simple.sh

# 4. 執行完整測試套件
./scripts/test_longest_span.sh
```

## 測試輸出範例

```
=========================================
測試 Trace ID: abc123def456
=========================================

步驟 1: 查詢 Tempo...
✓ Tempo 查詢成功

--- Trace 中的所有 Spans ---
  SpanID: abc123... | Parent: ROOT... | 1500ms | GET /api/user/profile
  SpanID: def456... | Parent: abc123... | 800ms | db.query
  SpanID: ghi789... | Parent: abc123... | 500ms | cache.get
  SpanID: jkl012... | Parent: abc123... | 200ms | auth.verify

總共 4 個 spans

Root Span:
  SpanID: abc123...
  Name: GET /api/user/profile
  Duration: 1500ms

Duration 最長的 Span:
  SpanID: abc123...
  Name: GET /api/user/profile
  Duration: 1500ms
  Parent: ROOT...

最長的 Leaf Span (沒有 children):
  SpanID: def456...
  Name: db.query
  Duration: 800ms

--- 分析結果 ---
⚠️  最長 span 就是 root span
   這證明了: parent span 通常是最長的

步驟 2: 測試 longest-span API...
✓ API 呼叫成功

API 回傳的 Span:
  SpanID: abc123...
  Name: GET /api/user/profile
  Duration: 1500ms
  Parent: ROOT...

--- 驗證結果 ---
✓ API 回傳的確實是 duration 最長的 span
⚠️  API 回傳的是 root span
   問題: 在實際應用中，這個資訊價值有限

建議: 應該回傳最長的 leaf span:
  Name: db.query
  Duration: 800ms
  這才是真正的性能瓶頸點
```

## 疑難排解

### 服務連線失敗

```
❌ Trace Demo 服務未啟動
請先啟動: cd tempo-otlp-trace-demo && make up
```

**解決方法**: 確保服務已啟動並監聽正確的端口

### Tempo 查詢失敗

```
❌ Tempo 查詢失敗或 trace 不存在
```

**可能原因**:
1. Trace 還沒寫入 Tempo (增加等待時間)
2. Tempo 服務未啟動
3. Trace ID 錯誤

### jq 命令找不到

```
bash: jq: command not found
```

**解決方法**:
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq
```

## 相關文件

- [API 文檔](../docs/api/README.md)
- [架構設計](../ARCHITECTURE.md)
- [Swagger UI](http://localhost:8081/swagger/index.html)

## 總結

這些測試腳本幫助我們驗證了一個重要的發現：

> 在真實的分散式追蹤場景中，root span (parent span) 幾乎總是 duration 最長的，因為它包含了所有子操作的時間。目前的 `longest-span` API 雖然技術上正確，但在實際應用中價值有限，建議改為回傳最長的 leaf span 或提供更靈活的查詢選項。
