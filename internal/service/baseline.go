package service

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/stats"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Baseline service recomputes baseline stats for dirty keys.
type Baseline struct {
    store store.Store
    cfg   *config.Config
}

func NewBaseline(store store.Store, cfg *config.Config) *Baseline {
    return &Baseline{store: store, cfg: cfg}
}

// RecomputeForKey recomputes baseline stats for a single baseline key (base:{...}).
// It derives the corresponding duration key (dur:{...}), fetches samples, computes stats,
// and persists them back to the baseline hash.
func (s *Baseline) RecomputeForKey(ctx context.Context, baselineKey string) (*domain.BaselineStats, error) {
    if s == nil || s.store == nil || s.cfg == nil {
        return nil, fmt.Errorf("baseline service not initialized")
    }

    service, endpoint, hour, dayType, err := parseBaselineKey(baselineKey)
    if err != nil {
        return nil, err
    }

    bucket := domain.TimeBucket{Hour: hour, DayType: dayType}
    durKey := domain.MakeDurationKey(service, endpoint, bucket)

    samples, err := s.store.GetDurations(ctx, durKey)
    if err != nil {
        return nil, fmt.Errorf("get durations: %w", err)
    }

    bs := stats.ComputeBaseline(samples)
    // Always store what we have; Check will guard on MinSamples
    err = s.store.SetBaseline(ctx, baselineKey, store.Baseline{
        P50:         bs.P50,
        P95:         bs.P95,
        MAD:         bs.MAD,
        SampleCount: bs.SampleCount,
        UpdatedAt:   time.Now().UTC(),
    })
    if err != nil {
        return nil, fmt.Errorf("set baseline: %w", err)
    }

    bs.UpdatedAt = time.Now().UTC()
    return &bs, nil
}

// parseBaselineKey expects format: base:{service}|{endpoint}|{hour}|{dayType}
func parseBaselineKey(key string) (service, endpoint string, hour int, dayType string, err error) {
    if !strings.HasPrefix(key, "base:") {
        err = fmt.Errorf("invalid baseline key prefix: %s", key)
        return
    }
    body := strings.TrimPrefix(key, "base:")
    parts := strings.Split(body, "|")
    if len(parts) != 4 {
        err = fmt.Errorf("invalid baseline key format: %s", key)
        return
    }
    service = parts[0]
    endpoint = parts[1]
    h, perr := strconv.Atoi(parts[2])
    if perr != nil {
        err = fmt.Errorf("invalid hour in key: %w", perr)
        return
    }
    hour = h
    dayType = parts[3]
    return
}

