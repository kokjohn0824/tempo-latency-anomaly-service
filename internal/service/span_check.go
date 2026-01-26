package service

import (
	"context"
	"fmt"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// SpanCheck evaluates whether a span is anomalous based on cached baselines.
type SpanCheck struct {
	store          store.Store
	cfg            *config.Config
	baselineLookup *SpanBaselineLookup
}

func NewSpanCheck(store store.Store, cfg *config.Config, baselineLookup *SpanBaselineLookup) *SpanCheck {
	return &SpanCheck{store: store, cfg: cfg, baselineLookup: baselineLookup}
}

func (s *SpanCheck) Evaluate(ctx context.Context, req domain.SpanAnomalyCheckRequest) (domain.AnomalyCheckResponse, error) {
	if s == nil || s.store == nil || s.cfg == nil || s.baselineLookup == nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("span check service not initialized")
	}

	timestampStr := fmt.Sprintf("%d", req.TimestampNano)
	bucket, err := domain.ParseTimeBucket(timestampStr, s.cfg.Timezone)
	if err != nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("parse time bucket: %w", err)
	}

	res, err := s.baselineLookup.LookupWithFallback(ctx, req.Service, req.SpanName, bucket)
	if err != nil {
		return domain.AnomalyCheckResponse{}, fmt.Errorf("lookup baseline: %w", err)
	}
	var b *store.Baseline
	if res != nil {
		b = res.Baseline
	}

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
