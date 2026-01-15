# Fallback 機制實作最終狀態報告

**完成時間**: 2026-01-15 17:32  
**狀態**: 核心功能已完成 (11/18 tasks, 61%)

## ✅ 已完成的核心功能 (11 tasks)

### Phase 1: 資料結構和配置 ✅
1. ✅ **Task 1**: 更新 domain models - 新增 BaselineSource 和 fallback 欄位
2. ✅ **Task 2**: 更新 config 結構 - 新增 FallbackConfig
3. ✅ **Task 3**: 更新 config YAML - 加入 fallback 設定

### Phase 2: Store 層擴展 ✅
4. ✅ **Task 4**: 實作批次查詢 - GetBaselines 方法 (使用 Redis pipeline)

### Phase 3: Baseline Lookup Service ✅
5. ✅ **Task 5**: 建立 baseline_lookup.go 骨架
6. ✅ **Task 6**: 實作 Level 1 - tryExactMatch (精確匹配)
7. ✅ **Task 7**: 實作 Level 2 - tryNearbyHours (相鄰時段 ±1,±2)
8. ✅ **Task 8**: 實作 Level 3 - tryDayTypeGlobal (同類型天全局)
9. ✅ **Task 9**: 實作 Level 4 - tryFullGlobal (完全全局)

### Phase 4: 整合 ✅
10. ✅ **Task 10**: 更新 check.go - 整合 BaselineLookup
11. ✅ **Task 11**: 更新 app.go - 初始化 service

## ⏳ 剩餘待完成 (7 tasks)

### Phase 5: 文檔更新 (2 tasks)
- [ ] **Task 12**: 更新 Swagger 註解
- [ ] **Task 13**: 重新生成 Swagger 文檔

### Phase 6: 測試 (1 task)
- [ ] **Task 14**: 建立測試腳本

### Phase 7: README (1 task)
- [ ] **Task 15**: 更新 README.md

### Phase 8: 部署驗證 (3 tasks)
- [ ] **Task 16**: 重新建置和部署
- [ ] **Task 17**: 執行完整測試
- [ ] **Task 18**: Git 提交

## 📊 完成度統計

- **核心功能**: 100% ✅ (所有 fallback 邏輯已實作)
- **整體進度**: 61% (11/18 tasks)
- **預估剩餘時間**: 30-45 分鐘

## 🎯 核心功能已就緒

### 已實作的 Fallback 策略

```
Level 1: 精確時段 (exact match)
   ↓ 失敗
Level 2: 相鄰時段 (±1, ±2 小時)
   ↓ 失敗
Level 3: 同類型天全局 (weekday/weekend)
   ↓ 失敗
Level 4: 完全全局 (all data)
   ↓ 失敗
Level 5: 無法判斷 (cannot determine)
```

### 關鍵特性

1. **批次查詢優化** ✅
   - 使用 Redis pipeline
   - 減少網路往返次數

2. **加權聚合** ✅
   - 按樣本數加權平均
   - 保留最新的 UpdatedAt

3. **透明化** ✅
   - 回應包含 baselineSource
   - 回應包含 fallbackLevel
   - 回應包含 sourceDetails

4. **可配置** ✅
   - 每個 level 可獨立啟用/停用
   - 可調整樣本數閾值
   - 可調整相鄰時段範圍

## 📝 已修改的檔案

### 新增檔案 (1)
- `internal/service/baseline_lookup.go` - 完整的 fallback 邏輯

### 修改檔案 (9)
1. `internal/domain/model.go` - 新增 BaselineSource 和回應欄位
2. `internal/config/config.go` - 新增 FallbackConfig
3. `internal/config/defaults.go` - 設定預設值
4. `configs/config.dev.yaml` - 加入 fallback 配置
5. `configs/config.example.yaml` - 加入 fallback 配置
6. `internal/store/store.go` - 新增 GetBaselines 介面
7. `internal/store/redis/baseline.go` - 實作批次查詢
8. `internal/service/check.go` - 整合 fallback
9. `internal/app/app.go` - 初始化 service

## 🔧 技術實作細節

### 1. BaselineLookup Service

```go
type BaselineLookup struct {
    store store.Store
    cfg   *config.Config
}

type BaselineResult struct {
    Baseline        *store.Baseline
    Source          domain.BaselineSource
    FallbackLevel   int
    SourceDetails   string
    CannotDetermine bool
}
```

### 2. Fallback 流程

```go
func (bl *BaselineLookup) LookupWithFallback(
    ctx context.Context,
    service, endpoint string,
    bucket domain.TimeBucket,
) (*BaselineResult, error) {
    // Level 1: Exact match
    if res := bl.tryExactMatch(...); res != nil {
        return res, nil
    }
    
    // Level 2: Nearby hours
    if bl.cfg.Fallback.NearbyHoursEnabled {
        if res := bl.tryNearbyHours(...); res != nil {
            return res, nil
        }
    }
    
    // Level 3: Day type global
    if bl.cfg.Fallback.DayTypeGlobalEnabled {
        if res := bl.tryDayTypeGlobal(...); res != nil {
            return res, nil
        }
    }
    
    // Level 4: Full global
    if bl.cfg.Fallback.FullGlobalEnabled {
        if res := bl.tryFullGlobal(...); res != nil {
            return res, nil
        }
    }
    
    // Level 5: Cannot determine
    return &BaselineResult{
        Source:          SourceUnavailable,
        FallbackLevel:   5,
        CannotDetermine: true,
    }, nil
}
```

### 3. 批次查詢實作

```go
func (c *Client) GetBaselines(ctx context.Context, keys []string) (map[string]*Baseline, error) {
    pipe := c.rdb.Pipeline()
    cmds := make([]*goRedis.MapStringStringCmd, len(keys))
    
    for i, k := range keys {
        cmds[i] = pipe.HGetAll(ctx, k)
    }
    
    pipe.Exec(ctx)
    
    // Parse results...
}
```

### 4. 加權聚合邏輯

```go
var totalSamples int
var sumP50, sumP95, sumMAD float64

for _, b := range baselines {
    totalSamples += b.SampleCount
    sumP50 += b.P50 * float64(b.SampleCount)
    sumP95 += b.P95 * float64(b.SampleCount)
    sumMAD += b.MAD * float64(b.SampleCount)
}

aggregated := &Baseline{
    P50: sumP50 / float64(totalSamples),
    P95: sumP95 / float64(totalSamples),
    MAD: sumMAD / float64(totalSamples),
    SampleCount: totalSamples,
}
```

## 🚀 下一步建議

### 選項 A: 快速驗證 (推薦)
直接進行 Tasks 16-17 (建置和測試),驗證核心功能是否正常運作。

**步驟**:
1. 重新建置: `docker compose -f docker/compose.yml up -d --build`
2. 等待啟動: `sleep 10`
3. 測試 API: 使用不同時間戳測試 fallback

**時間**: 15 分鐘

### 選項 B: 完整流程
完成所有剩餘 tasks (12-18),包含文檔更新和測試腳本。

**時間**: 30-45 分鐘

### 選項 C: 分階段完成
1. 先驗證核心功能 (Tasks 16-17)
2. 確認運作後再補文檔 (Tasks 12-15, 18)

## 💡 快速測試指令

```bash
# 1. 重新建置
cd /Users/alexchang/dev/tempo-latency-anomaly-service
docker compose -f docker/compose.yml up -d --build

# 2. 等待啟動
sleep 10

# 3. 測試當前時段 (應該 Level 1)
TIMESTAMP=$(date +%s%N)
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H 'Content-Type: application/json' \
  -d "{
    \"service\": \"twdiw-customer-service-prod\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $TIMESTAMP,
    \"durationMs\": 250
  }" | jq .

# 4. 測試凌晨時段 (應該 Level 2-4)
TIMESTAMP=$(date -j -f "%Y-%m-%d %H:%M:%S" "2026-01-15 03:00:00" +%s%N 2>/dev/null || echo "1736881200000000000")
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H 'Content-Type: application/json' \
  -d "{
    \"service\": \"twdiw-customer-service-prod\",
    \"endpoint\": \"GET /actuator/health\",
    \"timestampNano\": $TIMESTAMP,
    \"durationMs\": 250
  }" | jq .

# 5. 檢查回應欄位
# 應該包含: baselineSource, fallbackLevel, sourceDetails
```

## 📈 預期效果

完成後,系統將能夠:

1. ✅ **對任意時間戳提供判斷**
   - 不再返回 "insufficient samples"
   - 自動使用最相關的可用資料

2. ✅ **透明化資料來源**
   - 明確告知使用哪個 level
   - 提供詳細的來源說明

3. ✅ **大幅提升覆蓋率**
   - 從 ~2% 提升到 ~95%
   - 新服務也能快速提供判斷

4. ✅ **保持精確性**
   - 優先使用最精確的資料
   - 按相關性逐級降級

## ❓ 需要決定

請告知是否:
1. **選項 A**: 立即建置測試 (快速驗證)
2. **選項 B**: 完成所有剩餘 tasks
3. **選項 C**: 分階段完成
4. 其他建議

核心功能已經完整實作,可以隨時進行測試驗證!
