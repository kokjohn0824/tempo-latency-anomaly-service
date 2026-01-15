# 文檔重組計劃

**目標**: 將 20 個 MD 檔案整理為清晰、易於導覽的結構

**問題**: 
- 根目錄有 20 個 MD 檔案,過於混亂
- 多個重複主題的文檔 (Swagger x3, Fallback x6, Testing x3)
- 缺乏清晰的文檔導覽
- 開發過程文檔與使用者文檔混雜

---

## 📊 現有文檔分析

### 當前檔案清單 (20 個)

| 檔案名稱 | 行數 | 類型 | 處理方式 |
|---------|------|------|----------|
| **README.md** | 281 | 核心 | ✅ 保留,增強 |
| **ARCHITECTURE.md** | 479 | 核心 | ✅ 保留 |
| **EXAMPLES.md** | 504 | 核心 | ✅ 保留 |
| **QUICKSTART.md** | 265 | 核心 | 🔄 合併到 README |
| API_AVAILABLE_IMPLEMENTATION.md | 268 | 功能文檔 | 📁 移至 docs/features/ |
| FALLBACK_STRATEGY_DESIGN.md | 399 | 設計文檔 | 📁 移至 docs/design/ |
| FALLBACK_IMPLEMENTATION_COMPLETE.md | 567 | 開發記錄 | 📦 歸檔 docs/archive/ |
| FALLBACK_IMPLEMENTATION_PLAN.md | 279 | 開發記錄 | 📦 歸檔 docs/archive/ |
| FALLBACK_PROGRESS_REPORT.md | 195 | 開發記錄 | 📦 歸檔 docs/archive/ |
| FALLBACK_FINAL_STATUS.md | 287 | 開發記錄 | 📦 歸檔 docs/archive/ |
| FALLBACK_TEST_RESULTS.md | 345 | 測試報告 | 🔄 合併至 docs/reports/TESTING.md |
| TEST_REPORT.md | 241 | 測試報告 | 🔄 合併至 docs/reports/TESTING.md |
| TESTING_SUMMARY.md | 349 | 測試報告 | 🔄 合併至 docs/reports/TESTING.md |
| INTEGRATION_TEST_REPORT.md | 323 | 測試報告 | 🔄 合併至 docs/reports/TESTING.md |
| SWAGGER.md | 236 | Swagger文檔 | 🔄 合併至 docs/API.md |
| SWAGGER_IMPLEMENTATION.md | 247 | Swagger文檔 | 🔄 合併至 docs/API.md |
| SWAGGER_QUICKSTART.md | 175 | Swagger文檔 | 🔄 合併至 docs/API.md |
| TASK_PLAN.md | 353 | 臨時文檔 | 🗑️ 歸檔或刪除 |
| task.md | 321 | 臨時文檔 | 🗑️ 歸檔或刪除 |
| PROGRESS.md | 104 | 臨時文檔 | 🗑️ 歸檔或刪除 |

**圖例**:
- ✅ 保留在根目錄
- 🔄 合併整合
- 📁 移至新位置
- 📦 歸檔
- 🗑️ 刪除或歸檔

---

## 🎯 目標結構

### 根目錄 (4 個核心文檔)

```
/
├── README.md                    # 主要入口,包含快速開始
├── ARCHITECTURE.md              # 系統架構設計
├── EXAMPLES.md                  # 使用範例
└── DOCUMENTATION_INDEX.md       # 📚 文檔導覽 (新增)
```

### docs/ 目錄結構

```
docs/
├── design/                      # 設計文檔
│   ├── FALLBACK_STRATEGY.md    # Fallback 機制設計
│   └── TIME_BUCKETING.md       # 時間分桶設計 (可選)
│
├── features/                    # 功能說明文檔
│   ├── AVAILABLE_API.md        # /v1/available API
│   └── ANOMALY_DETECTION.md    # 異常檢測功能
│
├── api/                         # API 文檔
│   ├── README.md               # API 總覽
│   └── SWAGGER_GUIDE.md        # Swagger 使用指南
│
├── reports/                     # 測試與實作報告
│   ├── TESTING.md              # 綜合測試報告
│   └── IMPLEMENTATION.md       # 實作總結
│
└── archive/                     # 歷史開發記錄
    └── fallback-implementation/ # Fallback 實作過程
        ├── PLAN.md
        ├── PROGRESS.md
        ├── FINAL_STATUS.md
        └── COMPLETE.md
```

---

## 📋 詳細執行計劃

### Phase 1: 準備工作 (Tasks 1-2)

#### Task 1: 分析並分類所有 MD 檔案 ✅
- 已完成上述分析表格
- 確定每個檔案的處理方式

#### Task 2: 建立 docs/ 目錄結構
```bash
mkdir -p docs/{design,features,api,reports,archive/fallback-implementation}
```

---

### Phase 2: 整合重複文檔 (Tasks 3-5)

#### Task 3: 整合 Swagger 文檔
**目標**: 3 個 Swagger 文檔 → 1 個 `docs/api/SWAGGER_GUIDE.md`

**合併內容**:
- SWAGGER.md (236 行) - 基本說明
- SWAGGER_IMPLEMENTATION.md (247 行) - 實作細節
- SWAGGER_QUICKSTART.md (175 行) - 快速開始

**新文檔結構**:
```markdown
# Swagger API 文檔指南

## 快速開始
(來自 SWAGGER_QUICKSTART.md)

## API 總覽
(來自 SWAGGER.md)

## 實作細節
(來自 SWAGGER_IMPLEMENTATION.md)

## 使用範例
...
```

**Codex 指令**:
```bash
codex exec "合併 SWAGGER.md, SWAGGER_IMPLEMENTATION.md, SWAGGER_QUICKSTART.md 為單一檔案 docs/api/SWAGGER_GUIDE.md,結構為: 1) 快速開始, 2) API 總覽, 3) 實作細節, 4) 使用範例。移除重複內容,保持邏輯連貫" --full-auto
```

---

#### Task 4: 整合 Fallback 相關文檔
**目標**: 6 個 Fallback 文檔 → 1 個設計文檔 + 4 個歸檔

**保留為設計文檔**:
- `docs/design/FALLBACK_STRATEGY.md` (來自 FALLBACK_STRATEGY_DESIGN.md)

**歸檔開發記錄**:
- `docs/archive/fallback-implementation/PLAN.md` (來自 FALLBACK_IMPLEMENTATION_PLAN.md)
- `docs/archive/fallback-implementation/PROGRESS.md` (來自 FALLBACK_PROGRESS_REPORT.md)
- `docs/archive/fallback-implementation/FINAL_STATUS.md` (來自 FALLBACK_FINAL_STATUS.md)
- `docs/archive/fallback-implementation/COMPLETE.md` (來自 FALLBACK_IMPLEMENTATION_COMPLETE.md)

**整合測試結果到**:
- `docs/reports/TESTING.md` (部分內容來自 FALLBACK_TEST_RESULTS.md)

**Codex 指令**:
```bash
codex exec "1) 複製 FALLBACK_STRATEGY_DESIGN.md 到 docs/design/FALLBACK_STRATEGY.md, 2) 移動 FALLBACK_IMPLEMENTATION_PLAN.md 到 docs/archive/fallback-implementation/PLAN.md, 3) 移動 FALLBACK_PROGRESS_REPORT.md 到 docs/archive/fallback-implementation/PROGRESS.md, 4) 移動 FALLBACK_FINAL_STATUS.md 到 docs/archive/fallback-implementation/FINAL_STATUS.md, 5) 移動 FALLBACK_IMPLEMENTATION_COMPLETE.md 到 docs/archive/fallback-implementation/COMPLETE.md" --full-auto
```

---

#### Task 5: 整合測試報告
**目標**: 4 個測試報告 → 1 個 `docs/reports/TESTING.md`

**合併內容**:
- TEST_REPORT.md (241 行)
- TESTING_SUMMARY.md (349 行)
- INTEGRATION_TEST_REPORT.md (323 行)
- FALLBACK_TEST_RESULTS.md (345 行) - 部分內容

**新文檔結構**:
```markdown
# 測試報告總覽

## 測試摘要
(來自 TESTING_SUMMARY.md)

## 功能測試
(來自 TEST_REPORT.md)

## Fallback 機制測試
(來自 FALLBACK_TEST_RESULTS.md)

## 整合測試
(來自 INTEGRATION_TEST_REPORT.md)

## 測試腳本
- scripts/test_scenarios.sh
- scripts/test_fallback_scenarios.sh
- scripts/test_available_api.sh
```

**Codex 指令**:
```bash
codex exec "合併 TEST_REPORT.md, TESTING_SUMMARY.md, INTEGRATION_TEST_REPORT.md, FALLBACK_TEST_RESULTS.md 為單一檔案 docs/reports/TESTING.md,結構為: 1) 測試摘要, 2) 功能測試, 3) Fallback 機制測試, 4) 整合測試, 5) 測試腳本列表。移除重複內容,保持測試結果的完整性" --full-auto
```

---

### Phase 3: 處理其他文檔 (Task 6)

#### Task 6: 清理臨時任務文檔

**處理方式**:
- `task.md` → 刪除 (臨時任務記錄)
- `TASK_PLAN.md` → 刪除 (已完成的計劃)
- `PROGRESS.md` → 刪除 (已過時的進度)

**Codex 指令**:
```bash
codex exec "刪除 task.md, TASK_PLAN.md, PROGRESS.md 這三個臨時文檔" --full-auto
```

**處理其他文檔**:
- `API_AVAILABLE_IMPLEMENTATION.md` → `docs/features/AVAILABLE_API.md`
- `QUICKSTART.md` → 合併到 README.md

---

### Phase 4: 建立導覽文檔 (Tasks 7-8)

#### Task 7: 建立 DOCUMENTATION_INDEX.md

**內容**:
```markdown
# 📚 文檔導覽

## 🚀 快速開始

新使用者建議閱讀順序:
1. [README.md](README.md) - 專案介紹與快速開始
2. [EXAMPLES.md](EXAMPLES.md) - 使用範例
3. [docs/api/SWAGGER_GUIDE.md](docs/api/SWAGGER_GUIDE.md) - API 文檔

## 📖 核心文檔

### 根目錄
- **[README.md](README.md)** - 專案主要入口
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 系統架構設計
- **[EXAMPLES.md](EXAMPLES.md)** - 完整使用範例

## 🎨 設計文檔

- [Fallback 策略設計](docs/design/FALLBACK_STRATEGY.md)

## 🔧 功能說明

- [Available API 功能](docs/features/AVAILABLE_API.md)

## 📡 API 文檔

- [API 總覽](docs/api/README.md)
- [Swagger 使用指南](docs/api/SWAGGER_GUIDE.md)

## 📊 測試與報告

- [綜合測試報告](docs/reports/TESTING.md)

## 📦 歷史記錄

- [Fallback 實作過程](docs/archive/fallback-implementation/)
```

**Codex 指令**:
```bash
codex exec "建立 DOCUMENTATION_INDEX.md,內容包含: 1) 快速開始建議閱讀順序, 2) 核心文檔列表, 3) 設計文檔, 4) 功能說明, 5) API 文檔, 6) 測試報告, 7) 歷史記錄。使用清晰的分類和相對路徑連結" --full-auto
```

---

#### Task 8: 更新 README.md

**新增章節**:
```markdown
## 📚 文檔導覽

### 快速開始
- 新使用者請先閱讀本 README
- 查看 [EXAMPLES.md](EXAMPLES.md) 了解使用範例
- 訪問 [Swagger UI](http://localhost:8080/swagger/index.html) 測試 API

### 完整文檔
請參考 [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) 查看所有文檔。

**主要文檔**:
- 📐 [ARCHITECTURE.md](ARCHITECTURE.md) - 系統架構
- 📖 [EXAMPLES.md](EXAMPLES.md) - 使用範例
- 🎨 [設計文檔](docs/design/) - 功能設計
- 📡 [API 文檔](docs/api/) - API 說明
- 📊 [測試報告](docs/reports/) - 測試結果
```

**Codex 指令**:
```bash
codex exec "在 README.md 的 Quickstart 章節後新增 '文檔導覽' 章節,說明: 1) 新使用者閱讀順序, 2) 主要文檔連結 (ARCHITECTURE, EXAMPLES, docs/), 3) 參考 DOCUMENTATION_INDEX.md 查看完整列表。保持簡潔清晰" --full-auto
```

---

### Phase 5: 移動檔案與更新連結 (Tasks 9-10)

#### Task 9: 移動檔案到新結構

**移動清單**:
```bash
# 功能文檔
mv API_AVAILABLE_IMPLEMENTATION.md docs/features/AVAILABLE_API.md

# 設計文檔
cp FALLBACK_STRATEGY_DESIGN.md docs/design/FALLBACK_STRATEGY.md

# 歸檔
mv FALLBACK_IMPLEMENTATION_PLAN.md docs/archive/fallback-implementation/PLAN.md
mv FALLBACK_PROGRESS_REPORT.md docs/archive/fallback-implementation/PROGRESS.md
mv FALLBACK_FINAL_STATUS.md docs/archive/fallback-implementation/FINAL_STATUS.md
mv FALLBACK_IMPLEMENTATION_COMPLETE.md docs/archive/fallback-implementation/COMPLETE.md
```

**Codex 指令**:
```bash
codex exec "執行檔案移動: 1) API_AVAILABLE_IMPLEMENTATION.md → docs/features/AVAILABLE_API.md, 2) FALLBACK_STRATEGY_DESIGN.md → docs/design/FALLBACK_STRATEGY.md, 3) 移動 4 個 FALLBACK_IMPLEMENTATION_*.md 到 docs/archive/fallback-implementation/, 4) 建立 docs/api/README.md 作為 API 總覽" --full-auto
```

---

#### Task 10: 驗證與提交

**驗證清單**:
- [ ] 所有文檔連結正確
- [ ] DOCUMENTATION_INDEX.md 連結可用
- [ ] README.md 連結可用
- [ ] 根目錄只剩 4-5 個核心文檔
- [ ] docs/ 結構清晰

**Git 提交**:
```bash
git add .
git commit -m "docs: Reorganize documentation structure

- Consolidate 20 MD files into organized docs/ structure
- Merge duplicate docs (Swagger x3, Testing x4, Fallback x6)
- Create DOCUMENTATION_INDEX.md for easy navigation
- Archive development history to docs/archive/
- Update README.md with documentation guide

Structure:
- Root: 4 core docs (README, ARCHITECTURE, EXAMPLES, INDEX)
- docs/design/: Design documents
- docs/features/: Feature documentation
- docs/api/: API documentation
- docs/reports/: Test reports
- docs/archive/: Development history

This improves discoverability and reduces root directory clutter."
```

---

## 🎯 預期成果

### 整理前 (20 個檔案)
```
/ (根目錄)
├── README.md
├── ARCHITECTURE.md
├── EXAMPLES.md
├── QUICKSTART.md
├── API_AVAILABLE_IMPLEMENTATION.md
├── FALLBACK_STRATEGY_DESIGN.md
├── FALLBACK_IMPLEMENTATION_COMPLETE.md
├── FALLBACK_IMPLEMENTATION_PLAN.md
├── FALLBACK_PROGRESS_REPORT.md
├── FALLBACK_FINAL_STATUS.md
├── FALLBACK_TEST_RESULTS.md
├── TEST_REPORT.md
├── TESTING_SUMMARY.md
├── INTEGRATION_TEST_REPORT.md
├── SWAGGER.md
├── SWAGGER_IMPLEMENTATION.md
├── SWAGGER_QUICKSTART.md
├── TASK_PLAN.md
├── task.md
└── PROGRESS.md
```

### 整理後 (4-5 個核心 + docs/)
```
/ (根目錄)
├── README.md                    # 增強版,含文檔導覽
├── ARCHITECTURE.md
├── EXAMPLES.md
├── DOCUMENTATION_INDEX.md       # 新增:完整導覽
└── docs/
    ├── design/
    │   └── FALLBACK_STRATEGY.md
    ├── features/
    │   └── AVAILABLE_API.md
    ├── api/
    │   ├── README.md
    │   └── SWAGGER_GUIDE.md
    ├── reports/
    │   └── TESTING.md
    └── archive/
        └── fallback-implementation/
            ├── PLAN.md
            ├── PROGRESS.md
            ├── FINAL_STATUS.md
            └── COMPLETE.md
```

**改善**:
- ✅ 根目錄從 20 個減少到 4-5 個
- ✅ 重複文檔合併 (20 → 12)
- ✅ 清晰的分類結構
- ✅ 完整的導覽系統
- ✅ 開發歷史歸檔

---

## 📊 執行摘要

| 階段 | Tasks | 預估時間 | Codex 使用 |
|------|-------|----------|-----------|
| Phase 1 | 1-2 | 5 分鐘 | ✅ |
| Phase 2 | 3-5 | 15 分鐘 | ✅✅✅ |
| Phase 3 | 6 | 3 分鐘 | ✅ |
| Phase 4 | 7-8 | 10 分鐘 | ✅✅ |
| Phase 5 | 9-10 | 10 分鐘 | ✅ |
| **總計** | **10** | **~45 分鐘** | **8 次** |

---

## 🚀 開始執行

準備好後,按照 Phase 順序執行:

1. **Phase 1**: 建立目錄結構
2. **Phase 2**: 整合重複文檔 (使用 codex)
3. **Phase 3**: 清理臨時文檔
4. **Phase 4**: 建立導覽系統 (使用 codex)
5. **Phase 5**: 移動檔案並提交

每個 Phase 完成後更新 TODO 狀態。
