# Fallback Strategy 設計文檔

## 問題陳述

### 現有問題
1. **過於嚴格的時間分桶**: 只查詢精確的 `{hour}|{dayType}`,沒有資料就無法判斷
2. **資料利用率低**: 已收集的其他時段資料完全未使用
3. **冷啟動問題**: 新服務/端點需要很長時間才能覆蓋所有 48 個時段
4. **使用者體驗差**: 合理的請求卻得到 "insufficient samples" 回應

### 影響範圍
- 用戶輸入任意合理的 timestamp,應該都能得到異常判斷結果
- 系統應該盡可能利用已有的統計資料
- 即使是新服務也應該能快速提供基本的異常檢測

## 解決方案: 多層級 Fallback 策略

### 設計原則
1. **優先使用精確匹配**: 精確時段的資料最準確
2. **逐級降級**: 按照資料相關性從高到低嘗試
3. **透明化**: 明確告知使用者使用了哪一層的資料
4. **可配置**: 允許調整 fallback 行為

### Fallback 層級設計

```
Level 1: 精確時段 (Exact Match)
  ↓ (如果樣本數 < min_samples)
Level 2: 相鄰時段 (Nearby Hours)
  ↓ (如果所有相鄰時段都不足)
Level 3: 同類型天全局 (Day Type Global)
  ↓ (如果同類型天樣本不足)
Level 4: 完全全局 (Full Global)
  ↓ (如果仍無資料)
Level 5: 無法判斷 (No Data Available)
```

## 詳細設計

### Level 1: 精確時段 (Exact Match)
**查詢**: `base:{service}|{endpoint}|{hour}|{dayType}`

**條件**: `sampleCount >= min_samples`

**範例**:
- 請求時間: 2026-01-15 17:30 (Thursday)
- 查詢: `base:my-service|GET /api|17|weekday`
- 如果有 >= 30 樣本,使用此 baseline

**優點**: 最精確,考慮了時間模式
**缺點**: 需要每個時段都有足夠資料

---

### Level 2: 相鄰時段 (Nearby Hours)
**策略**: 查詢相鄰 ±1, ±2 小時的同類型天資料

**查詢順序**:
1. `base:{service}|{endpoint}|{hour±1}|{dayType}`
2. `base:{service}|{endpoint}|{hour±2}|{dayType}`

**範例**:
- 請求時間: 17:30 (weekday)
- 查詢順序:
  1. hour=16|weekday, hour=18|weekday (±1)
  2. hour=15|weekday, hour=19|weekday (±2)

**合併策略**:
- 收集所有符合條件的相鄰時段資料
- 如果總樣本數 >= min_samples,合併計算統計值
- 使用加權平均或直接合併所有樣本

**優點**: 考慮時間相近性,資料仍有相關性
**缺點**: 可能混合不同負載模式

**配置**:
```yaml
fallback:
  nearby_hours_range: 2  # 查詢 ±2 小時範圍
  nearby_min_samples: 20  # 相鄰時段最少樣本數
```

---

### Level 3: 同類型天全局 (Day Type Global)
**策略**: 使用所有同類型天(weekday/weekend)的資料,不分時段

**查詢**: 所有 `base:{service}|{endpoint}|*|{dayType}` 的資料

**範例**:
- 請求: weekday
- 查詢所有 hour=0-23 的 weekday 資料
- 合併計算全局 weekday baseline

**實作方式**:
```
Option A: 預先計算並儲存
  - Key: base:{service}|{endpoint}|global|weekday
  - 定期更新 (每小時)

Option B: 即時計算
  - 查詢所有相關 keys
  - 合併所有樣本計算統計值
```

**優點**: 
- 考慮工作日/週末的差異
- 樣本數大幅增加

**缺點**: 
- 忽略時段差異
- 可能混合高峰/離峰資料

**配置**:
```yaml
fallback:
  daytype_global_min_samples: 50  # 全局統計最少樣本數
  daytype_global_cache_ttl: 1h    # 快取時間
```

---

### Level 4: 完全全局 (Full Global)
**策略**: 使用該服務/端點的所有資料,不分時段和天類型

**查詢**: 所有 `base:{service}|{endpoint}|*|*` 的資料

**實作方式**:
```
Key: base:{service}|{endpoint}|global|all
```

**優點**: 
- 最大化樣本數
- 總是能提供判斷(只要有任何資料)

**缺點**: 
- 完全忽略時間模式
- 可能不夠精確

**使用場景**:
- 新服務剛上線
- 低流量端點
- 緊急情況需要基本判斷

---

### Level 5: 無資料 (No Data Available)
**回應**: 明確告知無法判斷,但不視為錯誤

**訊息**: 
```json
{
  "isAnomaly": false,
  "cannotDetermine": true,
  "reason": "no baseline data available for this service/endpoint",
  "suggestion": "wait for data collection or check service name"
}
```

## 實作細節

### 1. 配置結構

```yaml
# configs/config.dev.yaml
fallback:
  enabled: true                      # 啟用 fallback 機制
  
  # Level 2: 相鄰時段
  nearby_hours_enabled: true
  nearby_hours_range: 2              # ±2 小時
  nearby_min_samples: 20             # 每個相鄰時段最少樣本
  
  # Level 3: 同類型天全局
  daytype_global_enabled: true
  daytype_global_min_samples: 50
  daytype_global_precompute: true    # 是否預先計算
  
  # Level 4: 完全全局
  full_global_enabled: true
  full_global_min_samples: 30
  full_global_precompute: true
  
  # 通用設定
  max_combined_samples: 5000         # 合併樣本上限(避免計算過慢)
```

### 2. 資料結構更新

```go
// domain/model.go

type BaselineSource string

const (
    SourceExact       BaselineSource = "exact"        // Level 1
    SourceNearby      BaselineSource = "nearby"       // Level 2
    SourceDayType     BaselineSource = "daytype"      // Level 3
    SourceGlobal      BaselineSource = "global"       // Level 4
    SourceUnavailable BaselineSource = "unavailable"  // Level 5
)

type AnomalyCheckResponse struct {
    IsAnomaly        bool            `json:"isAnomaly"`
    CannotDetermine  bool            `json:"cannotDetermine,omitempty"`
    Bucket           TimeBucket      `json:"bucket"`
    Baseline         *BaselineStats  `json:"baseline,omitempty"`
    BaselineSource   BaselineSource  `json:"baselineSource"`          // 新增
    FallbackLevel    int             `json:"fallbackLevel,omitempty"` // 新增
    SourceDetails    string          `json:"sourceDetails,omitempty"` // 新增
    Explanation      string          `json:"explanation"`
}
```

### 3. Service 層實作

```go
// internal/service/baseline_lookup.go (新檔案)

type BaselineLookup struct {
    store store.Store
    cfg   *config.Config
}

// LookupWithFallback 使用 fallback 策略查詢 baseline
func (bl *BaselineLookup) LookupWithFallback(
    ctx context.Context,
    service, endpoint string,
    bucket TimeBucket,
) (*BaselineResult, error) {
    
    // Level 1: Exact match
    if result := bl.tryExactMatch(ctx, service, endpoint, bucket); result != nil {
        return result, nil
    }
    
    // Level 2: Nearby hours
    if bl.cfg.Fallback.NearbyHoursEnabled {
        if result := bl.tryNearbyHours(ctx, service, endpoint, bucket); result != nil {
            return result, nil
        }
    }
    
    // Level 3: Day type global
    if bl.cfg.Fallback.DayTypeGlobalEnabled {
        if result := bl.tryDayTypeGlobal(ctx, service, endpoint, bucket.DayType); result != nil {
            return result, nil
        }
    }
    
    // Level 4: Full global
    if bl.cfg.Fallback.FullGlobalEnabled {
        if result := bl.tryFullGlobal(ctx, service, endpoint); result != nil {
            return result, nil
        }
    }
    
    // Level 5: No data
    return &BaselineResult{
        Source:          SourceUnavailable,
        CannotDetermine: true,
    }, nil
}

type BaselineResult struct {
    Baseline        *store.Baseline
    Source          BaselineSource
    FallbackLevel   int
    SourceDetails   string
    CannotDetermine bool
}
```

### 4. Store 層擴展

```go
// internal/store/store.go

type BaselineOps interface {
    GetBaseline(ctx context.Context, key string) (*Baseline, error)
    SetBaseline(ctx context.Context, key string, b Baseline) error
    
    // 新增: 批次查詢
    GetBaselines(ctx context.Context, keys []string) (map[string]*Baseline, error)
    
    // 新增: 模式查詢
    GetBaselinesByPattern(ctx context.Context, pattern string) (map[string]*Baseline, error)
}
```

## 效能考量

### 1. 快取策略
- Level 3 和 Level 4 的全局統計應該預先計算並快取
- 快取 key: `base:{service}|{endpoint}|global|{dayType}` 和 `base:{service}|{endpoint}|global|all`
- TTL: 1 小時(可配置)

### 2. 計算優化
- 設定最大合併樣本數上限(如 5000)
- 使用抽樣技術處理大量樣本
- 批次查詢減少 Redis 往返次數

### 3. 預先計算
- Background job 定期計算全局統計
- 與現有的 baseline recompute job 整合

## 測試策略

### 1. 單元測試
- 測試每個 fallback level 的邏輯
- 測試 fallback 順序正確性
- 測試邊界條件

### 2. 整合測試
- 模擬不同資料可用性場景
- 驗證 fallback 行為符合預期
- 效能測試

### 3. 測試場景

```bash
# 場景 1: 精確匹配
timestamp: 17:00 weekday
資料: hour=17|weekday 有 50 samples
預期: 使用 Level 1, source=exact

# 場景 2: 相鄰時段 fallback
timestamp: 17:00 weekday
資料: hour=17|weekday 只有 10 samples
      hour=16|weekday 有 40 samples
預期: 使用 Level 2, source=nearby, details="16,18"

# 場景 3: 全局 fallback
timestamp: 03:00 weekday (凌晨,可能沒資料)
資料: 其他時段有資料
預期: 使用 Level 3 或 4, source=daytype/global

# 場景 4: 完全無資料
timestamp: any
資料: 完全沒有此 service/endpoint 的資料
預期: cannotDetermine=true, source=unavailable
```

## 向後相容性

### 1. API 回應
- 新增欄位使用 `omitempty`,不影響現有客戶端
- `isAnomaly` 和 `explanation` 保持原有行為

### 2. 配置
- fallback 功能可透過 `fallback.enabled: false` 關閉
- 關閉時行為與原有邏輯完全相同

### 3. 資料儲存
- 不改變現有 Redis key 結構
- 新增的全局統計使用新的 key 格式

## 部署計劃

### Phase 1: 基礎實作 (優先)
- [ ] 實作 Level 1-2 (精確 + 相鄰時段)
- [ ] 更新 domain models
- [ ] 基本測試

### Phase 2: 全局統計 (次要)
- [ ] 實作 Level 3-4 (全局統計)
- [ ] 預先計算 job
- [ ] 完整測試

### Phase 3: 優化 (可選)
- [ ] 效能優化
- [ ] 監控指標
- [ ] 文檔完善

## 監控指標

建議新增以下 metrics:

```
# Fallback 使用統計
anomaly_check_fallback_level{level="1|2|3|4|5"} counter
anomaly_check_baseline_source{source="exact|nearby|daytype|global|unavailable"} counter

# 效能指標
anomaly_check_baseline_lookup_duration_seconds{level="1|2|3|4"} histogram
anomaly_check_combined_samples{level="2|3|4"} histogram
```

## 總結

這個 fallback 策略將大幅改善系統的可用性和使用者體驗:

1. **提高覆蓋率**: 從 ~2% (有資料的時段) 提升到 ~95% (有任何資料)
2. **改善冷啟動**: 新服務可以立即使用全局統計
3. **保持精確性**: 優先使用最相關的資料
4. **透明化**: 使用者知道判斷的可信度

這是一個平衡精確性和可用性的實用方案。
