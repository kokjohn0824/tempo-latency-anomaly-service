package service

import (
    "context"
    "fmt"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Check service evaluates whether a given request is anomalous based on cached baselines.
type Check struct {
    store store.Store
    cfg   *config.Config
}

func NewCheck(store store.Store, cfg *config.Config) *Check {
    return &Check{store: store, cfg: cfg}
}

func (s *Check) Evaluate(ctx context.Context, req domain.AnomalyCheckRequest) (domain.AnomalyCheckResponse, error) {
    if s == nil || s.store == nil || s.cfg == nil {
        return domain.AnomalyCheckResponse{}, fmt.Errorf("check service not initialized")
    }

    // Convert int64 timestamp to string for ParseTimeBucket
    timestampStr := fmt.Sprintf("%d", req.TimestampNano)
    bucket, err := domain.ParseTimeBucket(timestampStr, s.cfg.Timezone)
    if err != nil {
        return domain.AnomalyCheckResponse{}, fmt.Errorf("parse time bucket: %w", err)
    }

    baseKey := domain.MakeBaselineKey(req.Service, req.Endpoint, bucket)
    b, err := s.store.GetBaseline(ctx, baseKey)
    if err != nil {
        return domain.AnomalyCheckResponse{}, fmt.Errorf("get baseline: %w", err)
    }

    // Prepare response scaffolding
    resp := domain.AnomalyCheckResponse{Bucket: bucket}
    if b != nil {
        resp.Baseline = &domain.BaselineStats{
            P50:         b.P50,
            P95:         b.P95,
            MAD:         b.MAD,
            SampleCount: b.SampleCount,
            UpdatedAt:   b.UpdatedAt,
        }
    }

    // Insufficient baseline data
    if b == nil || b.SampleCount < s.cfg.Stats.MinSamples {
        resp.IsAnomaly = false
        resp.Explanation = fmt.Sprintf(
            "no baseline available or insufficient samples (have %d, need >= %d)",
            valueOrZero(b, func(x *store.Baseline) int { return x.SampleCount }),
            s.cfg.Stats.MinSamples,
        )
        return resp, nil
    }

    // Compute threshold and compare
    rel := b.P95 * s.cfg.Stats.Factor
    abs := b.P50 + float64(s.cfg.Stats.K)*b.MAD
    threshold := rel
    if abs > threshold {
        threshold = abs
    }

    dur := float64(req.DurationMs)
    isAnomaly := dur > threshold

    resp.IsAnomaly = isAnomaly
    resp.Explanation = fmt.Sprintf(
        "duration %.0fms %s threshold %.2fms (p50=%.2f, p95=%.2f, MAD=%.2f, factor=%.2f, k=%d)",
        dur,
        ternary(isAnomaly, "exceeds", "within"),
        threshold,
        b.P50, b.P95, b.MAD,
        s.cfg.Stats.Factor,
        s.cfg.Stats.K,
    )
    return resp, nil
}

func valueOrZero[T any, R any](v *T, f func(*T) R) R {
    var zero R
    if v == nil {
        return zero
    }
    return f(v)
}

func ternary[T any](cond bool, a, b T) T {
    if cond {
        return a
    }
    return b
}

