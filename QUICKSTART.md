# 快速開始指南

這是 Tempo Latency Anomaly Service 的快速開始指南。

## 🚀 5 分鐘快速啟動

### 1. 啟動服務

```bash
./scripts/dev.sh up
```

等待服務啟動完成 (~30 秒)。

### 2. 驗證服務運行

```bash
curl http://localhost:8080/healthz
```

應該返回: `{"status":"ok"}`

### 3. 檢查資料收集

等待 1-2 分鐘讓系統收集資料,然後檢查:

```bash
docker exec tempo-anomaly-redis redis-cli KEYS "base:*" | wc -l
```

如果返回數字 > 0,表示已經開始收集資料。

### 4. 測試異常檢測

```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "test-service",
    "endpoint": "/test",
    "timestampNano": 1768463900000000000,
    "durationMs": 100
  }'
```

### 5. 運行完整測試

```bash
./scripts/test_final.sh
```

---

## 📋 常用指令

### 服務管理

```bash
# 啟動服務
./scripts/dev.sh up

# 停止服務
./scripts/dev.sh down

# 重啟服務
./scripts/dev.sh restart

# 查看日誌
./scripts/dev.sh logs

# 重新建構
./scripts/dev.sh build
```

### 資料檢查

```bash
# 檢查 duration keys
docker exec tempo-anomaly-redis redis-cli KEYS "dur:*"

# 檢查 baseline keys
docker exec tempo-anomaly-redis redis-cli KEYS "base:*"

# 查看特定 baseline
docker exec tempo-anomaly-redis redis-cli HGETALL "base:service|endpoint|15|weekday"

# 檢查 dirty keys
docker exec tempo-anomaly-redis redis-cli SMEMBERS dirtyKeys
```

### 日誌檢查

```bash
# 查看服務日誌
docker compose -f docker/compose.yml logs service --tail=50

# 查看 Tempo poller 日誌
docker compose -f docker/compose.yml logs service | grep "tempo poller"

# 查看 baseline 更新日誌
docker compose -f docker/compose.yml logs service | grep "baseline"
```

---

## 🔧 配置

### 修改配置

編輯 `configs/config.dev.yaml`:

```yaml
# Tempo 連接
tempo:
  url: http://192.168.4.138:3200  # 你的 Tempo URL
  auth_token: ""

# 統計參數
stats:
  factor: 1.5      # P95 乘數
  k: 3             # MAD 乘數
  min_samples: 50  # 最小樣本數
  max_samples: 500 # 最大樣本數

# 拉取頻率
polling:
  tempo_interval: 15s    # Tempo 拉取間隔
  tempo_lookback: 120s   # 拉取時間範圍
  baseline_interval: 30s # Baseline 更新間隔
```

修改後重啟服務:

```bash
./scripts/dev.sh restart
```

---

## 🧪 測試

### 快速測試

```bash
./scripts/test_simple.sh
```

### 完整測試

```bash
./scripts/test_final.sh
```

### 手動測試

**檢測正常請求**:
```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "my-service",
    "endpoint": "GET /api/users",
    "timestampNano": '$(date +%s)000000000',
    "durationMs": 50
  }'
```

**檢測異常請求**:
```bash
curl -X POST http://localhost:8080/v1/anomaly/check \
  -H "Content-Type: application/json" \
  -d '{
    "service": "my-service",
    "endpoint": "GET /api/users",
    "timestampNano": '$(date +%s)000000000',
    "durationMs": 5000
  }'
```

**查詢 baseline**:
```bash
curl "http://localhost:8080/v1/baseline?service=my-service&endpoint=GET%20%2Fapi%2Fusers&hour=15&dayType=weekday"
```

---

## 📊 監控

### 檢查系統狀態

```bash
# 檢查容器狀態
docker compose -f docker/compose.yml ps

# 檢查資料統計
echo "Duration keys: $(docker exec tempo-anomaly-redis redis-cli KEYS 'dur:*' | wc -l)"
echo "Baseline keys: $(docker exec tempo-anomaly-redis redis-cli KEYS 'base:*' | wc -l)"

# 檢查最近的 Tempo 拉取
docker compose -f docker/compose.yml logs service --tail=20 | grep "tempo poller"
```

### 查看 Metrics

```bash
curl http://localhost:8080/metrics
```

---

## 🐛 故障排除

### 問題: 服務無法啟動

**檢查**:
```bash
docker compose -f docker/compose.yml logs service
```

**常見原因**:
- Redis 未啟動
- 配置檔案錯誤
- 埠號被佔用

### 問題: 沒有資料

**檢查 Tempo 連接**:
```bash
curl http://192.168.4.138:3200/api/search?limit=1
```

**檢查服務日誌**:
```bash
docker compose -f docker/compose.yml logs service | grep "tempo poll"
```

### 問題: 總是返回 "no baseline"

**原因**: 樣本數不足

**解決**:
1. 等待更長時間 (5-10 分鐘)
2. 或降低 `stats.min_samples` 配置

---

## 📚 更多資訊

- **完整文件**: [README.md](./README.md)
- **API 範例**: [EXAMPLES.md](./EXAMPLES.md)
- **測試報告**: [TEST_REPORT.md](./TEST_REPORT.md)
- **架構說明**: [ARCHITECTURE.md](./ARCHITECTURE.md)

---

## 💡 提示

1. **初次啟動**: 需要等待 5-10 分鐘收集足夠的樣本
2. **配置調整**: 根據實際流量調整 `min_samples` 和閾值參數
3. **監控**: 定期檢查 Redis 記憶體使用和服務日誌
4. **備份**: Redis 資料可以定期備份 (RDB/AOF)

---

**需要幫助?** 查看 [README.md](./README.md) 或 [EXAMPLES.md](./EXAMPLES.md)
