package jobs

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/service"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

type BaselineRecompute struct {
	cfg      *config.Config
	baseline *service.Baseline
	spanBase *service.SpanBaseline
	store    store.Store
	batch    int64
}

func NewBaselineRecompute(cfg *config.Config, baseline *service.Baseline, spanBase *service.SpanBaseline, st store.Store, batch int64) *BaselineRecompute {
	if batch <= 0 {
		batch = 100
	}
	return &BaselineRecompute{cfg: cfg, baseline: baseline, spanBase: spanBase, store: st, batch: batch}
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
		if strings.HasPrefix(k, "spanbase:") {
			if b.spanBase == nil {
				log.Printf("span baseline recompute skipped (not configured) for %s", k)
				continue
			}
			if _, err := b.spanBase.RecomputeForKey(ctx, k); err != nil {
				log.Printf("span baseline recompute error for %s: %v", k, err)
			}
			continue
		}

		if strings.HasPrefix(k, "base:") {
			if _, err := b.baseline.RecomputeForKey(ctx, k); err != nil {
				log.Printf("baseline recompute error for %s: %v", k, err)
			}
			continue
		}

		log.Printf("unknown baseline key prefix: %s", k)
	}
}
