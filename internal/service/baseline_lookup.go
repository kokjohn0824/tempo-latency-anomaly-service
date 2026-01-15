package service

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// BaselineLookup provides baseline retrieval with multi-level fallback strategy.
type BaselineLookup struct {
    store store.Store
    cfg   *config.Config
}

// NewBaselineLookup constructs a new BaselineLookup service.
func NewBaselineLookup(store store.Store, cfg *config.Config) *BaselineLookup {
    return &BaselineLookup{store: store, cfg: cfg}
}

// BaselineResult represents the outcome of a baseline lookup with fallback.
type BaselineResult struct {
    Baseline        *store.Baseline
    Source          domain.BaselineSource
    FallbackLevel   int
    SourceDetails   string
    CannotDetermine bool
}

// LookupWithFallback attempts to find an appropriate baseline using the configured
// multi-level fallback flow. This defines only the main control flow; individual
// level try* methods are stubbed for later implementation.
func (bl *BaselineLookup) LookupWithFallback(
    ctx context.Context,
    service, endpoint string,
    bucket domain.TimeBucket,
) (*BaselineResult, error) {
    // Level 1: Exact hour | dayType match
    if res := bl.tryExactMatch(ctx, service, endpoint, bucket); res != nil {
        return res, nil
    }

    // Level 2: Nearby hours within configured range
    if bl.cfg != nil && bl.cfg.Fallback.NearbyHoursEnabled {
        if res := bl.tryNearbyHours(ctx, service, endpoint, bucket); res != nil {
            return res, nil
        }
    }

    // Level 3: Day type global (all hours for same day type)
    if bl.cfg != nil && bl.cfg.Fallback.DayTypeGlobalEnabled {
        if res := bl.tryDayTypeGlobal(ctx, service, endpoint, bucket.DayType); res != nil {
            return res, nil
        }
    }

    // Level 4: Full global (all data, any hour/dayType)
    if bl.cfg != nil && bl.cfg.Fallback.FullGlobalEnabled {
        if res := bl.tryFullGlobal(ctx, service, endpoint); res != nil {
            return res, nil
        }
    }

    // Level 5: No data available
    return &BaselineResult{
        Baseline:        nil,
        Source:          domain.SourceUnavailable,
        FallbackLevel:   5,
        SourceDetails:   "no baseline data available",
        CannotDetermine: true,
    }, nil
}

// tryExactMatch attempts Level 1 exact match baseline: base:{service}|{endpoint}|{hour}|{dayType}
func (bl *BaselineLookup) tryExactMatch(ctx context.Context, service, endpoint string, bucket domain.TimeBucket) *BaselineResult {
    if bl == nil || bl.store == nil || bl.cfg == nil {
        return nil
    }

    key := domain.MakeBaselineKey(service, endpoint, bucket)
    b, err := bl.store.GetBaseline(ctx, key)
    if err != nil || b == nil {
        return nil
    }
    if b.SampleCount < bl.cfg.Stats.MinSamples {
        return nil
    }
    detail := fmt.Sprintf("exact match: %d|%s", bucket.Hour, bucket.DayType)
    return &BaselineResult{
        Baseline:      b,
        Source:        domain.SourceExact,
        FallbackLevel: 1,
        SourceDetails: detail,
    }
}

// tryNearbyHours attempts Level 2 nearby hours aggregation within configured ±range.
func (bl *BaselineLookup) tryNearbyHours(ctx context.Context, service, endpoint string, bucket domain.TimeBucket) *BaselineResult {
    if bl == nil || bl.store == nil || bl.cfg == nil {
        return nil
    }

    r := bl.cfg.Fallback.NearbyHoursRange
    if r <= 0 {
        return nil
    }

    // Generate neighbor hours in the sequence: ±1, ±2, ... within [0,23] with wrap-around
    type hk struct {
        key  string
        hour int
    }
    var keys []hk
    seen := make(map[int]bool)
    center := bucket.Hour
    wrap := func(h int) int { return (h%24 + 24) % 24 }
    for d := 1; d <= r; d++ {
        for _, h := range []int{wrap(center - d), wrap(center + d)} {
            if seen[h] {
                continue
            }
            seen[h] = true
            k := domain.MakeBaselineKey(service, endpoint, domain.TimeBucket{Hour: h, DayType: bucket.DayType})
            keys = append(keys, hk{key: k, hour: h})
        }
    }
    if len(keys) == 0 {
        return nil
    }

    // Batch fetch
    rawKeys := make([]string, 0, len(keys))
    for _, it := range keys {
        rawKeys = append(rawKeys, it.key)
    }
    m, err := bl.store.GetBaselines(ctx, rawKeys)
    if err != nil || len(m) == 0 {
        return nil
    }

    // Aggregate weighted by sample count
    var totalSamples int
    var sumP50, sumP95, sumMAD float64
    var latest time.Time
    var usedHours []int
    for _, it := range keys { // keep order as generated (±1, then ±2, ...)
        b := m[it.key]
        if b == nil || b.SampleCount <= 0 {
            continue
        }
        totalSamples += b.SampleCount
        sumP50 += b.P50 * float64(b.SampleCount)
        sumP95 += b.P95 * float64(b.SampleCount)
        sumMAD += b.MAD * float64(b.SampleCount)
        if b.UpdatedAt.After(latest) {
            latest = b.UpdatedAt
        }
        usedHours = append(usedHours, it.hour)
    }
    if totalSamples < bl.cfg.Fallback.NearbyMinSamples {
        return nil
    }
    if totalSamples == 0 {
        return nil
    }

    // Details: list used hours in order, compact with comma
    // Sort hours by closeness to center for nicer details (1, -1, 2, -2 order already maintained)
    // but ensure deterministic formatting: 0-23 ascending by our generated order
    detailParts := make([]string, 0, len(usedHours))
    for _, h := range usedHours {
        detailParts = append(detailParts, fmt.Sprintf("%d", h))
    }
    details := fmt.Sprintf("nearby hours: %s (%s)", strings.Join(detailParts, ","), bucket.DayType)

    agg := &store.Baseline{
        P50:         sumP50 / float64(totalSamples),
        P95:         sumP95 / float64(totalSamples),
        MAD:         sumMAD / float64(totalSamples),
        SampleCount: totalSamples,
        UpdatedAt:   latest,
    }
    return &BaselineResult{
        Baseline:      agg,
        Source:        domain.SourceNearby,
        FallbackLevel: 2,
        SourceDetails: details,
    }
}

// tryDayTypeGlobal attempts Level 3 day-type global aggregation (all hours of same day type).
func (bl *BaselineLookup) tryDayTypeGlobal(ctx context.Context, service, endpoint string, dayType string) *BaselineResult {
    if bl == nil || bl.store == nil || bl.cfg == nil {
        return nil
    }

    // Build all 24 hour keys for the given dayType
    rawKeys := make([]string, 0, 24)
    for h := 0; h < 24; h++ {
        rawKeys = append(rawKeys, domain.MakeBaselineKey(service, endpoint, domain.TimeBucket{Hour: h, DayType: dayType}))
    }
    m, err := bl.store.GetBaselines(ctx, rawKeys)
    if err != nil || len(m) == 0 {
        return nil
    }

    var totalSamples int
    var sumP50, sumP95, sumMAD float64
    var latest time.Time
    var usedHours []int
    for h := 0; h < 24; h++ {
        k := domain.MakeBaselineKey(service, endpoint, domain.TimeBucket{Hour: h, DayType: dayType})
        b := m[k]
        if b == nil || b.SampleCount <= 0 {
            continue
        }
        totalSamples += b.SampleCount
        sumP50 += b.P50 * float64(b.SampleCount)
        sumP95 += b.P95 * float64(b.SampleCount)
        sumMAD += b.MAD * float64(b.SampleCount)
        if b.UpdatedAt.After(latest) {
            latest = b.UpdatedAt
        }
        usedHours = append(usedHours, h)
    }
    if totalSamples < bl.cfg.Fallback.DayTypeGlobalMinSamples {
        return nil
    }
    if totalSamples == 0 {
        return nil
    }

    // For readability, sort used hours ascending
    sort.Ints(usedHours)
    parts := make([]string, 0, len(usedHours))
    for _, h := range usedHours {
        parts = append(parts, fmt.Sprintf("%d", h))
    }
    details := fmt.Sprintf("daytype=%s hours=%s", dayType, strings.Join(parts, ","))

    agg := &store.Baseline{
        P50:         sumP50 / float64(totalSamples),
        P95:         sumP95 / float64(totalSamples),
        MAD:         sumMAD / float64(totalSamples),
        SampleCount: totalSamples,
        UpdatedAt:   latest,
    }
    return &BaselineResult{
        Baseline:      agg,
        Source:        domain.SourceDayType,
        FallbackLevel: 3,
        SourceDetails: details,
    }
}

// tryFullGlobal attempts Level 4 full global aggregation (all data for service/endpoint).
func (bl *BaselineLookup) tryFullGlobal(ctx context.Context, service, endpoint string) *BaselineResult {
    if bl == nil || bl.store == nil || bl.cfg == nil {
        return nil
    }

    dayTypes := []string{"weekday", "weekend"}
    rawKeys := make([]string, 0, 48)
    for _, dt := range dayTypes {
        for h := 0; h < 24; h++ {
            rawKeys = append(rawKeys, domain.MakeBaselineKey(service, endpoint, domain.TimeBucket{Hour: h, DayType: dt}))
        }
    }
    m, err := bl.store.GetBaselines(ctx, rawKeys)
    if err != nil || len(m) == 0 {
        return nil
    }

    var totalSamples int
    var sumP50, sumP95, sumMAD float64
    var latest time.Time
    for _, k := range rawKeys {
        b := m[k]
        if b == nil || b.SampleCount <= 0 {
            continue
        }
        totalSamples += b.SampleCount
        sumP50 += b.P50 * float64(b.SampleCount)
        sumP95 += b.P95 * float64(b.SampleCount)
        sumMAD += b.MAD * float64(b.SampleCount)
        if b.UpdatedAt.After(latest) {
            latest = b.UpdatedAt
        }
    }
    if totalSamples < bl.cfg.Fallback.FullGlobalMinSamples {
        return nil
    }
    if totalSamples == 0 {
        return nil
    }

    agg := &store.Baseline{
        P50:         sumP50 / float64(totalSamples),
        P95:         sumP95 / float64(totalSamples),
        MAD:         sumMAD / float64(totalSamples),
        SampleCount: totalSamples,
        UpdatedAt:   latest,
    }
    return &BaselineResult{
        Baseline:      agg,
        Source:        domain.SourceGlobal,
        FallbackLevel: 4,
        SourceDetails: "full global across all hours/daytypes",
    }
}
