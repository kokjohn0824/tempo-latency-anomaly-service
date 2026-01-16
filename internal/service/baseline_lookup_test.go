package service

import (
    "context"
    "testing"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
    smocks "github.com/alexchang/tempo-latency-anomaly-service/internal/store/mocks"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func blCfg() *config.Config {
    return &config.Config{
        Timezone: "Asia/Taipei",
        Stats: config.StatsConfig{
            MinSamples: 10,
        },
        Fallback: config.FallbackConfig{
            Enabled:                 true,
            NearbyHoursEnabled:      true,
            NearbyHoursRange:        2,
            NearbyMinSamples:        10,
            DayTypeGlobalEnabled:    true,
            DayTypeGlobalMinSamples: 20,
            FullGlobalEnabled:       true,
            FullGlobalMinSamples:    15,
        },
    }
}

func TestBaselineLookup_Level1_ExactMatch(t *testing.T) {
    ctx := context.Background()
    cfg := blCfg()
    m := new(smocks.MockStore)

    svc, ep := "svcA", "GET /foo"
    bucket := domain.TimeBucket{Hour: 10, DayType: "weekday"}
    key := domain.MakeBaselineKey(svc, ep, bucket)
    now := time.Now().UTC()
    m.On("GetBaseline", mock.Anything, key).Return(&store.Baseline{P50: 1, P95: 2, MAD: 3, SampleCount: 100, UpdatedAt: now}, nil)

    bl := NewBaselineLookup(m, cfg)
    res, err := bl.LookupWithFallback(ctx, svc, ep, bucket)
    if assert.NoError(t, err) && assert.NotNil(t, res) {
        assert.Equal(t, domain.SourceExact, res.Source)
        assert.Equal(t, 1, res.FallbackLevel)
        assert.NotNil(t, res.Baseline)
        assert.Contains(t, res.SourceDetails, "exact match")
    }
    m.AssertExpectations(t)
}

func TestBaselineLookup_Level2_NearbyHoursWeighted(t *testing.T) {
    ctx := context.Background()
    cfg := blCfg()
    m := new(smocks.MockStore)

    svc, ep := "svcA", "GET /foo"
    bucket := domain.TimeBucket{Hour: 10, DayType: "weekday"}
    // Level1 miss
    m.On("GetBaseline", mock.Anything, mock.Anything).Return((*store.Baseline)(nil), nil)

    // Provide ±1 hours with different sample weights
    k9 := domain.MakeBaselineKey(svc, ep, domain.TimeBucket{Hour: 9, DayType: "weekday"})
    k11 := domain.MakeBaselineKey(svc, ep, domain.TimeBucket{Hour: 11, DayType: "weekday"})
    latest := time.Now().UTC()
    mp := map[string]*store.Baseline{
        k9:  {P50: 100, P95: 300, MAD: 10, SampleCount: 10, UpdatedAt: latest.Add(-time.Hour)},
        k11: {P50: 200, P95: 400, MAD: 20, SampleCount: 30, UpdatedAt: latest},
    }
    m.On("GetBaselines", mock.Anything, mock.Anything).Return(mp, nil)

    bl := NewBaselineLookup(m, cfg)
    res, err := bl.LookupWithFallback(ctx, svc, ep, bucket)
    if assert.NoError(t, err) && assert.NotNil(t, res) {
        assert.Equal(t, domain.SourceNearby, res.Source)
        assert.Equal(t, 2, res.FallbackLevel)
        if assert.NotNil(t, res.Baseline) {
            // Weighted by sample counts: totals = 40
            // P50 = (100*10 + 200*30) / 40 = 175
            // P95 = (300*10 + 400*30) / 40 = 375
            // MAD = (10*10 + 20*30) / 40 = 17.5
            assert.InDelta(t, 175.0, res.Baseline.P50, 1e-9)
            assert.InDelta(t, 375.0, res.Baseline.P95, 1e-9)
            assert.InDelta(t, 17.5, res.Baseline.MAD, 1e-9)
            assert.Equal(t, 40, res.Baseline.SampleCount)
            assert.Equal(t, latest, res.Baseline.UpdatedAt)
        }
        assert.Contains(t, res.SourceDetails, "nearby hours:")
    }
    m.AssertExpectations(t)
}

func TestBaselineLookup_Level3_DayTypeGlobal(t *testing.T) {
    ctx := context.Background()
    cfg := blCfg()
    // Force skip Level 2
    cfg.Fallback.NearbyHoursEnabled = false
    m := new(smocks.MockStore)

    svc, ep := "svcB", "POST /bar"
    bucket := domain.TimeBucket{Hour: 22, DayType: "weekday"}
    // Level1 miss
    m.On("GetBaseline", mock.Anything, mock.Anything).Return((*store.Baseline)(nil), nil)

    // Reply with a subset of hours with total samples >= min
    data := map[string]*store.Baseline{}
    hours := []int{1, 5, 9}
    total := 0
    latest := time.Now().UTC()
    for i, h := range hours {
        cnt := (i + 1) * 10 // 10,20,30 => 60 total
        total += cnt
        data[domain.MakeBaselineKey(svc, ep, domain.TimeBucket{Hour: h, DayType: "weekday"})] = &store.Baseline{
            P50: float64(100 + h), P95: float64(200 + h), MAD: float64(10 + i), SampleCount: cnt, UpdatedAt: latest.Add(time.Duration(i) * time.Minute),
        }
    }
    m.On("GetBaselines", mock.Anything, mock.Anything).Return(data, nil)

    bl := NewBaselineLookup(m, cfg)
    res, err := bl.LookupWithFallback(ctx, svc, ep, bucket)
    if assert.NoError(t, err) && assert.NotNil(t, res) {
        assert.Equal(t, domain.SourceDayType, res.Source)
        assert.Equal(t, 3, res.FallbackLevel)
        assert.Equal(t, total, res.Baseline.SampleCount)
        assert.Contains(t, res.SourceDetails, "daytype=weekday")
    }
    m.AssertExpectations(t)
}

func TestBaselineLookup_Level4_FullGlobal(t *testing.T) {
    ctx := context.Background()
    cfg := blCfg()
    // Skip 2 and 3
    cfg.Fallback.NearbyHoursEnabled = false
    cfg.Fallback.DayTypeGlobalEnabled = false
    m := new(smocks.MockStore)

    svc, ep := "svcC", "GET /baz"
    bucket := domain.TimeBucket{Hour: 0, DayType: "weekday"}
    // Level1 miss
    m.On("GetBaseline", mock.Anything, mock.Anything).Return((*store.Baseline)(nil), nil)

    // Supply a couple of keys across both day types
    dt := []string{"weekday", "weekend"}
    mp := map[string]*store.Baseline{}
    total := 0
    for i, d := range dt {
        k := domain.MakeBaselineKey(svc, ep, domain.TimeBucket{Hour: i, DayType: d})
        cnt := (i + 1) * 10
        total += cnt
        mp[k] = &store.Baseline{P50: float64(100 + i), P95: float64(200 + i), MAD: float64(10 + i), SampleCount: cnt, UpdatedAt: time.Now().UTC()}
    }
    m.On("GetBaselines", mock.Anything, mock.Anything).Return(mp, nil)

    bl := NewBaselineLookup(m, cfg)
    res, err := bl.LookupWithFallback(ctx, svc, ep, bucket)
    if assert.NoError(t, err) && assert.NotNil(t, res) {
        assert.Equal(t, domain.SourceGlobal, res.Source)
        assert.Equal(t, 4, res.FallbackLevel)
        assert.Equal(t, total, res.Baseline.SampleCount)
        assert.Contains(t, res.SourceDetails, "full global")
    }
    m.AssertExpectations(t)
}

func TestBaselineLookup_Level5_Unavailable(t *testing.T) {
    ctx := context.Background()
    cfg := blCfg()
    m := new(smocks.MockStore)

    svc, ep := "svcD", "GET /none"
    bucket := domain.TimeBucket{Hour: 3, DayType: "weekend"}

    // Level1 miss
    m.On("GetBaseline", mock.Anything, mock.Anything).Return((*store.Baseline)(nil), nil)
    // Level2/3/4 return empty/insufficient
    m.On("GetBaselines", mock.Anything, mock.Anything).Return(map[string]*store.Baseline{}, nil)

    bl := NewBaselineLookup(m, cfg)
    res, err := bl.LookupWithFallback(ctx, svc, ep, bucket)
    if assert.NoError(t, err) && assert.NotNil(t, res) {
        assert.Nil(t, res.Baseline)
        assert.Equal(t, domain.SourceUnavailable, res.Source)
        assert.Equal(t, 5, res.FallbackLevel)
        assert.True(t, res.CannotDetermine)
        assert.Contains(t, res.SourceDetails, "no baseline")
    }
    m.AssertExpectations(t)
}

