# Tempo 資料撈取邏輯分析

**分析日期**: 2026-01-15  
**分析目的**: 了解目前從 Tempo 撈取資料的邏輯,以及是否有盡可能撈取歷史資料

---

## 📊 當前撈取邏輯

### 1. 輪詢機制 (Polling)

**檔案**: `internal/jobs/tempo_poller.go`

```go
// 每隔固定時間執行一次
interval := p.cfg.Polling.TempoInterval  // 預設 15 秒
lookback := int(p.cfg.Polling.TempoLookback / time.Second)  // 預設 120 秒
```

**運作方式**:
- ✅ 服務啟動時立即執行一次
- ✅ 之後每 15 秒執行一次查詢
- ✅ 每次查詢最近 120 秒的 traces

### 2. 時間範圍設定

**檔案**: `internal/tempo/query.go`

```go
func BuildQueryParams(lookbackSeconds int) url.Values {
    now := time.Now().Unix()
    start := now - int64(lookbackSeconds)  // 往前推 120 秒
    
    q := url.Values{}
    q.Set("start", strconv.FormatInt(start, 10))
    q.Set("end", strconv.FormatInt(now, 10))
    return q
}
```

**時間窗口**:
- 起始時間: `now - 120 秒`
- 結束時間: `now`
- **結論**: 只查詢最近 2 分鐘的資料

### 3. 資料筆數限制

**檔案**: `internal/tempo/client.go`

```go
params.Set("limit", "100") // 每次查詢最多 100 筆
```

**限制**:
- ❌ 每次查詢最多返回 100 筆 traces
- ❌ 沒有分頁或多次查詢機制
- ❌ 如果 2 分鐘內超過 100 筆,會遺漏資料

---

## 🔍 問題分析

### ❌ 問題 1: **沒有撈取歷史資料**

**現狀**:
```
服務啟動 ────> 只查詢最近 120 秒 ────> 持續輪詢
              (只有這 2 分鐘)
```

**影響**:
- ❌ 啟動前的歷史資料完全不會被撈取
- ❌ 無法建立完整的 baseline
- ❌ 需要運行 48+ 小時才能覆蓋所有時段
- ❌ 冷啟動問題嚴重

### ❌ 問題 2: **資料可能遺漏**

**場景**:
```
時間: 00:00:00 - 00:02:00 (120秒)
產生的 traces: 150 筆
查詢限制: 100 筆
結果: 遺漏 50 筆 (33%)
```

**風險**:
- ❌ 高流量時段會遺漏資料
- ❌ 影響 baseline 準確性
- ❌ 可能導致誤判

### ❌ 問題 3: **時間覆蓋不均勻**

**現狀**:
```
輪詢頻率: 15 秒一次
查詢範圍: 120 秒

重疊率: 800%
(同一時間點會被查詢 8 次)
```

**問題**:
- ⚠️ 資源浪費 (重複查詢)
- ⚠️ Tempo 負載較高
- ⚠️ 但確保不會遺漏時段

---

## 📈 配置分析

### 當前配置 (`configs/config.dev.yaml`)

```yaml
polling:
  tempo_interval: 15s      # 每 15 秒輪詢一次
  tempo_lookback: 120s     # 查詢最近 120 秒
  baseline_interval: 30s   # 每 30 秒重算 baseline
```

### 實際行為

| 時間 | 查詢範圍 | 說明 |
|------|----------|------|
| T=0s | [-120s, 0s] | 服務啟動,查詢最近 2 分鐘 |
| T=15s | [-105s, 15s] | 第二次查詢 |
| T=30s | [-90s, 30s] | 第三次查詢 |
| ... | ... | 持續輪詢 |

**觀察**:
- ✅ 時間連續性良好 (重疊 8 倍)
- ❌ 永遠不會查詢超過 2 分鐘前的資料
- ❌ 歷史資料無法補齊

---

## 💡 改進建議

### 方案 1: 初始回填 (Backfill) 機制 ⭐ 推薦

**概念**:
```
服務啟動時:
1. 先執行歷史資料回填 (backfill)
2. 再開始正常的輪詢

時間軸:
[-7天] ────────> [-2分鐘] ────────> [現在]
    ↑ 回填階段 ↑      ↑ 正常輪詢 ↑
```

**實作建議**:

```go
// 新增配置
type PollingConfig struct {
    TempoInterval    time.Duration
    TempoLookback    time.Duration
    BaselineInterval time.Duration
    
    // 新增: 回填設定
    BackfillEnabled  bool          // 是否啟用回填
    BackfillDuration time.Duration // 回填時間範圍 (例如 7 天)
    BackfillBatch    time.Duration // 每批查詢範圍 (例如 1 小時)
}

// 新增回填邏輯
func (p *TempoPoller) backfill(ctx context.Context) {
    if !p.cfg.Polling.BackfillEnabled {
        return
    }
    
    duration := p.cfg.Polling.BackfillDuration
    batchSize := p.cfg.Polling.BackfillBatch
    
    start := time.Now().Add(-duration)
    end := time.Now().Add(-p.cfg.Polling.TempoLookback)
    
    log.Printf("Starting backfill: %s to %s", start, end)
    
    for current := start; current.Before(end); current = current.Add(batchSize) {
        batchEnd := current.Add(batchSize)
        if batchEnd.After(end) {
            batchEnd = end
        }
        
        // 查詢這個時段
        lookbackSec := int(time.Since(current).Seconds())
        events, err := p.client.QueryTraces(ctx, lookbackSec)
        if err != nil {
            log.Printf("backfill error: %v", err)
            continue
        }
        
        // 處理資料...
        log.Printf("Backfilled %d traces from %s to %s", 
            len(events), current, batchEnd)
        
        // 避免過度負載 Tempo
        time.Sleep(1 * time.Second)
    }
    
    log.Printf("Backfill completed")
}

// 在 Run() 中調用
func (p *TempoPoller) Run(ctx context.Context) {
    // 先執行回填
    p.backfill(ctx)
    
    // 再開始正常輪詢
    p.tick(ctx)
    
    t := time.NewTicker(interval)
    // ...
}
```

**配置範例**:
```yaml
polling:
  tempo_interval: 15s
  tempo_lookback: 120s
  baseline_interval: 30s
  
  # 新增: 回填設定
  backfill_enabled: true
  backfill_duration: 168h    # 7 天
  backfill_batch: 1h         # 每次查詢 1 小時
```

**優點**:
- ✅ 快速建立完整 baseline (7 天資料)
- ✅ 冷啟動時間大幅縮短 (從 48 小時 → 1 小時)
- ✅ 所有時段都有資料
- ✅ 可配置回填範圍

**缺點**:
- ⚠️ 初始啟動時間較長 (取決於回填範圍)
- ⚠️ 對 Tempo 負載較高 (需要限流)

---

### 方案 2: 增加每次查詢的筆數限制

**當前**: `limit=100`  
**建議**: `limit=1000` 或更高

```go
params.Set("limit", "1000") // 增加到 1000 筆
```

**優點**:
- ✅ 減少遺漏資料的風險
- ✅ 簡單易實作

**缺點**:
- ⚠️ 單次查詢時間變長
- ⚠️ 記憶體使用增加

---

### 方案 3: 動態調整 lookback 時間

**概念**: 根據資料量動態調整查詢範圍

```go
// 如果上次查詢接近 limit,縮短 lookback
// 如果上次查詢很少,延長 lookback

if len(events) > 90 {
    // 接近 limit,縮短時間範圍
    lookback = max(60, lookback / 2)
} else if len(events) < 10 {
    // 資料很少,延長時間範圍
    lookback = min(600, lookback * 2)
}
```

**優點**:
- ✅ 自適應調整
- ✅ 高流量時不遺漏,低流量時更高效

**缺點**:
- ⚠️ 複雜度較高
- ⚠️ 需要仔細調優

---

### 方案 4: 分頁查詢 (如果 Tempo 支援)

**概念**: 使用 Tempo 的分頁 API 多次查詢

```go
func (p *TempoPoller) queryAllTraces(ctx context.Context, lookback int) []TraceEvent {
    var allEvents []TraceEvent
    offset := 0
    limit := 100
    
    for {
        params := BuildQueryParams(lookback)
        params.Set("limit", strconv.Itoa(limit))
        params.Set("offset", strconv.Itoa(offset))
        
        events, err := p.client.QueryTraces(ctx, params)
        if err != nil || len(events) == 0 {
            break
        }
        
        allEvents = append(allEvents, events...)
        if len(events) < limit {
            break // 沒有更多資料了
        }
        
        offset += limit
    }
    
    return allEvents
}
```

**優點**:
- ✅ 完全不會遺漏資料
- ✅ 可以獲取所有 traces

**缺點**:
- ❌ 需要 Tempo API 支援分頁
- ⚠️ 查詢時間較長

---

## 📋 建議實作優先順序

### 短期 (立即可做)

1. **增加 limit 參數** ⭐⭐⭐
   - 從 100 增加到 500-1000
   - 簡單快速,立即改善

2. **調整 lookback 配置** ⭐⭐
   - 根據實際流量調整 120s → 60s 或 180s
   - 平衡資料覆蓋和重複查詢

### 中期 (1-2 週)

3. **實作回填機制** ⭐⭐⭐⭐⭐
   - 啟動時自動回填 7 天歷史資料
   - 大幅改善冷啟動體驗
   - **最推薦的改進!**

4. **加入查詢統計** ⭐⭐
   - 記錄每次查詢的資料筆數
   - 監控是否接近 limit (警示可能遺漏)

### 長期 (1+ 月)

5. **動態調整機制** ⭐⭐⭐
   - 自適應 lookback 時間
   - 智能調整查詢頻率

6. **分頁查詢** ⭐
   - 如果 Tempo 支援
   - 完全避免遺漏

---

## 🎯 推薦配置

### 立即改進 (最小變更)

```yaml
polling:
  tempo_interval: 15s
  tempo_lookback: 120s
  baseline_interval: 30s
```

```go
// internal/tempo/client.go
params.Set("limit", "500") // 從 100 增加到 500
```

### 理想配置 (含回填)

```yaml
polling:
  tempo_interval: 15s
  tempo_lookback: 120s
  baseline_interval: 30s
  
  # 回填設定
  backfill_enabled: true
  backfill_duration: 168h    # 7 天
  backfill_batch: 1h         # 每批 1 小時
  backfill_limit: 1000       # 回填時的 limit
```

---

## 📊 預期效果

### 改進前

| 指標 | 當前值 | 問題 |
|------|--------|------|
| 歷史資料覆蓋 | 0 天 | ❌ 無歷史資料 |
| 冷啟動時間 | 48+ 小時 | ❌ 太長 |
| 資料遺漏風險 | 高 (limit=100) | ❌ 高流量會遺漏 |
| Tempo 查詢負載 | 中 (8x 重疊) | ⚠️ 可接受 |

### 改進後 (方案 1: 回填 + 增加 limit)

| 指標 | 預期值 | 改善 |
|------|--------|------|
| 歷史資料覆蓋 | 7 天 | ✅ 完整覆蓋 |
| 冷啟動時間 | 1-2 小時 | ✅ 減少 96% |
| 資料遺漏風險 | 低 (limit=500) | ✅ 大幅降低 |
| Tempo 查詢負載 | 中-高 (回填期) | ⚠️ 需限流 |

---

## 🛠️ 實作範例

### 完整的回填實作

請參考以下完整實作範例:

```go
// configs/config.dev.yaml 新增配置
polling:
  tempo_interval: 15s
  tempo_lookback: 120s
  baseline_interval: 30s
  backfill_enabled: true
  backfill_duration: 168h    # 7 天
  backfill_batch: 1h         # 每批 1 小時

// internal/config/config.go 更新結構
type PollingConfig struct {
    TempoInterval    time.Duration `mapstructure:"tempo_interval"`
    TempoLookback    time.Duration `mapstructure:"tempo_lookback"`
    BaselineInterval time.Duration `mapstructure:"baseline_interval"`
    BackfillEnabled  bool          `mapstructure:"backfill_enabled"`
    BackfillDuration time.Duration `mapstructure:"backfill_duration"`
    BackfillBatch    time.Duration `mapstructure:"backfill_batch"`
}

// internal/jobs/tempo_poller.go 新增回填邏輯
// (完整程式碼見上方方案 1)
```

---

## 💡 總結

### 當前狀況

❌ **沒有撈取歷史資料**
- 只查詢最近 120 秒
- 啟動前的資料完全不會被撈取
- 需要 48+ 小時才能建立完整 baseline

❌ **可能遺漏資料**
- 每次最多 100 筆
- 高流量時段會遺漏
- 影響 baseline 準確性

### 建議改進

⭐⭐⭐⭐⭐ **立即實作回填機制**
- 啟動時回填 7 天歷史資料
- 冷啟動時間從 48 小時 → 1-2 小時
- **最重要的改進!**

⭐⭐⭐ **增加 limit 參數**
- 從 100 → 500 或 1000
- 簡單快速,立即改善

⭐⭐ **加入監控和統計**
- 追蹤查詢筆數
- 警示可能遺漏的情況

---

**結論**: 當前系統**沒有**盡可能撈取所有時間點的資料,只撈取最近 2 分鐘。強烈建議實作回填機制以改善冷啟動體驗和資料完整性!
