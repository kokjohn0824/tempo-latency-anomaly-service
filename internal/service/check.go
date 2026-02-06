package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Check service evaluates whether a given request is anomalous based on cached baselines.
type Check struct {
	store          store.Store
	cfg            *config.Config
	baselineLookup *BaselineLookup
}

func NewCheck(store store.Store, cfg *config.Config, baselineLookup *BaselineLookup) *Check {
	return &Check{store: store, cfg: cfg, baselineLookup: baselineLookup}
}

func (s *Check) Evaluate(ctx context.Context, req domain.AnomalyCheckRequest) (domain.AnomalyCheckResponse, error) {
	if s == nil || s.store == nil || s.cfg == nil || s.baselineLookup == nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("check service not initialized")
	}

	// Convert int64 timestamp to string for ParseTimeBucket
	timestampStr := fmt.Sprintf("%d", req.TimestampNano)
	bucket, err := domain.ParseTimeBucket(timestampStr, s.cfg.Timezone)
	if err != nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("parse time bucket: %w", err)
	}

	// Use BaselineLookup with fallback strategy
	res, err := s.baselineLookup.LookupWithFallback(ctx, req.Service, req.Endpoint, bucket)
	if err != nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("lookup baseline: %w", err)
	}
	var b *store.Baseline
	if res != nil {
		b = res.Baseline
	}

	// Prepare response scaffolding
	resp := domain.AnomalyCheckResponse{Bucket: bucket}
	if res != nil {
		resp.BaselineSource = res.Source
		resp.FallbackLevel = res.FallbackLevel
		resp.SourceDetails = res.SourceDetails
		resp.CannotDetermine = res.CannotDetermine
	}
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

	eval := EvaluateDuration(s.cfg, req.DurationMs, b)
	resp.IsAnomaly = eval.IsAnomaly
	resp.Explanation = eval.Explanation
	return resp, nil
}

// AnnotateTraces adds IsAnomaly to each TraceEvent using time-bucketed baselines.
// It batches baseline lookups per unique (service|endpoint|hour|dayType) bucket to reduce store calls.
//
// Note: baseline keys are derived from each trace's RootServiceName/RootTraceName
// (the same pair used during ingestion to build baselines).
func (s *Check) AnnotateTraces(ctx context.Context, traces []domain.TraceEvent) ([]domain.TraceEvent, error) {
	if s == nil || s.store == nil || s.cfg == nil || s.baselineLookup == nil {
		return nil, fmt.Errorf("check service not initialized")
	}

	// Copy to avoid mutating caller slice.
	out := make([]domain.TraceEvent, len(traces))
	copy(out, traces)

	// Cache baseline results by key "service|endpoint|hour|dayType".
	cache := make(map[string]*BaselineResult, 16)

	for i := range out {
		// Use trace's own root identifiers to match ingestion baseline keys.
		svc := out[i].RootServiceName
		ep := out[i].RootTraceName
		if svc == "" || ep == "" {
			out[i].IsAnomaly = false
			continue
		}

		tsStr := out[i].StartTimeUnixNano
		tsNano, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil || tsNano <= 0 {
			out[i].IsAnomaly = false
			continue
		}

		bucket, err := domain.ParseTimeBucket(fmt.Sprintf("%d", tsNano), s.cfg.Timezone)
		if err != nil {
			out[i].IsAnomaly = false
			continue
		}

		bucketKey := fmt.Sprintf("%s|%s|%d|%s", svc, ep, bucket.Hour, bucket.DayType)
		res, ok := cache[bucketKey]
		if !ok {
			res, err = s.baselineLookup.LookupWithFallback(ctx, svc, ep, bucket)
			if err != nil {
				return nil, fmt.Errorf("lookup baseline: %w", err)
			}
			cache[bucketKey] = res
		}

		var b *store.Baseline
		if res != nil {
			b = res.Baseline
		}

		// If baseline is missing or insufficient, don't flag anomalies.
		if b == nil || b.SampleCount < s.cfg.Stats.MinSamples {
			out[i].IsAnomaly = false
			continue
		}

		eval := EvaluateDuration(s.cfg, out[i].DurationMs, b)
		out[i].IsAnomaly = eval.IsAnomaly
	}

	return out, nil
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
