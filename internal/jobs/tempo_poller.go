package jobs

import (
    "context"
    "log"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/service"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

type TempoPoller struct {
    cfg    *config.Config
    client *tempo.Client
    ingest *service.Ingest
}

func NewTempoPoller(cfg *config.Config, client *tempo.Client, ingest *service.Ingest) *TempoPoller {
    return &TempoPoller{cfg: cfg, client: client, ingest: ingest}
}

func (p *TempoPoller) Run(ctx context.Context) {
    if p == nil || p.client == nil || p.ingest == nil || p.cfg == nil {
        return
    }
    interval := p.cfg.Polling.TempoInterval
    if interval <= 0 {
        interval = 15 * time.Second
    }
    
    // Run immediately on startup
    p.tick(ctx)
    
    t := time.NewTicker(interval)
    defer t.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            p.tick(ctx)
        }
    }
}

func (p *TempoPoller) tick(ctx context.Context) {
    lookback := int(p.cfg.Polling.TempoLookback / time.Second)
    if lookback <= 0 {
        lookback = 120
    }
    log.Printf("tempo poller: querying last %d seconds", lookback)
    events, err := p.client.QueryTraces(ctx, lookback)
    if err != nil {
        log.Printf("tempo poll error: %v", err)
        return
    }
    log.Printf("tempo poller: received %d traces", len(events))
    ingested := 0
    for _, ev := range events {
        if err := p.ingest.Trace(ctx, ev); err != nil {
            log.Printf("ingest error: %v", err)
        } else {
            ingested++
        }
    }
    if ingested > 0 {
        log.Printf("tempo poller: ingested %d traces", ingested)
    }
}

