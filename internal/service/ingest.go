package service

import (
	"context"
	"fmt"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Ingest handles ingestion of a single trace event:
// - Deduplicate by traceID
// - Derive time bucket and keys
// - Append duration sample to rolling window
// - Mark corresponding baseline key as dirty for recomputation
type Ingest struct {
	store store.Store
	cfg   *config.Config
}

func NewIngest(store store.Store, cfg *config.Config) *Ingest {
	return &Ingest{store: store, cfg: cfg}
}

// Trace ingests a single Tempo trace event.
func (s *Ingest) Trace(ctx context.Context, ev domain.TraceEvent) error {
	_, err := s.TraceWithResult(ctx, ev)
	return err
}

// TraceWithResult ingests a trace event and returns whether it was newly ingested.
func (s *Ingest) TraceWithResult(ctx context.Context, ev domain.TraceEvent) (bool, error) {
	if s == nil || s.store == nil || s.cfg == nil {
		return false, fmt.Errorf("ingest service not initialized")
	}

	// Deduplicate by trace ID
	dup, err := s.store.IsDuplicateOrMark(ctx, ev.TraceID, s.cfg.Dedup.TTL)
	if err != nil {
		return false, fmt.Errorf("dedup: %w", err)
	}
	if dup {
		return false, nil
	}

	bucket, err := domain.ParseTimeBucket(ev.StartTimeUnixNano, s.cfg.Timezone)
	if err != nil {
		return false, fmt.Errorf("parse time bucket: %w", err)
	}

	service := ev.RootServiceName
	endpoint := ev.RootTraceName

	durKey := domain.MakeDurationKey(service, endpoint, bucket)
	baseKey := domain.MakeBaselineKey(service, endpoint, bucket)

	if err := s.store.AppendDuration(ctx, durKey, ev.DurationMs, s.cfg.WindowSize); err != nil {
		return false, fmt.Errorf("append duration: %w", err)
	}

	if err := s.store.MarkDirty(ctx, baseKey); err != nil {
		return false, fmt.Errorf("mark dirty: %w", err)
	}

	return true, nil
}
