package jobs

import (
    "context"
    "log"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/service"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

type BaselineRecompute struct {
    cfg      *config.Config
    baseline *service.Baseline
    store    store.Store
    batch    int64
}

func NewBaselineRecompute(cfg *config.Config, baseline *service.Baseline, st store.Store, batch int64) *BaselineRecompute {
    if batch <= 0 {
        batch = 100
    }
    return &BaselineRecompute{cfg: cfg, baseline: baseline, store: st, batch: batch}
}

func (b *BaselineRecompute) Run(ctx context.Context) {
    if b == nil || b.baseline == nil || b.store == nil || b.cfg == nil {
        return
    }
    interval := b.cfg.Polling.BaselineInterval
    if interval <= 0 {
        interval = 30 * time.Second
    }
    
    // Run immediately on startup
    b.tick(ctx)
    
    t := time.NewTicker(interval)
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            b.tick(ctx)
        }
    }
}

func (b *BaselineRecompute) tick(ctx context.Context) {
    keys, err := b.store.PopDirtyBatch(ctx, b.batch)
    if err != nil {
        log.Printf("dirty pop error: %v", err)
        return
    }
    for _, k := range keys {
        if _, err := b.baseline.RecomputeForKey(ctx, k); err != nil {
            log.Printf("baseline recompute error for %s: %v", k, err)
        }
    }
}

