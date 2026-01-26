package jobs

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/service"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

type TempoPoller struct {
	cfg    *config.Config
	client *tempo.Client
	ingest *service.Ingest
	spans  *service.SpanIngest
}

func NewTempoPoller(cfg *config.Config, client *tempo.Client, ingest *service.Ingest, spans *service.SpanIngest) *TempoPoller {
	return &TempoPoller{cfg: cfg, client: client, ingest: ingest, spans: spans}
}

func (p *TempoPoller) Run(ctx context.Context) {
	if p == nil || p.client == nil || p.ingest == nil || p.cfg == nil {
		return
	}
	interval := p.cfg.Polling.TempoInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	// Perform historical backfill before starting regular polling
	p.backfill(ctx)

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
	if len(events) > 450 {
		log.Printf("tempo poller WARNING: query results (%d) close to limit (500). Consider increasing limit or reducing lookback to avoid drops.", len(events))
	}
	ingested := 0
	for _, ev := range events {
		ok, err := p.ingestTraceAndSpans(ctx, ev)
		if err != nil {
			log.Printf("ingest error: %v", err)
			continue
		}
		if ok {
			ingested++
		}
	}
	if ingested > 0 {
		log.Printf("tempo poller: ingested %d traces", ingested)
	}
}

// backfill performs historical data ingestion before regular polling starts.
// It queries data in batches to avoid overloading Tempo, based on configuration:
// - cfg.Polling.BackfillEnabled: toggle
// - cfg.Polling.BackfillDuration: how far back to backfill from now
// - cfg.Polling.BackfillBatch: batch window size per query (default 1h)
// Implementation notes:
//   - Each batch limits query rate by sleeping 1 second between calls.
//   - We use the existing client that queries [now - lookback, now]. To approximate
//     fixed windows, we filter events to the [batchStart, batchEnd) interval.
func (p *TempoPoller) backfill(ctx context.Context) {
	if p == nil || p.cfg == nil || !p.cfg.Polling.BackfillEnabled {
		return
	}

	duration := p.cfg.Polling.BackfillDuration
	if duration <= 0 {
		duration = config.DefaultBackfillDuration
	}
	batch := p.cfg.Polling.BackfillBatch
	if batch <= 0 {
		batch = config.DefaultBackfillBatch
	}

	// End backfill at the boundary before normal lookback window to reduce overlap
	now := time.Now()
	end := now.Add(-p.cfg.Polling.TempoLookback)
	start := now.Add(-duration)
	if end.Before(start) {
		// Nothing to backfill
		return
	}

	log.Printf("tempo backfill: starting %s to %s (batch %s)", start.Format(time.RFC3339), end.Format(time.RFC3339), batch.String())

	// Iterate from oldest to newest within [start, end)
	for current := start; current.Before(end); current = current.Add(batch) {
		select {
		case <-ctx.Done():
			log.Printf("tempo backfill: canceled")
			return
		default:
		}

		batchEnd := current.Add(batch)
		if batchEnd.After(end) {
			batchEnd = end
		}

		// Query traces since the batch start relative to now
		lookbackSec := int(time.Since(current).Seconds())
		if lookbackSec <= 0 {
			lookbackSec = 1
		}

		log.Printf("tempo backfill: querying window %s to %s (lookback ~%ds)", current.Format(time.RFC3339), batchEnd.Format(time.RFC3339), lookbackSec)
		events, err := p.client.QueryTraces(ctx, lookbackSec)
		if err != nil {
			log.Printf("tempo backfill error: %v", err)
			// continue to next batch after brief pause
			time.Sleep(1 * time.Second)
			continue
		}

		// Filter events to the target batch window [current, batchEnd)
		// StartTimeUnixNano is a string of unix nanos; parse for comparison
		var (
			filtered  = make([]domain.TraceEvent, 0, len(events))
			lowerNano = current.UnixNano()
			upperNano = batchEnd.UnixNano()
		)
		for _, ev := range events {
			ns, err := strconv.ParseInt(ev.StartTimeUnixNano, 10, 64)
			if err != nil {
				// if parse fails, skip this event
				continue
			}
			if ns >= lowerNano && ns < upperNano {
				filtered = append(filtered, ev)
			}
		}

		// Ingest filtered events
		ingested := 0
		for _, fev := range filtered {
			ok, err := p.ingestTraceAndSpans(ctx, fev)
			if err != nil {
				log.Printf("tempo backfill ingest error: %v", err)
				continue
			}
			if ok {
				ingested++
			}
		}

		log.Printf("tempo backfill: received %d traces, filtered %d, ingested %d for %s to %s", len(events), len(filtered), ingested, current.Format(time.RFC3339), batchEnd.Format(time.RFC3339))

		// If we are consistently close to the limit, surface a warning
		if len(events) > 450 {
			log.Printf("tempo backfill WARNING: batch query results (%d) close to limit (500). Consider increasing limit or reducing batch size.", len(events))
		}

		// Sleep to avoid overloading Tempo during backfill
		time.Sleep(1 * time.Second)
	}

	log.Printf("tempo backfill: completed")
}

func (p *TempoPoller) ingestTraceAndSpans(ctx context.Context, ev domain.TraceEvent) (bool, error) {
	if p == nil || p.ingest == nil {
		return false, fmt.Errorf("ingest service not initialized")
	}

	ingested, err := p.ingest.TraceWithResult(ctx, ev)
	if err != nil || !ingested {
		return ingested, err
	}

	if p.spans == nil {
		return ingested, nil
	}

	spans, err := p.client.GetTraceSpans(ctx, ev.TraceID)
	if err != nil {
		return ingested, fmt.Errorf("fetch trace spans: %w", err)
	}

	if err := p.spans.Spans(ctx, spans); err != nil {
		return ingested, fmt.Errorf("ingest span durations: %w", err)
	}

	return ingested, nil
}

// (no additional types)
