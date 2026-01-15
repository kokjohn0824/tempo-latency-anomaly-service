# Fallback 機制實作計劃

## 執行策略

使用 **codex exec** 來協助產生程式碼,AI 負責協調、統籌和追蹤進度。

## Subtasks 列表 (18 項)

### Phase 1: 資料結構和配置 (Tasks 1-3)

#### Task 1: 更新 domain models
**檔案**: `internal/domain/model.go`
**內容**:
- 新增 `BaselineSource` 類型 (exact/nearby/daytype/global/unavailable)
- 更新 `AnomalyCheckResponse` 加入:
  - `BaselineSource` 欄位
  - `FallbackLevel` 欄位
  - `SourceDetails` 欄位
  - `CannotDetermine` 欄位

#### Task 2: 更新 config 結構
**檔案**: 
- `internal/config/config.go` - 新增 `FallbackConfig` 結構
- `internal/config/defaults.go` - 設定預設值

**FallbackConfig 內容**:
```go
type FallbackConfig struct {
    Enabled                bool `mapstructure:"enabled"`
    NearbyHoursEnabled     bool `mapstructure:"nearby_hours_enabled"`
    NearbyHoursRange       int  `mapstructure:"nearby_hours_range"`
    NearbyMinSamples       int  `mapstructure:"nearby_min_samples"`
    DayTypeGlobalEnabled   bool `mapstructure:"daytype_global_enabled"`
    DayTypeGlobalMinSamples int `mapstructure:"daytype_global_min_samples"`
    FullGlobalEnabled      bool `mapstructure:"full_global_enabled"`
    FullGlobalMinSamples   int  `mapstructure:"full_global_min_samples"`
}
```

#### Task 3: 更新 config YAML
**檔案**: 
- `configs/config.dev.yaml`
- `configs/config.example.yaml`

**新增內容**:
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

---

### Phase 2: Store 層擴展 (Task 4)

#### Task 4: 實作批次查詢
**檔案**:
- `internal/store/store.go` - 新增介面方法
- `internal/store/redis/baseline.go` - 實作批次查詢

**新增方法**:
```go
// GetBaselines 批次查詢多個 baseline keys
GetBaselines(ctx context.Context, keys []string) (map[string]*Baseline, error)
```

---

### Phase 3: Baseline Lookup Service (Tasks 5-9)

#### Task 5: 建立 baseline_lookup.go
**檔案**: `internal/service/baseline_lookup.go` (新檔案)

**結構**:
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

func NewBaselineLookup(store store.Store, cfg *config.Config) *BaselineLookup
func (bl *BaselineLookup) LookupWithFallback(ctx, service, endpoint, bucket) (*BaselineResult, error)
```

#### Task 6: 實作 Level 1 - tryExactMatch
**方法**: `tryExactMatch(ctx, service, endpoint, bucket) *BaselineResult`
- 查詢精確的 `base:{service}|{endpoint}|{hour}|{dayType}`
- 檢查 sampleCount >= min_samples
- 返回 source=exact, level=1

#### Task 7: 實作 Level 2 - tryNearbyHours
**方法**: `tryNearbyHours(ctx, service, endpoint, bucket) *BaselineResult`
- 查詢 ±1, ±2 小時的同 dayType 資料
- 合併符合條件的相鄰時段
- 計算合併後的統計值
- 返回 source=nearby, level=2, details="16,18"

#### Task 8: 實作 Level 3 - tryDayTypeGlobal
**方法**: `tryDayTypeGlobal(ctx, service, endpoint, dayType) *BaselineResult`
- 查詢所有 hour (0-23) 的同 dayType 資料
- 合併所有時段的樣本
- 返回 source=daytype, level=3

#### Task 9: 實作 Level 4 - tryFullGlobal
**方法**: `tryFullGlobal(ctx, service, endpoint) *BaselineResult`
- 查詢所有 hour 和 dayType 的資料
- 合併所有樣本
- 返回 source=global, level=4

---

### Phase 4: 整合到 Check Service (Tasks 10-11)

#### Task 10: 更新 check.go
**檔案**: `internal/service/check.go`

**變更**:
- 加入 `baselineLookup *BaselineLookup` 欄位
- 更新 `NewCheck` 接受 baselineLookup 參數
- 替換原有的 `GetBaseline` 邏輯為 `LookupWithFallback`
- 更新回應包含 fallback 資訊

#### Task 11: 更新 app.go
**檔案**: `internal/app/app.go`

**變更**:
- 建立 `BaselineLookup` service 實例
- 傳遞給 `Check` service
- 更新 `App` 結構加入 `BaselineLookup` 欄位

---

### Phase 5: 文檔更新 (Tasks 12-13)

#### Task 12: 更新 Swagger 註解
**檔案**: `internal/api/handlers/check.go`

**新增說明**:
- 說明 fallback 機制
- 說明新增的回應欄位
- 提供範例

#### Task 13: 重新生成 Swagger 文檔
**指令**: `swag init -g cmd/server/main.go -o ./docs`

---

### Phase 6: 測試 (Task 14)

#### Task 14: 建立測試腳本
**檔案**: `scripts/test_fallback_scenarios.sh` (新檔案)

**測試場景**:
1. Level 1: 精確匹配 (當前時段有足夠資料)
2. Level 2: 相鄰時段 fallback (當前時段資料不足)
3. Level 3: 同類型天全局 (凌晨時段測試)
4. Level 4: 完全全局 (使用不存在的時段)
5. Level 5: 無資料 (使用不存在的 service)
6. 驗證回應包含正確的 fallback 資訊
7. 效能測試 (fallback 不應顯著增加延遲)

---

### Phase 7: README 更新 (Task 15)

#### Task 15: 更新 README.md
**檔案**: `README.md`

**新增章節**:
```markdown
## Fallback 機制

系統使用多層級 fallback 策略確保任何合理的請求都能得到異常判斷:

1. **Level 1 - 精確時段**: 使用請求時間的精確時段資料
2. **Level 2 - 相鄰時段**: 使用 ±2 小時的相鄰時段資料
3. **Level 3 - 同類型天**: 使用所有 weekday 或 weekend 的資料
4. **Level 4 - 完全全局**: 使用該服務/端點的所有資料

### 配置

fallback:
  enabled: true                      # 啟用 fallback
  nearby_hours_range: 2              # 相鄰時段範圍
  nearby_min_samples: 20             # 相鄰時段最少樣本
  ...

### 回應範例

{
  "isAnomaly": false,
  "baselineSource": "nearby",        # 使用相鄰時段
  "fallbackLevel": 2,                # Level 2
  "sourceDetails": "16,18",          # 使用 16 和 18 時的資料
  ...
}
```

---

### Phase 8: 部署和驗證 (Tasks 16-18)

#### Task 16: 重新建置和部署
**指令**: `docker compose -f docker/compose.yml up -d --build`

#### Task 17: 執行完整測試
**指令**: 
- `./scripts/test_fallback_scenarios.sh`
- `./scripts/test_available_api.sh`
- 手動測試各種時間戳

#### Task 18: 提交變更
**指令**: `git add . && git commit -m "..."`

---

## Codex Exec 使用策略

### 方式 1: 單一檔案生成
```bash
# 使用 codex exec 生成單一檔案
codex exec "根據 FALLBACK_STRATEGY_DESIGN.md 的設計,生成 internal/domain/model.go 的更新,新增 BaselineSource 類型和更新 AnomalyCheckResponse 結構"
```

### 方式 2: 多檔案協調
```bash
# 使用 codex exec 生成相關的多個檔案
codex exec "實作 baseline lookup service,包含 baseline_lookup.go 和相關的 helper 方法"
```

### 方式 3: 測試生成
```bash
# 生成測試腳本
codex exec "根據 fallback 設計生成完整的測試腳本 test_fallback_scenarios.sh,涵蓋所有 5 個 level 的測試場景"
```

## 執行順序

1. **Tasks 1-3**: 基礎結構 (可並行)
2. **Task 4**: Store 層擴展 (依賴 Task 1)
3. **Tasks 5-9**: Lookup service (依賴 Tasks 1-4)
4. **Tasks 10-11**: 整合 (依賴 Tasks 5-9)
5. **Tasks 12-13**: 文檔 (依賴 Tasks 1, 10)
6. **Task 14**: 測試腳本 (依賴 Tasks 1-11)
7. **Task 15**: README (依賴全部)
8. **Tasks 16-18**: 部署驗證 (依賴全部)

## 預期時間

- Phase 1-2: 30 分鐘
- Phase 3: 60 分鐘
- Phase 4: 30 分鐘
- Phase 5-6: 30 分鐘
- Phase 7-8: 30 分鐘

**總計**: 約 3 小時

## 成功指標

- [ ] 所有 18 個 tasks 完成
- [ ] 測試腳本全部通過
- [ ] 任意時間戳都能得到判斷結果
- [ ] 回應包含正確的 fallback 資訊
- [ ] 文檔完整更新
- [ ] Git 提交完成
