# 📚 文檔導覽

歡迎使用 Tempo Latency Anomaly Service!本文檔提供完整的導覽,幫助您快速找到需要的資訊。

---

## 🚀 新使用者快速開始

建議閱讀順序:

1. **[README.md](README.md)** - 專案介紹、快速開始、配置說明
2. **[EXAMPLES.md](EXAMPLES.md)** - 完整使用範例與測試場景
3. **[docs/api/SWAGGER_GUIDE.md](docs/api/SWAGGER_GUIDE.md)** - API 文檔與 Swagger UI 使用

**5 分鐘快速體驗**:
```bash
# 1. 啟動服務
docker compose -f docker/compose.yml up -d

# 2. 訪問 Swagger UI
open http://localhost:8080/swagger/index.html

# 3. 測試 API
curl http://localhost:8080/healthz
```

---

## 📖 核心文檔

### 根目錄

| 文檔 | 說明 | 適合對象 |
|------|------|----------|
| **[README.md](README.md)** | 專案主要入口,包含安裝、配置、API 概覽 | 所有使用者 |
| **[ARCHITECTURE.md](ARCHITECTURE.md)** | 系統架構設計、資料流程、設計決策 | 開發者、架構師 |
| **[EXAMPLES.md](EXAMPLES.md)** | 完整使用範例、測試場景、故障排除 | 使用者、測試人員 |

---

## 🎨 設計文檔 (`docs/design/`)

深入了解系統設計理念與實作策略:

- **[FALLBACK_STRATEGY.md](docs/design/FALLBACK_STRATEGY.md)** - Fallback 機制設計
  - 5 層 fallback 策略 (exact → nearby → daytype → global → unavailable)
  - 設計原則與權衡
  - 效能考量與優化

---

## 🔧 功能說明 (`docs/features/`)

各功能模組的詳細說明:

- **[AVAILABLE_API.md](docs/features/AVAILABLE_API.md)** - `/v1/available` API 功能
  - 查詢可用的服務與端點
  - 實作細節與使用範例
  - 整合測試報告

---

## 📡 API 文檔 (`docs/api/`)

完整的 API 使用指南:

- **[README.md](docs/api/README.md)** - API 總覽
  - 可用端點列表
  - Swagger UI 位置
  - OpenAPI 文檔來源

- **[SWAGGER_GUIDE.md](docs/api/SWAGGER_GUIDE.md)** - Swagger 使用指南
  - 快速開始
  - API 端點詳解
  - 實作細節
  - 使用範例

**快速連結**:
- Swagger UI: http://localhost:8080/swagger/index.html
- OpenAPI JSON: http://localhost:8080/swagger/doc.json

---

## 📊 測試與報告 (`docs/reports/`)

測試結果與實作總結:

- **[TESTING.md](docs/reports/TESTING.md)** - 綜合測試報告
  - 測試摘要
  - 功能測試結果
  - Fallback 機制測試
  - 整合測試
  - 測試腳本列表

**測試腳本** (`scripts/`):
- `test_scenarios.sh` - 基本功能測試
- `test_fallback_scenarios.sh` - Fallback 機制測試
- `test_available_api.sh` - Available API 測試
- `test_swagger.sh` - Swagger UI 測試
- `test_twdiw_customer_service.sh` - 特定服務測試

---

## 📦 歷史記錄 (`docs/archive/`)

開發過程的歷史文檔,供參考:

### Fallback 實作過程 (`docs/archive/fallback-implementation/`)

- **[PLAN.md](docs/archive/fallback-implementation/PLAN.md)** - 實作計劃
- **[PROGRESS.md](docs/archive/fallback-implementation/PROGRESS.md)** - 進度報告
- **[FINAL_STATUS.md](docs/archive/fallback-implementation/FINAL_STATUS.md)** - 最終狀態
- **[COMPLETE.md](docs/archive/fallback-implementation/COMPLETE.md)** - 完成報告

> 💡 **提示**: 這些是開發過程文檔,一般使用者可以跳過。如果想了解 fallback 機制的實作歷程,可以參考這些文檔。

---

## 🗂️ 文檔結構總覽

```
/
├── README.md                    # 主要入口
├── ARCHITECTURE.md              # 系統架構
├── EXAMPLES.md                  # 使用範例
├── DOCUMENTATION_INDEX.md       # 本文檔
│
└── docs/
    ├── design/                  # 設計文檔
    │   └── FALLBACK_STRATEGY.md
    │
    ├── features/                # 功能說明
    │   └── AVAILABLE_API.md
    │
    ├── api/                     # API 文檔
    │   ├── README.md
    │   └── SWAGGER_GUIDE.md
    │
    ├── reports/                 # 測試報告
    │   └── TESTING.md
    │
    └── archive/                 # 歷史記錄
        └── fallback-implementation/
            ├── PLAN.md
            ├── PROGRESS.md
            ├── FINAL_STATUS.md
            └── COMPLETE.md
```

---

## 🔍 按需求查找文檔

### 我想...

**快速開始使用服務**
→ [README.md](README.md) → [EXAMPLES.md](EXAMPLES.md)

**了解 API 端點**
→ [docs/api/SWAGGER_GUIDE.md](docs/api/SWAGGER_GUIDE.md) 或訪問 http://localhost:8080/swagger/index.html

**了解系統架構**
→ [ARCHITECTURE.md](ARCHITECTURE.md)

**了解 Fallback 機制**
→ [README.md#fallback-strategy](README.md#fallback-strategy) → [docs/design/FALLBACK_STRATEGY.md](docs/design/FALLBACK_STRATEGY.md)

**查看測試結果**
→ [docs/reports/TESTING.md](docs/reports/TESTING.md)

**查詢可用的服務/端點**
→ [docs/features/AVAILABLE_API.md](docs/features/AVAILABLE_API.md)

**排查問題**
→ [EXAMPLES.md](EXAMPLES.md) 的故障排除章節

**了解實作細節**
→ [ARCHITECTURE.md](ARCHITECTURE.md) + [docs/design/](docs/design/)

---

## 📝 文檔維護

### 文檔更新原則

1. **核心文檔** (README, ARCHITECTURE, EXAMPLES) - 隨功能變更即時更新
2. **設計文檔** (docs/design/) - 重大設計變更時更新
3. **API 文檔** (docs/api/) - API 變更時自動生成 (Swagger)
4. **測試報告** (docs/reports/) - 重大測試後更新
5. **歷史記錄** (docs/archive/) - 僅供參考,不再更新

### 新增文檔指南

- 核心功能說明 → `docs/features/`
- 設計文檔 → `docs/design/`
- 測試報告 → `docs/reports/`
- 開發過程記錄 → `docs/archive/`

---

## 💡 提示與最佳實踐

1. **初次使用**: 從 README.md 開始,按照快速開始步驟操作
2. **開發整合**: 閱讀 ARCHITECTURE.md 了解系統設計
3. **API 測試**: 使用 Swagger UI 進行互動式測試
4. **問題排查**: 查看 EXAMPLES.md 的故障排除章節
5. **深入了解**: 閱讀 docs/design/ 下的設計文檔

---

## 🤝 貢獻指南

如果您想貢獻文檔:

1. 確保文檔放在正確的目錄
2. 更新本導覽文檔
3. 使用清晰的標題和結構
4. 提供實際的程式碼範例
5. 保持文檔的一致性

---

## 📞 需要幫助?

- 查看 [EXAMPLES.md](EXAMPLES.md) 的常見問題
- 訪問 [Swagger UI](http://localhost:8080/swagger/index.html) 測試 API
- 參考 [docs/reports/TESTING.md](docs/reports/TESTING.md) 了解測試方法

---

**最後更新**: 2026-01-15  
**文檔版本**: 1.0
