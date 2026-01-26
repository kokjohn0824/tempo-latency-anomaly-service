package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

// SpanIngest handles ingestion of span-level duration samples.
type SpanIngest struct {
	store store.Store
	cfg   *config.Config
}

func NewSpanIngest(store store.Store, cfg *config.Config) *SpanIngest {
	return &SpanIngest{store: store, cfg: cfg}
}

// Spans ingests span duration samples and marks span baselines as dirty.
func (s *SpanIngest) Spans(ctx context.Context, spans []tempo.SpanData) error {
	if s == nil || s.store == nil || s.cfg == nil {
		return fmt.Errorf("span ingest service not initialized")
	}

	for _, span := range spans {
		if span.ServiceName == "" || span.Name == "" {
			continue
		}
		start, err := strconv.ParseInt(span.StartTimeUnixNano, 10, 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseInt(span.EndTimeUnixNano, 10, 64)
		if err != nil || end <= start {
			continue
		}
		durationMs := (end - start) / int64(1e6)
		bucket, err := domain.ParseTimeBucket(span.StartTimeUnixNano, s.cfg.Timezone)
		if err != nil {
			continue
		}

		durKey := domain.MakeSpanDurationKey(span.ServiceName, span.Name, bucket)
		baseKey := domain.MakeSpanBaselineKey(span.ServiceName, span.Name, bucket)
		if err := s.store.AppendDuration(ctx, durKey, durationMs, s.cfg.WindowSize); err != nil {
			return fmt.Errorf("append span duration: %w", err)
		}
		if err := s.store.MarkDirty(ctx, baseKey); err != nil {
			return fmt.Errorf("mark span baseline dirty: %w", err)
		}
	}

	return nil
}
