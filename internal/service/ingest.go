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
    if s == nil || s.store == nil || s.cfg == nil {
        return fmt.Errorf("ingest service not initialized")
    }

    // Deduplicate by trace ID
    dup, err := s.store.IsDuplicateOrMark(ctx, ev.TraceID, s.cfg.Dedup.TTL)
    if err != nil {
        return fmt.Errorf("dedup: %w", err)
    }
    if dup {
        // Already seen; skip further processing
        return nil
    }

    // Determine bucket and keys
    bucket, err := domain.ParseTimeBucket(ev.StartTimeUnixNano, s.cfg.Timezone)
    if err != nil {
        return fmt.Errorf("parse time bucket: %w", err)
    }

    service := ev.RootServiceName
    endpoint := ev.RootTraceName

    durKey := domain.MakeDurationKey(service, endpoint, bucket)
    baseKey := domain.MakeBaselineKey(service, endpoint, bucket)

    // Append duration sample and trim
    if err := s.store.AppendDuration(ctx, durKey, ev.DurationMs, s.cfg.WindowSize); err != nil {
        return fmt.Errorf("append duration: %w", err)
    }

    // Mark baseline dirty for recomputation
    if err := s.store.MarkDirty(ctx, baseKey); err != nil {
        return fmt.Errorf("mark dirty: %w", err)
    }

    return nil
}

