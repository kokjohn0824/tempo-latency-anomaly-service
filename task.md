Below is a ready-to-use Markdown task prompt you can hand directly to a coding agent.
It is implementation-oriented, unambiguous, and scoped to build a production-ready service.

⸻

# Task Prompt: Time-Aware API Latency Anomaly Detection Service (Tempo-based)

## Role
You are a **Senior Backend Engineer / SRE** building a production-grade anomaly detection service for API latency using **Grafana Tempo trace data**.

Your task is to design and implement a **self-updating, time-aware latency anomaly detection service** with a clean API.

The system must be:
- Explainable (no black-box ML)
- Time-aware (latency can be normal at certain hours)
- Low-latency on the check path
- Fully automatic once deployed

---

## Problem Statement
We want to detect **abnormal API duration (latency)** based on historical behavior observed in **Tempo traces**.

Latency may be higher at certain times of day and still be normal.  
The system must account for this automatically.

---

## Core Design Principles (DO NOT VIOLATE)
1. **NO real-time percentile calculation in the check API**
2. **Baselines must be precomputed and cached**
3. **Time-awareness is mandatory**
4. **Statistical, not ML-based (no LSTM, no Isolation Forest)**
5. **Explainable decisions only**

---

## Input Data (Tempo Trace Fields)
Each trace provides at least:
```json
{
  "traceID": "string",
  "rootServiceName": "string",
  "rootTraceName": "string",
  "startTimeUnixNano": "string",
  "durationMs": number
}


⸻

Time Bucketing Rules

Convert startTimeUnixNano to local time (default: Asia/Taipei).

Extract:
	•	hourOfDay (0–23)
	•	dayType: weekday or weekend

⸻

Baseline Key

Baselines are computed per:

(service, endpoint, hourOfDay, dayType)

Where:
	•	service = rootServiceName
	•	endpoint = rootTraceName (assume already normalized)

⸻

Statistical Model (MANDATORY)

For each key, maintain a rolling window of recent durationMs values.

From this window, compute:
	•	p50 (median)
	•	p95
	•	MAD = median(|x - p50|)
	•	sampleCount

Anomaly Rule (Hybrid)

A request is anomalous if:

durationMs > max(p95 * factor, p50 + k * MAD)

Defaults:
	•	factor = 2.0
	•	k = 10
	•	minSamples = 50
	•	MAD epsilon = 1ms

⸻

Architecture Overview

The service must contain three logical components:

1. Auto-Ingest (Background)
	•	Periodically query Tempo for recent traces
	•	Deduplicate by traceID
	•	Store durationMs into rolling storage
	•	Mark corresponding baseline as “dirty”

2. Baseline Updater (Background)
	•	Periodically recompute baselines for dirty keys
	•	Store results in fast-access storage
	•	Never block API requests

3. Check API (Synchronous)
	•	Fetch cached baseline
	•	Apply anomaly rule
	•	Return decision + explanation
	•	Must be O(1)

⸻

Storage Requirements

Use Redis (or equivalent) with these logical structures:

Rolling Samples

dur:{service|endpoint|hour|dayType} -> LIST of durationMs (max N=1000)

Cached Baseline

base:{service|endpoint|hour|dayType} -> HASH
  - p50
  - p95
  - mad
  - sampleCount
  - updatedAt

Dirty Tracking

dirtyKeys -> SET of keys needing recompute


⸻

APIs to Implement

1. Ingest (Internal or Public)

POST /v1/traces/ingest

Input:

{
  "traces": [ Tempo trace objects ]
}

Behavior:
	•	Parse timestamps
	•	Derive time bucket
	•	Push duration to rolling storage
	•	Mark key dirty

⸻

2. Anomaly Check (Critical Path)

POST /v1/anomaly/check

Input:

{
  "rootServiceName": "string",
  "rootTraceName": "string",
  "startTimeUnixNano": "string",
  "durationMs": number
}

Output:

{
  "isAnomaly": boolean,
  "bucket": { "hour": number, "dayType": "weekday|weekend" },
  "baseline": {
    "p50": number,
    "p95": number,
    "mad": number,
    "sampleCount": number
  },
  "reason": "human-readable explanation"
}

Rules:
	•	If sampleCount < minSamples, return INSUFFICIENT_DATA
	•	No baseline recomputation allowed here

⸻

3. Baseline Debug (Optional but Recommended)

GET /v1/baseline

Returns current baseline stats for explainability and debugging.

⸻

Deduplication Rule

When ingesting:
	•	Use traceID
	•	Store seen IDs in Redis SET with TTL (e.g. 6 hours)
	•	Skip duplicates

⸻

Scheduling Defaults
	•	Tempo poll interval: 15s
	•	Tempo lookback window: 120s
	•	Baseline recompute interval: 30s
	•	Rolling window size: 1000 samples per key

⸻

Non-Goals (Explicitly Excluded)
	•	No ML forecasting models
	•	No long-horizon prediction
	•	No retraining pipelines
	•	No dynamic threshold learning
	•	No blocking operations in APIs

⸻

Expected Outcome

A self-running service where:
	•	Traces are ingested automatically
	•	Baselines update continuously
	•	API checks are fast and explainable
	•	Latency anomalies respect time-of-day behavior

Deliver production-quality code, not a prototype.

Expected project Structures

tempo-latency-anomaly-service/
├─ README.md
├─ go.mod
├─ go.sum
├─ docker/
│  ├─ Dockerfile
│  └─ compose.yml
├─ configs/
│  ├─ config.example.yaml
│  └─ config.dev.yaml
├─ cmd/
│  └─ server/
│     └─ main.go
├─ internal/
│  ├─ app/
│  │  ├─ app.go                # wiring: config, clients, stores, services, http, jobs
│  │  └─ lifecycle.go          # start/stop, context cancellation
│  ├─ config/
│  │  ├─ config.go             # load env/yaml
│  │  └─ defaults.go
│  ├─ api/
│  │  ├─ router.go             # chi router + middlewares
│  │  ├─ middleware.go         # logging, request id, recover
│  │  └─ handlers/
│  │     ├─ healthz.go
│  │     ├─ check.go           # POST /v1/anomaly/check
│  │     └─ baseline.go        # GET /v1/baseline
│  ├─ tempo/
│  │  ├─ client.go             # HTTP client, auth headers, retries
│  │  ├─ query.go              # build Tempo query params
│  │  └─ types.go              # structs for Tempo response (traces[])
│  ├─ domain/
│  │  ├─ model.go              # TraceEvent, TimeBucket, BaselineStats
│  │  └─ key.go                # key derivation (svc|trace|hour|dayType)
│  ├─ store/
│  │  ├─ redis/
│  │  │  ├─ client.go
│  │  │  ├─ durations.go       # dur:* LIST ops (LPUSH/LTRIM/LRANGE)
│  │  │  ├─ baseline.go        # base:* HASH ops (HGETALL/HSET)
│  │  │  ├─ dedup.go           # seen:traceID SET w/ TTL
│  │  │  └─ dirty.go           # dirtyKeys SET ops (SADD/SPOP/SSCAN)
│  │  └─ store.go              # interfaces
│  ├─ stats/
│  │  ├─ percentile.go         # p50/p95
│  │  ├─ mad.go                # MAD
│  │  └─ calculator.go         # compute BaselineStats from samples
│  ├─ service/
│  │  ├─ ingest.go             # derive key, store duration, mark dirty, dedup
│  │  ├─ check.go              # fetch baseline, apply rule, explain
│  │  └─ baseline.go           # recompute logic, thresholds, fallback
│  ├─ jobs/
│  │  ├─ tempo_poller.go        # runs every 15s, lookback 120s
│  │  └─ baseline_recompute.go  # runs every 30s, processes dirtyKeys
│  └─ observability/
│     ├─ logger.go              # zap/zerolog
│     └─ metrics.go             # Prometheus metrics (optional, but recommended)
├─ testdata/
│  └─ tempo_response.json
└─ scripts/
   └─ dev.sh

---
## Codex Execution Instructions (IMPORTANT)

This task is expected to be **long-running** and may exceed the optimal context length of a single agent.

### Execution Rule
For **each execution cycle**, you MUST:

1. Open a **NEW agent (Task Tool)**.
2. In the **current project directory**, start Codex using the following command:
   ```bash
   export TERM=xterm && codex exec "continue to next task" --full-auto

If you want next steps, the natural follow-ups would be:
- Redis schema + Lua optimization
- Percentile/MAD implementation details
- Load & cardinality safeguards
