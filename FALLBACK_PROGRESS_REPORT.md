# Fallback 機制實作進度報告

**更新時間**: 2026-01-15 17:30  
**狀態**: Phase 1 進行中 (3/18 完成)

## ✅ 已完成任務 (3/18)

### Task 1: ✅ 更新 domain models
**檔案**: `internal/domain/model.go`
**完成內容**:
- ✅ 新增 `BaselineSource` 類型 (exact/nearby/daytype/global/unavailable)
- ✅ 更新 `AnomalyCheckResponse` 結構:
  - `BaselineSource` - 標註使用的 baseline 來源
  - `FallbackLevel` - 標註 fallback 層級 (1-5)
  - `SourceDetails` - 詳細說明使用的資料來源
  - `CannotDetermine` - 標註是否無法判斷

### Task 2: ✅ 更新 config 結構
**檔案**: 
- `internal/config/config.go`
- `internal/config/defaults.go`

**完成內容**:
- ✅ 新增 `FallbackConfig` 結構到 Config
- ✅ 設定所有 fallback 相關的預設值
- ✅ 支援環境變數覆蓋

### Task 3: ✅ 更新 config YAML
**檔案**:
- `configs/config.dev.yaml`
- `configs/config.example.yaml`

**完成內容**:
- ✅ 加入完整的 fallback 配置區塊
- ✅ 所有參數都有合理的預設值

## ⏳ 進行中 / 待完成任務 (15/18)

### Phase 2: Store 層擴展 (1 task)
- [ ] **Task 4**: 實作批次查詢 - `GetBaselines` 方法

### Phase 3: Baseline Lookup Service (5 tasks)  
- [ ] **Task 5**: 建立 `baseline_lookup.go` 骨架
- [ ] **Task 6**: 實作 Level 1 - tryExactMatch
- [ ] **Task 7**: 實作 Level 2 - tryNearbyHours  
- [ ] **Task 8**: 實作 Level 3 - tryDayTypeGlobal
- [ ] **Task 9**: 實作 Level 4 - tryFullGlobal

### Phase 4: 整合 (2 tasks)
- [ ] **Task 10**: 更新 `check.go` 使用 BaselineLookup
- [ ] **Task 11**: 更新 `app.go` 初始化 service

### Phase 5: 文檔 (2 tasks)
- [ ] **Task 12**: 更新 Swagger 註解
- [ ] **Task 13**: 重新生成 Swagger 文檔

### Phase 6: 測試 (1 task)
- [ ] **Task 14**: 建立完整測試腳本

### Phase 7: README (1 task)
- [ ] **Task 15**: 更新 README.md

### Phase 8: 部署驗證 (3 tasks)
- [ ] **Task 16**: 重新建置和部署
- [ ] **Task 17**: 執行完整測試
- [ ] **Task 18**: Git 提交

## 📊 進度統計

- **完成**: 3 tasks (16.7%)
- **剩餘**: 15 tasks (83.3%)
- **預估剩餘時間**: 2.5 小時

## 🎯 下一步建議

### 選項 A: 繼續完整實作 (推薦)
繼續實作剩餘的 15 個 tasks,完成完整的 fallback 機制。

**優點**:
- 一次性解決所有問題
- 達到最佳使用者體驗
- 完整的測試覆蓋

**時間**: 約 2.5 小時

### 選項 B: 分階段實作
先實作 Level 1-2 (Tasks 4-7, 10-18),後續再加入 Level 3-4。

**優點**:
- 更快看到成效
- 降低風險
- 可以先部署測試

**時間**: 
- Phase 1: 約 1.5 小時 (Level 1-2)
- Phase 2: 約 1 小時 (Level 3-4)

### 選項 C: 暫停並討論
暫停實作,討論設計細節或調整方案。

## 📝 已建立的文檔

1. ✅ `FALLBACK_STRATEGY_DESIGN.md` - 完整設計文檔
2. ✅ `FALLBACK_IMPLEMENTATION_PLAN.md` - 實作計劃
3. ✅ `FALLBACK_PROGRESS_REPORT.md` - 本進度報告

## 🔧 技術細節

### 已完成的程式碼變更

#### 1. Domain Models (model.go)
```go
type BaselineSource string

const (
    SourceExact       BaselineSource = "exact"
    SourceNearby      BaselineSource = "nearby"
    SourceDayType     BaselineSource = "daytype"
    SourceGlobal      BaselineSource = "global"
    SourceUnavailable BaselineSource = "unavailable"
)

type AnomalyCheckResponse struct {
    // ... 原有欄位 ...
    BaselineSource   BaselineSource  `json:"baselineSource"`
    FallbackLevel    int             `json:"fallbackLevel,omitempty"`
    SourceDetails    string          `json:"sourceDetails,omitempty"`
    CannotDetermine  bool            `json:"cannotDetermine,omitempty"`
}
```

#### 2. Config 結構 (config.go)
```go
type FallbackConfig struct {
    Enabled                  bool
    NearbyHoursEnabled       bool
    NearbyHoursRange         int
    NearbyMinSamples         int
    DayTypeGlobalEnabled     bool
    DayTypeGlobalMinSamples  int
    FullGlobalEnabled        bool
    FullGlobalMinSamples     int
}
```

#### 3. YAML 配置
```yaml
fallback:
  enabled: true
  nearby_hours_enabled: true
  nearby_hours_range: 2
  nearby_min_samples: 20
  daytype_global_enabled: true
  daytype_global_min_samples: 50
  full_global_enabled: true
  full_global_min_samples: 30
```

## 💡 關鍵實作要點

### 接下來需要實作的核心邏輯

1. **批次查詢** (Task 4)
   - Redis MGET 或 pipeline 查詢多個 keys
   - 提高查詢效能

2. **BaselineLookup Service** (Tasks 5-9)
   - 主要邏輯在 `LookupWithFallback` 方法
   - 依序嘗試 5 個 level
   - 合併多個時段的樣本數據
   - 計算合併後的統計值

3. **整合到 Check** (Task 10)
   - 替換原有的單一 GetBaseline 調用
   - 使用 BaselineLookup.LookupWithFallback
   - 更新回應包含 fallback 資訊

## 🚀 預期效果

完成後,系統將能夠:

- ✅ 對任意合理的 timestamp 提供異常判斷
- ✅ 自動使用最相關的可用資料
- ✅ 透明化告知使用者資料來源
- ✅ 大幅提升覆蓋率 (從 ~2% 到 ~95%)

## ❓ 需要決定

請告知是否:
1. 繼續完整實作 (選項 A)
2. 分階段實作 (選項 B)
3. 暫停討論 (選項 C)
4. 其他建議

我已準備好繼續執行剩餘的 15 個 tasks!
