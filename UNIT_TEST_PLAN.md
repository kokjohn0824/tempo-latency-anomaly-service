# 單元測試實作計畫

**目標**: 在 Docker build 前偵測 breaking changes,確保程式碼品質與穩定性

---

## 📋 測試範圍分析

### 核心模組 (優先級: 高)

1. **internal/stats/** - 統計計算邏輯
   - P50, P95, MAD 計算
   - 異常判斷邏輯
   - 關鍵: 直接影響異常判斷結果

2. **internal/domain/** - 時間桶與模型
   - TimeBucket 生成邏輯
   - DayType 判斷 (weekday/weekend)
   - 關鍵: 影響 baseline 分桶

3. **internal/config/** - 配置處理
   - 預設值處理
   - 配置驗證
   - 關鍵: 影響服務啟動

### 服務層 (優先級: 高)

4. **internal/service/check.go** - 異常檢測服務
   - Evaluate 方法邏輯
   - 閾值計算
   - 關鍵: 核心業務邏輯

5. **internal/service/baseline_lookup.go** - Fallback 機制
   - LookupWithFallback 5 層邏輯
   - 加權平均計算
   - 關鍵: 影響資料稀疏時的判斷

6. **internal/service/ingest.go** - 資料寫入
   - Trace 事件處理
   - 時間桶計算
   - Dedup 邏輯

### Store 層 (優先級: 中)

7. **internal/store/** - 資料存取層
   - Mock interfaces
   - 基本操作驗證

---

## 🛠 測試工具選擇

### 標準庫 + Testify

**選擇理由**:
- Go 標準 testing 套件穩定可靠
- testify/assert 提供清晰的斷言
- testify/mock 支援 interface mocking
- 無需額外複雜依賴

**安裝**:
```bash
go get github.com/stretchr/testify
```

---

## 📁 測試檔案結構

```
internal/
├── stats/
│   ├── calculator.go
│   └── calculator_test.go       # 新增
├── domain/
│   ├── time.go
│   └── time_test.go              # 新增
├── config/
│   ├── config.go
│   └── config_test.go            # 新增
├── service/
│   ├── check.go
│   ├── check_test.go             # 新增
│   ├── baseline_lookup.go
│   ├── baseline_lookup_test.go   # 新增
│   ├── ingest.go
│   └── ingest_test.go            # 新增
└── store/
    ├── mocks/                     # 新增
    │   └── store_mocks.go
    └── store_test.go              # 新增
```

---

## 🎯 測試覆蓋目標

| 模組 | 目標覆蓋率 | 優先級 |
|------|-----------|--------|
| stats | 90%+ | 高 |
| domain/time | 85%+ | 高 |
| service/check | 85%+ | 高 |
| service/baseline_lookup | 80%+ | 高 |
| service/ingest | 75%+ | 中 |
| config | 70%+ | 中 |

---

## 🔄 CI 整合流程

### Makefile 目標

```makefile
.PHONY: test
test:
	@echo "Running unit tests..."
	go test -v -race -cover ./internal/...

.PHONY: test-coverage
test-coverage:
	@echo "Generating coverage report..."
	go test -v -race -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-short
test-short:
	@echo "Running short tests..."
	go test -short -v ./internal/...
```

### Docker Build 前置檢查

修改 `docker/Dockerfile`:
```dockerfile
# 在 build 前執行測試
RUN go test -short ./internal/...
```

或使用獨立的 build script:
```bash
#!/bin/bash
# scripts/build.sh

echo "Running tests..."
make test || exit 1

echo "Building Docker image..."
docker compose -f docker/compose.yml build
```

---

## 📝 測試案例設計

### 1. stats/calculator_test.go

**測試重點**:
- ✅ P50 計算正確性 (奇數/偶數樣本)
- ✅ P95 計算正確性
- ✅ MAD 計算正確性
- ✅ 空樣本處理
- ✅ 單一樣本處理
- ✅ 異常閾值計算

### 2. domain/time_test.go

**測試重點**:
- ✅ TimeBucket 生成 (各時區)
- ✅ DayType 判斷 (週一~日)
- ✅ 時區轉換正確性
- ✅ 邊界條件 (午夜、週末轉換)

### 3. service/check_test.go

**測試重點**:
- ✅ 正常延遲判斷為非異常
- ✅ 超過閾值判斷為異常
- ✅ 無 baseline 處理
- ✅ 樣本不足處理
- ✅ Baseline lookup 整合

### 4. service/baseline_lookup_test.go

**測試重點**:
- ✅ Level 1 (exact) 匹配
- ✅ Level 2 (nearby) fallback
- ✅ Level 3 (daytype) fallback
- ✅ Level 4 (global) fallback
- ✅ Level 5 (unavailable)
- ✅ 加權平均計算
- ✅ Min samples 驗證

### 5. service/ingest_test.go

**測試重點**:
- ✅ TraceEvent 解析
- ✅ 時間桶計算
- ✅ Dedup 機制
- ✅ Rolling window 維護

---

## 🚨 Breaking Change 偵測

### 測試場景

1. **統計計算變更**:
   ```go
   // 測試確保 P95 計算不會改變
   func TestP95Stability(t *testing.T) {
       samples := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
       p95 := CalculateP95(samples)
       assert.Equal(t, 9.55, p95) // 固定期望值
   }
   ```

2. **Fallback 邏輯變更**:
   ```go
   // 測試確保 fallback 順序不變
   func TestFallbackOrder(t *testing.T) {
       // 應依序嘗試 exact → nearby → daytype → global
   }
   ```

3. **閾值計算變更**:
   ```go
   // 測試確保閾值公式不變
   func TestThresholdFormula(t *testing.T) {
       // threshold = max(P95, P50 + k*MAD)
   }
   ```

---

## 📊 執行順序

1. **Task 1-2**: 分析結構 + 建立基礎 (本文件)
2. **Task 3**: 使用 codex 實作核心模組測試
3. **Task 4**: 使用 codex 實作服務層測試
4. **Task 5**: 使用 codex 實作 mock 與 store 測試
5. **Task 6**: 建立 Makefile + 整合 CI
6. **Task 7**: 設定覆蓋率報告
7. **Task 8**: 撰寫文檔
8. **Task 9**: 驗證流程
9. **Task 10-11**: 更新文檔 + 提交

---

## ✅ 驗收標準

- [ ] 所有測試通過 (`make test`)
- [ ] 整體覆蓋率 > 70%
- [ ] 核心模組覆蓋率 > 80%
- [ ] Docker build 前自動執行測試
- [ ] 測試執行時間 < 30 秒
- [ ] 文檔完整清晰
