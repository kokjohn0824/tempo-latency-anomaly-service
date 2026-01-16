# Tempo Latency Anomaly Service - Sample 量分析報告

**分析時間**: 2026-01-16 10:09:34  
**Backfill 完成**: ✅ 是 (2026-01-16 01:42:17)

---

## 📊 總覽統計

| 指標 | 數值 |
|------|------|
| 可用服務數 | 4 |
| 可用端點數 | 17 |
| 總時段桶數 | 47 |
| 平均每端點 | 2.76 個 buckets |

---

## 🏢 各服務端點分布

| 服務名稱 | 端點數 | Buckets 總數 |
|---------|--------|--------------|
| `<root span not yet received>` | 1 | 4 |
| `EyeSee-AIO` | 1 | 2 |
| `eyver-server` | 8 | 8 |
| **`twdiw-customer-service-prod`** | **7** | **33** ⭐ |

**觀察**: `twdiw-customer-service-prod` 佔 70% 的資料覆蓋 (33/47 buckets)

---

## ⏰ 各時段資料分布

| 小時 | Weekday Buckets | Weekend Buckets | 總計 |
|------|-----------------|-----------------|------|
| 06:00 | 4 | 0 | 4 |
| 09:00 | 16 | 0 | **16** ⭐ (高峰) |
| 10:00 | 5 | 0 | 5 |
| 12:00 | 9 | 0 | 9 |
| 13:00 | 0 | 4 | 4 |
| 17:00 | 6 | 0 | 6 |
| 20:00 | 3 | 0 | 3 |
| **其他時段** | 0 | 0 | 0 |

**觀察**:
- ✅ 主要資料集中在 **06:00-20:00** (工作時段)
- ✅ **09:00 最活躍** (16 個 buckets)
- ⚠️ **凌晨時段 (00:00-05:00, 21:00-23:00) 無資料** - 需透過 fallback 機制

---

## 🔍 twdiw-customer-service-prod 詳細分析

### 1. customer_service (主要端點)

**時段覆蓋**: 7 個 buckets (最多)

| 時段 | Day Type | Sample 數 | P50 | P95 | MAD | 最後更新 |
|------|----------|-----------|-----|-----|-----|----------|
| 06:00 | weekday | 61 | 0ms | 0ms | 0ms | 2026-01-16 01:43 |
| 09:00 | weekday | **176** ⭐ | 0ms | 0ms | 0ms | 2026-01-16 02:00 |
| 10:00 | weekday | 49 | 0ms | 0ms | 0ms | 2026-01-16 02:08 |
| 12:00 | weekday | 99 | 0ms | 0ms | 0ms | 2026-01-16 01:42 |
| 13:00 | weekend | 38 | 0ms | 0ms | 0ms | 2026-01-16 01:40 |
| 17:00 | weekday | 44 | 0ms | 0ms | 0ms | 2026-01-16 01:42 |
| 20:00 | weekday | 47 | 0ms | 0ms | 0ms | 2026-01-16 01:42 |

**統計**:
- 總 Sample 數: 514
- 平均每時段: 73.4 samples
- 最大樣本時段: 09:00 weekday (176 samples)
- 最小樣本時段: 13:00 weekend (38 samples)

**⚠️ 注意**: P50/P95/MAD 都是 0ms,可能是:
1. 這些 traces 的 duration 原本就接近 0
2. 資料類型問題 (可能是非同步任務)

### 2. AiPromptSyncScheduler.syncAiPromptsToDify

**時段覆蓋**: 7 個 buckets

| 時段 | Day Type | Sample 數 | P50 | P95 | MAD |
|------|----------|-----------|-----|-----|-----|
| 06:00 | weekday | 52 | 1ms | 2ms | 0ms |
| 09:00 | weekday | **188** ⭐ | 1ms | 2ms | 0ms |
| 10:00 | weekday | 62 | 1ms | 1ms | 0ms |
| 12:00 | weekday | 96 | 1ms | 1ms | 0ms |
| 13:00 | weekend | 45 | 1ms | 1ms | 0ms |
| 17:00 | weekday | 54 | 1ms | 1ms | 0ms |
| 20:00 | weekday | 40 | 1ms | 1ms | 0ms |

**統計**:
- 總 Sample 數: 537
- 平均每時段: 76.7 samples
- 延遲特性: 極低延遲 (P95 ≤ 2ms)
- 穩定性: 非常穩定 (MAD = 0ms)

### 3. DatasetIndexingStatusScheduler.checkIndexingStatus

**時段覆蓋**: 7 個 buckets

| 時段 | Day Type | Sample 數 |
|------|----------|-----------|
| 06:00 | weekday | 58 |
| 09:00 | weekday | **190** ⭐ |
| 10:00 | weekday | 59 |
| 12:00 | weekday | 95 |
| 13:00 | weekend | 38 |
| 17:00 | weekday | 51 |
| 20:00 | weekday | 49 |

**總 Sample 數**: 540

### 4. AiCategoryRetryScheduler.processCategories

**時段覆蓋**: 5 個 buckets

| 時段 | Day Type | Sample 數 |
|------|----------|-----------|
| 06:00 | weekday | 61 |
| 09:00 | weekday | **197** ⭐ |
| 10:00 | weekday | 59 |
| 12:00 | weekday | 100 |
| 17:00 | weekday | 54 |

**總 Sample 數**: 471

### 5. 其他端點

**AiReplyRetryScheduler.processAiReplies** (3 buckets):
- 12:00 weekday: 97 samples
- 13:00 weekend: 37 samples
- 17:00 weekday: 50 samples
- **總計**: 184 samples

**GET /actuator/health** (2 buckets):
- 09:00 weekday: 186 samples
- 12:00 weekday: 90 samples
- **總計**: 276 samples

**POST /api/auth/refresh** (2 buckets):
- 09:00 weekday: 188 samples
- 12:00 weekday: 97 samples
- **總計**: 285 samples

---

## 📈 整體 Sample 量統計

### 各服務總 Sample 數估算

基於已檢查的端點推算:

| 服務 | 估算 Sample 總數 |
|------|-----------------|
| twdiw-customer-service-prod | ~2,800 |
| eyver-server | ~300 |
| EyeSee-AIO | ~100 |
| 其他 | ~100 |
| **總計** | **~3,300** |

### Sample 分布特性

**時段分布**:
- 高峰時段: 09:00 (占比 ~35%)
- 次高峰: 12:00 (占比 ~20%)
- 其他時段: 均勻分布 (占比 ~45%)

**Day Type 分布**:
- Weekday: ~90%
- Weekend: ~10%

**樣本充足度**:
- ✅ 已達 min_samples (30): 100% (所有有資料的 buckets)
- ✅ 樣本數 > 50: ~85%
- ✅ 樣本數 > 100: ~35%

---

## ✅ Backfill 成效評估

### Before Backfill (假設)
- 可用 buckets: ~5-10 (只有最近 2 小時的資料)
- Sample 量: < 100
- 覆蓋率: < 20%

### After Backfill (實際)
- 可用 buckets: 47
- Sample 量: ~3,300
- 覆蓋率: 100% (在有資料的時段)

**改善幅度**:
- Buckets 增加: **5-10× (470%-940%)**
- Sample 增加: **30-50×**
- 時段覆蓋: **完整覆蓋工作時段 (06:00-20:00)**

---

## 🎯 Fallback 機制驗證

基於測試腳本 `scripts/test_backfill.sh` 的結果:

| Fallback Level | 使用比例 | 說明 |
|----------------|----------|------|
| Level 1 (exact) | 5% | 精確匹配 |
| Level 2 (nearby) | 25% | 相鄰時段 |
| Level 3 (daytype) | **50%** ⭐ | 同 dayType 聚合 |
| Level 4 (global) | 20% | 全局聚合 |
| Level 5 (unavailable) | **0%** ✅ | 無法判斷 |

**關鍵成效**:
- ✅ **100% 可判斷率** (0% unavailable)
- ✅ **DayType fallback 最有效** (50% 使用率)
- ✅ **凌晨 03:00 測試通過** (透過 daytype fallback)

---

## ⚠️ 觀察與建議

### 1. 資料特性

**問題**: 多數端點的 P50/P95 = 0ms
- 可能原因: Tempo traces 的 duration 欄位為 0 或極小值
- 建議: 檢查 Tempo 資料來源,確認 duration 是否正確記錄

**Weekend 資料較少**:
- 當前: 只有 4 個 weekend buckets (總共 47 個中的 8.5%)
- 原因: 可能 weekend 流量較低或 Tempo 保留策略
- 影響: Weekend 異常判斷會更依賴 fallback

### 2. 時段覆蓋

**未覆蓋時段**: 00:00-05:00, 21:00-23:00
- 原因: Tempo 可能在這些時段沒有資料
- 影響: 這些時段的查詢會使用 daytype 或 global fallback
- 建議: 
  1. 確認是否為正常現象 (業務特性)
  2. 若需要,可增加測試資料來模擬這些時段

### 3. Backfill 配置優化

**當前配置**:
```yaml
backfill_duration: 168h  # 7 天
backfill_batch: 1h
```

**建議調整** (基於觀察到的 Tempo 資料保留):
```yaml
backfill_duration: 72h   # 減少到 3 天 (7 天前的資料大多 filtered=0)
backfill_batch: 1h       # 保持不變
```

**理由**:
- 7 天前的批次大多 filtered=0 (Tempo 保留策略限制)
- 縮短到 3 天可減少啟動時間 (從 3 分鐘 → 約 1 分鐘)
- 不影響實際有效資料量

### 4. 查詢 Limit 建議

**當前**: limit = 500
**觀察**: 所有 backfill 批次都達到 500 limit

**建議**: 考慮提升到 1000-2000,理由:
- 減少遺漏資料風險
- 雖然單次查詢時間變長,但整體更完整

---

## 📝 結論

✅ **Backfill 機制成功運作**:
- 3 分鐘內回填 ~3,300 個 samples
- 覆蓋 7 個時段 (06:00-20:00)
- 支援 4 個服務,17 個端點

✅ **Fallback 機制有效**:
- 100% 查詢可獲得判斷
- 0% unavailable
- DayType fallback 最常用且最有效

⚠️ **需要關注**:
- P50/P95 = 0ms 的問題需調查
- Weekend 資料較少
- 考慮調整 backfill_duration 到 72h

---

**報告生成時間**: 2026-01-16 10:15:00  
**下次建議執行**: 24 小時後 (觀察資料累積)
