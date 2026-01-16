# 單元測試文檔

**Last Updated**: 2026-01-16

---

## 📊 測試覆蓋率總覽

| 模組 | 覆蓋率 | 狀態 | 優先級 |
|------|--------|------|--------|
| **internal/stats** | **89.1%** | ✅ 優秀 | 高 |
| **internal/config** | **88.7%** | ✅ 優秀 | 中 |
| **internal/domain** | **88.2%** | ✅ 優秀 | 高 |
| **internal/service** | **64.5%** | ✅ 良好 | 高 |
| **總體覆蓋率** | **33.1%** | ⚠️ 需改進 | - |

**核心業務邏輯覆蓋率**: ~82% (stats + domain + service)

---

## 🎯 執行測試

### 基本測試

```bash
# 執行所有單元測試
make test

# 或直接使用 go test
go test ./internal/...
```

### 測試覆蓋率

```bash
# 生成覆蓋率報告 (HTML)
make test-coverage

# 查看覆蓋率報告
open coverage.html
```

### 快速測試 (CI/CD)

```bash
# 只執行快速測試
make test-short
```

### 詳細輸出

```bash
# 查看詳細測試過程
make test-verbose
```

---

## 📁 測試檔案結構

```
internal/
├── stats/
│   ├── calculator.go
│   ├── calculator_test.go      ✅ 89.1% coverage
│   ├── percentile.go
│   ├── percentile_test.go
│   ├── mad.go
│   └── mad_test.go
├── domain/
│   ├── key.go
│   ├── time_test.go            ✅ 88.2% coverage
│   └── model.go
├── config/
│   ├── config.go
│   ├── config_test.go          ✅ 88.7% coverage
│   └── defaults.go
├── service/
│   ├── check.go
│   ├── check_test.go           ✅ 測試異常判斷邏輯
│   ├── baseline_lookup.go
│   ├── baseline_lookup_test.go ✅ 測試 5 層 fallback
│   ├── ingest.go
│   ├── ingest_test.go          ✅ 測試資料寫入與 dedup
│   └── ...
└── store/
    └── mocks/
        └── store_mocks.go      ✅ testify/mock 實作
```

---

## ✅ 已測試功能

### 1. 統計計算 (internal/stats)

**測試檔案**: `calculator_test.go`, `percentile_test.go`, `mad_test.go`

**測試案例**:
- ✅ P50 計算 (奇數/偶數樣本)
- ✅ P95 計算 (nearest-rank 演算法)
- ✅ MAD 計算
- ✅ 空樣本處理
- ✅ 單一樣本處理
- ✅ 閾值公式 (max(P95 × factor, P50 + k × MAD))

**關鍵測試**:
```go
// TestThresholdFormula_MaxOfRelativeAndAbsolute
// 驗證異常閾值計算公式不會改變
```

### 2. 時間桶 (internal/domain)

**測試檔案**: `time_test.go`

**測試案例**:
- ✅ TimeBucket 生成 (各時區)
- ✅ DayType 判斷 (weekday/weekend)
- ✅ 時區轉換 (Asia/Taipei)
- ✅ 邊界條件 (午夜00:00, 23:59)
- ✅ 無效輸入處理

**關鍵測試**:
```go
// TestParseTimeBucket_TimezoneAndDayType
// 確保週一~五=weekday, 週六日=weekend
```

### 3. 配置載入 (internal/config)

**測試檔案**: `config_test.go`

**測試案例**:
- ✅ 預設值載入
- ✅ YAML 檔案覆寫
- ✅ 環境變數覆寫
- ✅ 所有配置欄位驗證

### 4. 異常檢測 (internal/service/check.go)

**測試檔案**: `check_test.go`

**測試案例**:
- ✅ 正常延遲判斷為非異常
- ✅ 超過閾值判斷為異常
- ✅ 無 baseline 處理
- ✅ 樣本不足處理
- ✅ Baseline lookup 整合
- ✅ 回應欄位完整性

**關鍵測試**:
```go
// TestCheck_Evaluate_NormalAndAnomaly
// 驗證閾值判斷邏輯正確性
```

### 5. Fallback 機制 (internal/service/baseline_lookup.go)

**測試檔案**: `baseline_lookup_test.go`

**測試案例**:
- ✅ Level 1: 精確匹配 (exact)
- ✅ Level 2: 相鄰時段 (nearby, 加權平均)
- ✅ Level 3: 同 dayType 全局 (daytype)
- ✅ Level 4: 完全全局 (global)
- ✅ Level 5: 無可用資料 (unavailable)
- ✅ 加權平均計算驗證
- ✅ Min samples 門檻驗證

**關鍵測試**:
```go
// TestBaselineLookup_Level2_NearbyHoursWeighted
// 驗證加權平均計算: (P50₁×n₁ + P50₂×n₂) / (n₁+n₂)
```

### 6. 資料寫入 (internal/service/ingest.go)

**測試檔案**: `ingest_test.go`

**測試案例**:
- ✅ TraceEvent 解析
- ✅ Dedup 機制 (重複 traceID 跳過)
- ✅ 時間桶計算整合
- ✅ 寫入流程完整性
- ✅ Mock store 驗證

---

## 🚨 Breaking Change 偵測

### 如何偵測

所有測試都包含**固定期望值**,任何邏輯變更都會導致測試失敗:

#### 範例 1: 統計計算變更偵測

```go
func TestP50_OddAndEvenSamples(t *testing.T) {
    odd := []int64{5, 1, 3}
    mOdd := P50(odd)
    assert.Equal(t, 3.0, mOdd) // 固定期望值
    
    // 如果有人修改 P50 計算邏輯,此測試會失敗
}
```

#### 範例 2: Fallback 順序變更偵測

```go
func TestBaselineLookup_Level1_ExactMatch(t *testing.T) {
    // 測試確保 exact match 優先於 fallback
    result, err := bl.LookupWithFallback(...)
    assert.Equal(t, domain.SourceExact, result.Source)
    assert.Equal(t, 1, result.FallbackLevel)
    
    // 如果 fallback 順序被改變,此測試會失敗
}
```

#### 範例 3: 閾值公式變更偵測

```go
func TestThresholdFormula_MaxOfRelativeAndAbsolute(t *testing.T) {
    // 驗證 threshold = max(P95*factor, P50+k*MAD)
    threshold := /* 計算 */
    assert.InDelta(t, 2600.0, threshold, 1e-9)
    
    // 如果公式被修改,期望值會不匹配
}
```

### CI 整合

**Makefile 已配置**:
```makefile
# Docker build 前自動執行測試
docker-build: test
    @echo "Tests passed! Building Docker image..."
    docker compose -f docker/compose.yml build
```

**使用方式**:
```bash
# 建置前自動測試
make docker-build

# 如果測試失敗,建置會停止
# ✓ 防止 breaking changes 進入 production
```

---

## 🔍 測試最佳實踐

### 1. 使用 testify/assert

```go
import "github.com/stretchr/testify/assert"

func TestExample(t *testing.T) {
    result := Calculate(input)
    
    // 清晰的斷言
    assert.Equal(t, expected, result)
    assert.NoError(t, err)
    assert.True(t, condition)
}
```

### 2. 使用 testify/mock

```go
import "github.com/stretchr/testify/mock"

func TestWithMock(t *testing.T) {
    m := new(mocks.MockStore)
    
    // 設定期望
    m.On("GetBaseline", mock.Anything, "key").Return(baseline, nil)
    
    // 執行測試
    service := NewService(m)
    result := service.DoSomething()
    
    // 驗證 mock 被正確調用
    m.AssertExpectations(t)
}
```

### 3. 測試命名規範

```go
// 模式: Test<FunctionName>_<Scenario>
func TestP50_OddAndEvenSamples(t *testing.T)
func TestCheck_Evaluate_NormalAndAnomaly(t *testing.T)
func TestBaselineLookup_Level1_ExactMatch(t *testing.T)
```

### 4. 表格驅動測試

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct{
        name     string
        input    int
        expected int
    }{
        {"case1", 1, 2},
        {"case2", 2, 4},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Calculate(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 5. 不修改輸入

```go
func TestP50_OddAndEvenSamples(t *testing.T) {
    odd := []int64{5, 1, 3}
    origOdd := append([]int64(nil), odd...) // 備份
    
    mOdd := P50(odd)
    
    // 驗證輸入未被修改
    assert.Equal(t, origOdd, odd)
}
```

---

## 📈 持續改進

### 未來改進項目

1. **提升整體覆蓋率** (目標: 50%+)
   - [ ] API handlers 測試
   - [ ] Jobs 層測試 (tempo_poller, baseline_recompute)
   - [ ] Redis store 層整合測試

2. **整合測試**
   - [ ] 端到端測試場景
   - [ ] Redis integration tests (使用 testcontainers)

3. **性能測試**
   - [ ] Benchmark tests for stats calculations
   - [ ] Load testing for anomaly detection

---

## 🛠 故障排除

### 測試失敗常見原因

1. **時區問題**
   ```go
   // 確保使用正確時區
   loc, _ := time.LoadLocation("Asia/Taipei")
   ts := time.Date(2024, 1, 8, 12, 0, 0, 0, loc)
   ```

2. **浮點數比較**
   ```go
   // 使用 InDelta 而非 Equal
   assert.InDelta(t, expected, actual, 1e-9)
   ```

3. **Mock 未設定**
   ```go
   // 記得設定所有預期的 mock 調用
   m.On("Method", mock.Anything).Return(value, nil)
   ```

### 清除測試快取

```bash
# 如果測試結果不更新
go clean -testcache
make test
```

---

## 📚 相關資源

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify GitHub](https://github.com/stretchr/testify)
- [UNIT_TEST_PLAN.md](UNIT_TEST_PLAN.md) - 原始測試計畫

---

**維護者**: 請在修改核心邏輯前先執行測試,確保沒有 breaking changes。

**CI 保護**: 所有 Docker builds 都需要先通過測試。
