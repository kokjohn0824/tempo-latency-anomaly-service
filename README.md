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
      "bucket": { "hour": 14, "dayType": "weekday" },
      "baseline": null,
      "reason": "insufficient data: have 0 samples, need >= 50"
    }
    ```
  - Response (normal example):
    ```json
    {
      "isAnomaly": false,
      "bucket": { "hour": 14, "dayType": "weekday" },
      "baseline": { "p50": 180, "p95": 300, "mad": 12, "sampleCount": 200, "updatedAt": "2026-01-15T07:10:23Z" },
      "reason": "duration 220ms within threshold 300.00ms (p50=180.00, p95=300.00, MAD=12.00, factor=2.00, k=10)"
    }
    ```
  - Response (anomalous example):
    ```json
    {
      "isAnomaly": true,
      "bucket": { "hour": 21, "dayType": "weekend" },
      "baseline": { "p50": 200, "p95": 400, "mad": 15, "sampleCount": 500, "updatedAt": "2026-01-15T07:10:23Z" },
      "reason": "duration 900ms exceeds threshold 400.00ms (p50=200.00, p95=400.00, MAD=15.00, factor=2.00, k=10)"
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

