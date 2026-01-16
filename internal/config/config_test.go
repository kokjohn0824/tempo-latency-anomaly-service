package config

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

// setDefaultLikeEnv sets a selection of env vars to their default values
// to avoid interference from the host environment.
func setDefaultLikeEnv(t *testing.T) {
    t.Helper()
    t.Setenv("TIMEZONE", DefaultTimezone)
    t.Setenv("REDIS_HOST", "127.0.0.1")
    t.Setenv("REDIS_PORT", "6379")
    t.Setenv("REDIS_PASSWORD", "")
    t.Setenv("REDIS_DB", "0")

    t.Setenv("TEMPO_URL", "http://localhost:3200")
    t.Setenv("TEMPO_AUTH_TOKEN", "")

    t.Setenv("STATS_FACTOR", "2")
    t.Setenv("STATS_K", "10")
    t.Setenv("STATS_MIN_SAMPLES", "50")
    t.Setenv("STATS_MAD_EPSILON", DefaultMadepsilon.String())

    t.Setenv("POLLING_TEMPO_INTERVAL", DefaultTempoInterval.String())
    t.Setenv("POLLING_TEMPO_LOOKBACK", DefaultTempoLookback.String())
    t.Setenv("POLLING_BASELINE_INTERVAL", DefaultBaselineInterval.String())
    t.Setenv("POLLING_BACKFILL_ENABLED", "true")
    t.Setenv("POLLING_BACKFILL_DURATION", DefaultBackfillDuration.String())
    t.Setenv("POLLING_BACKFILL_BATCH", DefaultBackfillBatch.String())

    t.Setenv("WINDOW_SIZE", "1000")
    t.Setenv("DEDUP_TTL", DefaultDedupTTL.String())

    t.Setenv("HTTP_PORT", "8080")
    t.Setenv("HTTP_TIMEOUT", DefaultHTTPTimeout.String())

    t.Setenv("FALLBACK_ENABLED", "true")
    t.Setenv("FALLBACK_NEARBY_HOURS_ENABLED", "true")
    t.Setenv("FALLBACK_NEARBY_HOURS_RANGE", "2")
    t.Setenv("FALLBACK_NEARBY_MIN_SAMPLES", "20")
    t.Setenv("FALLBACK_DAYTYPE_GLOBAL_ENABLED", "true")
    t.Setenv("FALLBACK_DAYTYPE_GLOBAL_MIN_SAMPLES", "50")
    t.Setenv("FALLBACK_FULL_GLOBAL_ENABLED", "true")
    t.Setenv("FALLBACK_FULL_GLOBAL_MIN_SAMPLES", "30")
}

func TestLoad_Defaults(t *testing.T) {
    setDefaultLikeEnv(t)

    cfg, err := Load("")
    assert.NoError(t, err)

    assert.Equal(t, DefaultTimezone, cfg.Timezone)
    assert.Equal(t, 6379, cfg.Redis.Port)
    assert.Equal(t, DefaultFactor, cfg.Stats.Factor)
    assert.Equal(t, DefaultK, cfg.Stats.K)
    assert.Equal(t, DefaultMinSamples, cfg.Stats.MinSamples)
    assert.Equal(t, DefaultMadepsilon, cfg.Stats.MADEpsilon)

    assert.Equal(t, DefaultTempoInterval, cfg.Polling.TempoInterval)
    assert.Equal(t, DefaultTempoLookback, cfg.Polling.TempoLookback)
    assert.Equal(t, DefaultBaselineInterval, cfg.Polling.BaselineInterval)
    assert.Equal(t, DefaultBackfillEnabled, cfg.Polling.BackfillEnabled)
    assert.Equal(t, DefaultBackfillDuration, cfg.Polling.BackfillDuration)
    assert.Equal(t, DefaultBackfillBatch, cfg.Polling.BackfillBatch)

    assert.Equal(t, DefaultWindowSize, cfg.WindowSize)
    assert.Equal(t, DefaultDedupTTL, cfg.Dedup.TTL)
    assert.Equal(t, DefaultHTTPPort, cfg.HTTP.Port)
    assert.Equal(t, DefaultHTTPTimeout, cfg.HTTP.Timeout)

    assert.Equal(t, DefaultFallbackEnabled, cfg.Fallback.Enabled)
    assert.Equal(t, DefaultFallbackNearbyHoursEnabled, cfg.Fallback.NearbyHoursEnabled)
    assert.Equal(t, DefaultFallbackNearbyHoursRange, cfg.Fallback.NearbyHoursRange)
    assert.Equal(t, DefaultFallbackNearbyMinSamples, cfg.Fallback.NearbyMinSamples)
    assert.Equal(t, DefaultFallbackDayTypeGlobalEnabled, cfg.Fallback.DayTypeGlobalEnabled)
    assert.Equal(t, DefaultFallbackDayTypeGlobalMinSamples, cfg.Fallback.DayTypeGlobalMinSamples)
    assert.Equal(t, DefaultFallbackFullGlobalEnabled, cfg.Fallback.FullGlobalEnabled)
    assert.Equal(t, DefaultFallbackFullGlobalMinSamples, cfg.Fallback.FullGlobalMinSamples)
}

func TestLoad_FromFileOverrides(t *testing.T) {
    // Ensure env variables do not conflict by aligning them with file values
    t.Setenv("TIMEZONE", "UTC")
    t.Setenv("REDIS_HOST", "10.0.0.1")
    t.Setenv("REDIS_PORT", "6380")
    t.Setenv("STATS_FACTOR", "3.5")
    t.Setenv("STATS_K", "7")
    t.Setenv("STATS_MIN_SAMPLES", "5")
    t.Setenv("STATS_MAD_EPSILON", "2ms")
    t.Setenv("POLLING_TEMPO_INTERVAL", "10s")
    t.Setenv("POLLING_TEMPO_LOOKBACK", "1m")
    t.Setenv("POLLING_BASELINE_INTERVAL", "2m")
    t.Setenv("POLLING_BACKFILL_ENABLED", "false")
    t.Setenv("POLLING_BACKFILL_DURATION", "24h")
    t.Setenv("POLLING_BACKFILL_BATCH", "30m")
    t.Setenv("WINDOW_SIZE", "123")
    t.Setenv("DEDUP_TTL", "3h")
    t.Setenv("HTTP_PORT", "9090")
    t.Setenv("HTTP_TIMEOUT", "20s")
    t.Setenv("FALLBACK_ENABLED", "true")
    t.Setenv("FALLBACK_NEARBY_HOURS_ENABLED", "false")
    t.Setenv("FALLBACK_NEARBY_HOURS_RANGE", "1")
    t.Setenv("FALLBACK_NEARBY_MIN_SAMPLES", "10")
    t.Setenv("FALLBACK_DAYTYPE_GLOBAL_ENABLED", "true")
    t.Setenv("FALLBACK_DAYTYPE_GLOBAL_MIN_SAMPLES", "40")
    t.Setenv("FALLBACK_FULL_GLOBAL_ENABLED", "false")
    t.Setenv("FALLBACK_FULL_GLOBAL_MIN_SAMPLES", "25")

    dir := t.TempDir()
    file := filepath.Join(dir, "config.yaml")
    yaml := []byte(`
timezone: UTC
redis:
  host: 10.0.0.1
  port: 6380
stats:
  factor: 3.5
  k: 7
  min_samples: 5
  mad_epsilon: 2ms
polling:
  tempo_interval: 10s
  tempo_lookback: 1m
  baseline_interval: 2m
  backfill_enabled: false
  backfill_duration: 24h
  backfill_batch: 30m
window_size: 123
dedup:
  ttl: 3h
http:
  port: 9090
  timeout: 20s
fallback:
  enabled: true
  nearby_hours_enabled: false
  nearby_hours_range: 1
  nearby_min_samples: 10
  daytype_global_enabled: true
  daytype_global_min_samples: 40
  full_global_enabled: false
  full_global_min_samples: 25
`)
    if err := os.WriteFile(file, yaml, 0o600); err != nil {
        t.Fatalf("write temp config: %v", err)
    }

    cfg, err := Load(file)
    assert.NoError(t, err)

    assert.Equal(t, "UTC", cfg.Timezone)
    assert.Equal(t, "10.0.0.1", cfg.Redis.Host)
    assert.Equal(t, 6380, cfg.Redis.Port)

    assert.InDelta(t, 3.5, cfg.Stats.Factor, 1e-9)
    assert.Equal(t, 7, cfg.Stats.K)
    assert.Equal(t, 5, cfg.Stats.MinSamples)
    assert.Equal(t, 2*time.Millisecond, cfg.Stats.MADEpsilon)

    assert.Equal(t, 10*time.Second, cfg.Polling.TempoInterval)
    assert.Equal(t, 1*time.Minute, cfg.Polling.TempoLookback)
    assert.Equal(t, 2*time.Minute, cfg.Polling.BaselineInterval)
    assert.Equal(t, false, cfg.Polling.BackfillEnabled)
    assert.Equal(t, 24*time.Hour, cfg.Polling.BackfillDuration)
    assert.Equal(t, 30*time.Minute, cfg.Polling.BackfillBatch)

    assert.Equal(t, 123, cfg.WindowSize)
    assert.Equal(t, 3*time.Hour, cfg.Dedup.TTL)

    assert.Equal(t, 9090, cfg.HTTP.Port)
    assert.Equal(t, 20*time.Second, cfg.HTTP.Timeout)

    assert.Equal(t, true, cfg.Fallback.Enabled)
    assert.Equal(t, false, cfg.Fallback.NearbyHoursEnabled)
    assert.Equal(t, 1, cfg.Fallback.NearbyHoursRange)
    assert.Equal(t, 10, cfg.Fallback.NearbyMinSamples)
    assert.Equal(t, true, cfg.Fallback.DayTypeGlobalEnabled)
    assert.Equal(t, 40, cfg.Fallback.DayTypeGlobalMinSamples)
    assert.Equal(t, false, cfg.Fallback.FullGlobalEnabled)
    assert.Equal(t, 25, cfg.Fallback.FullGlobalMinSamples)
}
