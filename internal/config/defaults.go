package config

import (
    "time"

    "github.com/spf13/viper"
)

const (
    DefaultTimezone = "Asia/Taipei"
    DefaultWindowSize = 1000
)

var (
    // Stats defaults
    DefaultFactor     = 2.0
    DefaultK          = 10
    DefaultMinSamples = 50
    DefaultMadepsilon = time.Millisecond

    // Polling defaults
    DefaultTempoInterval    = 15 * time.Second
    DefaultTempoLookback    = 120 * time.Second
    DefaultBaselineInterval = 30 * time.Second

    // Dedup TTL default
    DefaultDedupTTL = 6 * time.Hour

    // HTTP defaults
    DefaultHTTPPort    = 8080
    DefaultHTTPTimeout = 15 * time.Second
)

// setDefaults registers all default values on the provided viper instance.
func setDefaults(v *viper.Viper) {
    v.SetDefault("timezone", DefaultTimezone)

    v.SetDefault("redis.host", "127.0.0.1")
    v.SetDefault("redis.port", 6379)
    v.SetDefault("redis.password", "")
    v.SetDefault("redis.db", 0)

    v.SetDefault("tempo.url", "http://localhost:3200")
    v.SetDefault("tempo.auth_token", "")

    v.SetDefault("stats.factor", DefaultFactor)
    v.SetDefault("stats.k", DefaultK)
    v.SetDefault("stats.min_samples", DefaultMinSamples)
    v.SetDefault("stats.mad_epsilon", DefaultMadepsilon.String())

    v.SetDefault("polling.tempo_interval", DefaultTempoInterval.String())
    v.SetDefault("polling.tempo_lookback", DefaultTempoLookback.String())
    v.SetDefault("polling.baseline_interval", DefaultBaselineInterval.String())

    v.SetDefault("window_size", DefaultWindowSize)

    v.SetDefault("dedup.ttl", DefaultDedupTTL.String())

    v.SetDefault("http.port", DefaultHTTPPort)
    v.SetDefault("http.timeout", DefaultHTTPTimeout.String())
}

