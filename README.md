# Tempo Latency Anomaly Service

Time-aware, explainable API latency anomaly detection using Grafana Tempo traces. Baselines are precomputed per time bucket (hour-of-day × weekday/weekend) and checked in O(1) on the request path.

Key properties:
- Explainable decisions (no black-box ML)
- Time-aware thresholds per hour and day type
- Fast check path (reads cached baselines only)
- Self-updating via background jobs (ingest + recompute)

See `ARCHITECTURE.md` for a detailed design and data flows.

## Quickstart

- Prerequisites: Go 1.22+, Redis 7+, optional Tempo endpoint

- Run locally (Go):
  1) Copy or edit `configs/config.dev.yaml`
  2) Start Redis (e.g., `docker run -p 6379:6379 redis:7-alpine`)
  3) `go run ./cmd/server -config configs/config.dev.yaml`

- Run with Docker Compose:
  - `scripts/dev.sh up` (builds service and starts Redis)
  - Service: `http://localhost:8080`

Health check: `GET /healthz`

Metrics (basic Prometheus format): `GET /metrics`

**Swagger UI**: `http://localhost:8080/swagger/index.html` - 互動式 API 文檔和測試介面

## Configuration

Config file example (`configs/config.example.yaml`):

```
timezone: Asia/Taipei

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0

tempo:
  url: http://localhost:3200
  auth_token: ""

stats:
  factor: 2.0
  k: 10
  min_samples: 50
  mad_epsilon: 1ms

polling:
  tempo_interval: 15s
  tempo_lookback: 120s
  baseline_interval: 30s

window_size: 1000

dedup:
  ttl: 6h

http:
  port: 8080
  timeout: 15s
```

Environment variables override file values (dot → underscore):
- `TIMEZONE`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`
- `TEMPO_URL`, `TEMPO_AUTH_TOKEN`
- `STATS_FACTOR`, `STATS_K`, `STATS_MIN_SAMPLES`, `STATS_MAD_EPSILON`
- `POLLING_TEMPO_INTERVAL`, `POLLING_TEMPO_LOOKBACK`, `POLLING_BASELINE_INTERVAL`
- `WINDOW_SIZE`, `DEDUP_TTL`, `HTTP_PORT`, `HTTP_TIMEOUT`

You can also pass a config file path via `-config` flag or `CONFIG_FILE` env var.

## Fallback Strategy

當前服務在查詢 baseline 時，採用 5 層 fallback 機制以最大化可用性並保持可解釋性。若上一層無法取得足夠樣本或被停用，才會往下一層嘗試。

Levels 概覽：

1) Level 1 — 精確時段 (exact): 使用當前請求所屬的 `hour|dayType` 精確 baseline。
   - 觸發條件: 該 bucket 存在且樣本數 ≥ `stats.min_samples`。
   - 配置參數: `stats.min_samples`。

2) Level 2 — 相鄰時段 (nearby): 於相同 `dayType` 下，聚合 ±N 小時內的可用 baseline，按樣本數加權平均。
   - 觸發條件: Level 1 失敗，且 `fallback.nearby_hours_enabled: true`，聚合後總樣本數 ≥ `fallback.nearby_min_samples`。
   - 配置參數: `fallback.nearby_hours_enabled`, `fallback.nearby_hours_range`, `fallback.nearby_min_samples`。

3) Level 3 — 類型天全局 (daytype): 聚合同一 `dayType` 下 24 個小時的 baseline，按樣本數加權平均。
   - 觸發條件: Level 1、2 失敗，且 `fallback.daytype_global_enabled: true`，總樣本數 ≥ `fallback.daytype_global_min_samples`。
   - 配置參數: `fallback.daytype_global_enabled`, `fallback.daytype_global_min_samples`。

4) Level 4 — 完全全局 (global): 聚合兩種 `dayType` 下所有小時的 baseline，按樣本數加權平均。
   - 觸發條件: Level 1–3 失敗，且 `fallback.full_global_enabled: true`，總樣本數 ≥ `fallback.full_global_min_samples`。
   - 配置參數: `fallback.full_global_enabled`, `fallback.full_global_min_samples`。

5) Level 5 — 無法判斷 (unavailable): 無任何可用 baseline，可回應 `cannotDetermine=true` 並解釋原因。

對應的來源標記會透過回應欄位呈現：`baselineSource` ∈ {`exact`,`nearby`,`daytype`,`global`,`unavailable`}, `fallbackLevel` ∈ {1..5}, 並附帶 `sourceDetails`。

Fallback 相關配置示例 (加入到 config 檔的 `fallback:` 區段)：

```
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

使用範例 — 各層回應示意：

- Level 1 (exact):
  ```json
  {
    "isAnomaly": false,
    "bucket": { "hour": 14, "dayType": "weekday" },
    "baselineSource": "exact",
    "fallbackLevel": 1,
    "sourceDetails": "exact match: 14|weekday",
    "explanation": "duration 220ms within threshold 300.00ms ..."
  }
  ```

- Level 2 (nearby):
  ```json
  {
    "isAnomaly": false,
    "bucket": { "hour": 3, "dayType": "weekday" },
    "baselineSource": "nearby",
    "fallbackLevel": 2,
    "sourceDetails": "nearby hours: 2,4,1,5 (weekday)",
    "explanation": "duration 260ms within threshold 340.00ms ..."
  }
  ```

- Level 3 (daytype):
  ```json
  {
    "isAnomaly": false,
    "bucket": { "hour": 22, "dayType": "weekend" },
    "baselineSource": "daytype",
    "fallbackLevel": 3,
    "sourceDetails": "daytype=weekend hours=0,1,2,...,23",
    "explanation": "duration 310ms within threshold 400.00ms ..."
  }
  ```

- Level 4 (global):
  ```json
  {
    "isAnomaly": false,
    "bucket": { "hour": 5, "dayType": "weekday" },
    "baselineSource": "global",
    "fallbackLevel": 4,
    "sourceDetails": "full global across all hours/daytypes",
    "explanation": "duration 290ms within threshold 380.00ms ..."
  }
  ```

- Level 5 (unavailable):
  ```json
  {
    "isAnomaly": false,
    "cannotDetermine": true,
    "bucket": { "hour": 11, "dayType": "weekday" },
    "baseline": null,
    "baselineSource": "unavailable",
    "fallbackLevel": 5,
    "sourceDetails": "no baseline data available",
    "explanation": "no baseline available or insufficient samples (have 0, need >= 50)"
  }
  ```

## APIs

- GET `/healthz`: Service liveness check.

- POST `/v1/anomaly/check`
  - Request body:
    ```json
    {
      "rootServiceName": "api-gateway",
      "rootTraceName": "/users/profile",
      "startTimeUnixNano": "1736928000000000000",
      "durationMs": 320
    }
    ```
  - Response (insufficient data):
    ```json
    {
      "isAnomaly": false,
      "cannotDetermine": true,
      "bucket": { "hour": 14, "dayType": "weekday" },
      "baseline": null,
      "baselineSource": "unavailable",
      "fallbackLevel": 5,
      "sourceDetails": "no baseline data available",
      "explanation": "no baseline available or insufficient samples (have 0, need >= 50)"
    }
    ```
  - Response (normal example):
    ```json
    {
      "isAnomaly": false,
      "bucket": { "hour": 14, "dayType": "weekday" },
      "baseline": { "p50": 180, "p95": 300, "mad": 12, "sampleCount": 200, "updatedAt": "2026-01-15T07:10:23Z" },
      "baselineSource": "exact",
      "fallbackLevel": 1,
      "sourceDetails": "exact match: 14|weekday",
      "explanation": "duration 220ms within threshold 300.00ms (p50=180.00, p95=300.00, MAD=12.00, factor=2.00, k=10)"
    }
    ```
  - Response (anomalous example):
    ```json
    {
      "isAnomaly": true,
      "bucket": { "hour": 21, "dayType": "weekend" },
      "baseline": { "p50": 200, "p95": 400, "mad": 15, "sampleCount": 500, "updatedAt": "2026-01-15T07:10:23Z" },
      "baselineSource": "daytype",
      "fallbackLevel": 3,
      "sourceDetails": "daytype=weekend hours=...",
      "explanation": "duration 900ms exceeds threshold 400.00ms (p50=200.00, p95=400.00, MAD=15.00, factor=2.00, k=10)"
    }
    ```

- GET `/v1/baseline?service=api-gateway&endpoint=%2Fusers%2Fprofile&hour=14&dayType=weekday`
  - Response when found:
    ```json
    { "p50": 180, "p95": 300, "mad": 12, "sampleCount": 200, "updatedAt": "2026-01-15T07:10:23Z" }
    ```
  - Response when missing: `404` with `{ "error": "not found" }`

- GET `/v1/available`: List all services and endpoints with sufficient baseline data
  - Response:
    ```json
    {
      "totalServices": 3,
      "totalEndpoints": 15,
      "services": [
        {
          "service": "twdiw-customer-service-prod",
          "endpoint": "GET /actuator/health",
          "buckets": ["16|weekday", "17|weekday"]
        },
        {
          "service": "CHT_aiops",
          "endpoint": "OpenApiPmSchedule.pmDbSchedule",
          "buckets": ["16|weekday"]
        }
      ]
    }
    ```
  - Use this API to discover which services/endpoints are ready for anomaly detection

## Background Jobs

- Tempo poller: every `polling.tempo_interval` (default 15s), queries last `polling.tempo_lookback` seconds (default 120s), deduplicates by traceID, stores durations, marks keys dirty.
- Baseline recompute: every `polling.baseline_interval` (default 30s), pops dirty keys in batches, recomputes p50/p95/MAD/sampleCount, updates cache.

## Data Model & Keys

- Rolling samples: `dur:{service}|{endpoint}|{hour}|{dayType}` → Redis LIST (max `window_size`)
- Baseline cache: `base:{service}|{endpoint}|{hour}|{dayType}` → Redis HASH
- Dedup: `seen:{traceID}` → STRING with TTL
- Dirty tracking: `dirtyKeys` → SET

## Troubleshooting

- Redis connection: ensure `REDIS_HOST`/`REDIS_PORT` are reachable from the service container or Go process.
- Tempo URL: set `TEMPO_URL` (and `TEMPO_AUTH_TOKEN` if required) to a reachable Tempo instance. Sample Tempo JSON lives in `testdata/tempo_response.json`.
- Ports: adjust `HTTP_PORT` if `8080` is occupied.
