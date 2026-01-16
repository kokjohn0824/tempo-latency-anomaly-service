package config

import (
    "fmt"
    "strings"
    "time"

    "github.com/spf13/viper"
    "github.com/mitchellh/mapstructure"
)

// Config represents the full application configuration.
type Config struct {
    Timezone     string         `mapstructure:"timezone" yaml:"timezone"`
    Redis        RedisConfig    `mapstructure:"redis" yaml:"redis"`
    Tempo        TempoConfig    `mapstructure:"tempo" yaml:"tempo"`
    Stats        StatsConfig    `mapstructure:"stats" yaml:"stats"`
    Polling      PollingConfig  `mapstructure:"polling" yaml:"polling"`
    WindowSize   int            `mapstructure:"window_size" yaml:"window_size"`
    Dedup        DedupConfig    `mapstructure:"dedup" yaml:"dedup"`
    HTTP         HTTPConfig     `mapstructure:"http" yaml:"http"`
    Fallback     FallbackConfig `mapstructure:"fallback" yaml:"fallback"`
}

type RedisConfig struct {
    Host     string `mapstructure:"host" yaml:"host"`
    Port     int    `mapstructure:"port" yaml:"port"`
    Password string `mapstructure:"password" yaml:"password"`
    DB       int    `mapstructure:"db" yaml:"db"`
}

type TempoConfig struct {
    URL       string `mapstructure:"url" yaml:"url"`
    AuthToken string `mapstructure:"auth_token" yaml:"auth_token"`
}

type StatsConfig struct {
    Factor      float64       `mapstructure:"factor" yaml:"factor"`
    K           int           `mapstructure:"k" yaml:"k"`
    MinSamples  int           `mapstructure:"min_samples" yaml:"min_samples"`
    MADEpsilon  time.Duration `mapstructure:"mad_epsilon" yaml:"mad_epsilon"`
}

type PollingConfig struct {
    TempoInterval    time.Duration `mapstructure:"tempo_interval" yaml:"tempo_interval"`
    TempoLookback    time.Duration `mapstructure:"tempo_lookback" yaml:"tempo_lookback"`
    BaselineInterval time.Duration `mapstructure:"baseline_interval" yaml:"baseline_interval"`
    BackfillEnabled  bool          `mapstructure:"backfill_enabled" yaml:"backfill_enabled"`
    BackfillDuration time.Duration `mapstructure:"backfill_duration" yaml:"backfill_duration"`
    BackfillBatch    time.Duration `mapstructure:"backfill_batch" yaml:"backfill_batch"`
}

type DedupConfig struct {
    TTL time.Duration `mapstructure:"ttl" yaml:"ttl"`
}

type HTTPConfig struct {
    Port    int           `mapstructure:"port" yaml:"port"`
    Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
}

type FallbackConfig struct {
    Enabled                  bool `mapstructure:"enabled" yaml:"enabled"`
    NearbyHoursEnabled       bool `mapstructure:"nearby_hours_enabled" yaml:"nearby_hours_enabled"`
    NearbyHoursRange         int  `mapstructure:"nearby_hours_range" yaml:"nearby_hours_range"`
    NearbyMinSamples         int  `mapstructure:"nearby_min_samples" yaml:"nearby_min_samples"`
    DayTypeGlobalEnabled     bool `mapstructure:"daytype_global_enabled" yaml:"daytype_global_enabled"`
    DayTypeGlobalMinSamples  int  `mapstructure:"daytype_global_min_samples" yaml:"daytype_global_min_samples"`
    FullGlobalEnabled        bool `mapstructure:"full_global_enabled" yaml:"full_global_enabled"`
    FullGlobalMinSamples     int  `mapstructure:"full_global_min_samples" yaml:"full_global_min_samples"`
}

// Load reads configuration from a YAML file (if provided) and environment variables.
// - filePath: optional path to a YAML config file. If empty, it will search common locations.
// Environment variables override file/defaults automatically. Example env vars:
//   REDIS_HOST, REDIS_PORT, TEMPO_URL, TEMPO_AUTH_TOKEN, TIMEZONE,
//   STATS_FACTOR, STATS_K, STATS_MIN_SAMPLES, STATS_MAD_EPSILON,
//   POLLING_TEMPO_INTERVAL, POLLING_TEMPO_LOOKBACK, POLLING_BASELINE_INTERVAL,
//   WINDOW_SIZE, DEDUP_TTL, HTTP_PORT, HTTP_TIMEOUT
func Load(filePath string) (*Config, error) {
    v := viper.New()

    // Defaults
    setDefaults(v)

    // Env overrides
    v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
    v.AutomaticEnv()

    // Config file (optional)
    if filePath != "" {
        v.SetConfigFile(filePath)
    } else {
        v.SetConfigName("config")
        v.SetConfigType("yaml")
        v.AddConfigPath(".")
        v.AddConfigPath("./configs")
    }

    if err := v.ReadInConfig(); err != nil {
        // If a file was specified explicitly, error out; otherwise continue with env + defaults
        if filePath != "" {
            return nil, fmt.Errorf("read config file: %w", err)
        }
    }

    var cfg Config
    decoder := func(c *mapstructure.DecoderConfig) {
        c.TagName = "mapstructure"
        c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
            mapstructure.StringToTimeDurationHookFunc(),
        )
    }

    if err := v.Unmarshal(&cfg, decoder); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    return &cfg, nil
}

// MustLoad is a helper that panics on error.
func MustLoad(filePath string) *Config {
    cfg, err := Load(filePath)
    if err != nil {
        panic(err)
    }
    return cfg
}

