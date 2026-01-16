package service

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
    smocks "github.com/alexchang/tempo-latency-anomaly-service/internal/store/mocks"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func tsNanoI64(t time.Time) int64 { return t.UnixNano() }

func baseCfg() *config.Config {
    return &config.Config{
        Timezone:   "Asia/Taipei",
        WindowSize: 1000,
        Stats: config.StatsConfig{
            Factor:     1.5,
            K:          3,
            MinSamples: 10,
        },
        Dedup: config.DedupConfig{TTL: 6 * time.Hour},
        Fallback: config.FallbackConfig{
            Enabled:                 true,
            NearbyHoursEnabled:      false,
            DayTypeGlobalEnabled:    false,
            FullGlobalEnabled:       false,
            NearbyHoursRange:        2,
            NearbyMinSamples:        20,
            DayTypeGlobalMinSamples: 50,
            FullGlobalMinSamples:    30,
        },
    }
}

func TestCheck_Evaluate_NormalAndAnomaly(t *testing.T) {
    ctx := context.Background()
    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 10, 30, 0, 0, loc) // Tuesday, weekday, hour=10
    bucket, _ := domain.ParseTimeBucket(fmt.Sprintf("%d", ts.UnixNano()), "Asia/Taipei")

    svc := "svcA"
    ep := "GET /foo"
    baseKey := domain.MakeBaselineKey(svc, ep, bucket)

    // Baseline such that threshold = max(p95*factor, p50 + k*MAD) = 300
    b := &store.Baseline{P50: 100, P95: 200, MAD: 20, SampleCount: 100, UpdatedAt: time.Now().UTC()}

    m := new(smocks.MockStore)
    m.On("GetBaseline", mock.Anything, baseKey).Return(b, nil)

    bl := NewBaselineLookup(m, baseCfg())
    ck := NewCheck(m, baseCfg(), bl)

    // Case 1: within threshold
    resp, err := ck.Evaluate(ctx, domain.AnomalyCheckRequest{
        Service:       svc,
        Endpoint:      ep,
        TimestampNano: tsNanoI64(ts),
        DurationMs:    250, // < 300
    })
    if assert.NoError(t, err) {
        assert.False(t, resp.IsAnomaly)
        assert.Contains(t, resp.Explanation, "within")
        assert.Equal(t, domain.SourceExact, resp.BaselineSource)
        assert.Equal(t, 1, resp.FallbackLevel)
        assert.Equal(t, bucket, resp.Bucket)
        if assert.NotNil(t, resp.Baseline) {
            assert.Equal(t, b.SampleCount, resp.Baseline.SampleCount)
        }
    }

    // Case 2: exceeds threshold
    resp2, err := ck.Evaluate(ctx, domain.AnomalyCheckRequest{
        Service:       svc,
        Endpoint:      ep,
        TimestampNano: tsNanoI64(ts),
        DurationMs:    350, // > 300
    })
    if assert.NoError(t, err) {
        assert.True(t, resp2.IsAnomaly)
        assert.Contains(t, resp2.Explanation, "exceeds")
    }

    m.AssertExpectations(t)
}

func TestCheck_Evaluate_NoBaselineOrInsufficientSamples(t *testing.T) {
    ctx := context.Background()
    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 9, 0, 0, 0, loc) // weekday

    svc := "svcA"
    ep := "GET /foo"

    // Disable all fallbacks except final unavailable
    cfg := baseCfg()
    cfg.Fallback.NearbyHoursEnabled = false
    cfg.Fallback.DayTypeGlobalEnabled = false
    cfg.Fallback.FullGlobalEnabled = false

    m := new(smocks.MockStore)
    // Exact match returns nil to trigger fallthrough to level 5
    m.On("GetBaseline", mock.Anything, mock.Anything).Return((*store.Baseline)(nil), nil)

    bl := NewBaselineLookup(m, cfg)
    ck := NewCheck(m, cfg, bl)

    resp, err := ck.Evaluate(ctx, domain.AnomalyCheckRequest{
        Service:       svc,
        Endpoint:      ep,
        TimestampNano: tsNanoI64(ts),
        DurationMs:    123,
    })
    if assert.NoError(t, err) {
        assert.False(t, resp.IsAnomaly)
        assert.True(t, resp.CannotDetermine)
        assert.Contains(t, resp.Explanation, "no baseline available or insufficient samples")
        assert.Equal(t, 5, resp.FallbackLevel)
        assert.Equal(t, domain.SourceUnavailable, resp.BaselineSource)
    }

    m.AssertExpectations(t)
}

