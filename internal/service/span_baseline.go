package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/stats"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// SpanBaseline recomputes baseline stats for span-level keys.
type SpanBaseline struct {
	store store.Store
	cfg   *config.Config
}

func NewSpanBaseline(store store.Store, cfg *config.Config) *SpanBaseline {
	return &SpanBaseline{store: store, cfg: cfg}
}

// RecomputeForKey recomputes baseline stats for a single span baseline key (spanbase:{...}).
func (s *SpanBaseline) RecomputeForKey(ctx context.Context, baselineKey string) (*domain.BaselineStats, error) {
	if s == nil || s.store == nil || s.cfg == nil {
		return nil, fmt.Errorf("span baseline service not initialized")
	}

	service, spanName, hour, dayType, err := parseSpanBaselineKey(baselineKey)
	if err != nil {
		return nil, err
	}

	bucket := domain.TimeBucket{Hour: hour, DayType: dayType}
	durKey := domain.MakeSpanDurationKey(service, spanName, bucket)

	samples, err := s.store.GetDurations(ctx, durKey)
	if err != nil {
		return nil, fmt.Errorf("get durations: %w", err)
	}

	bs := stats.ComputeBaseline(samples)
	err = s.store.SetBaseline(ctx, baselineKey, store.Baseline{
		P50:         bs.P50,
		P95:         bs.P95,
		MAD:         bs.MAD,
		SampleCount: bs.SampleCount,
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("set baseline: %w", err)
	}

	bs.UpdatedAt = time.Now().UTC()
	return &bs, nil
}

// parseSpanBaselineKey expects format: spanbase:{service}|{spanName}|{hour}|{dayType}
func parseSpanBaselineKey(key string) (service, spanName string, hour int, dayType string, err error) {
	if !strings.HasPrefix(key, "spanbase:") {
		err = fmt.Errorf("invalid span baseline key prefix: %s", key)
		return
	}
	body := strings.TrimPrefix(key, "spanbase:")
	parts := strings.Split(body, "|")
	if len(parts) != 4 {
		err = fmt.Errorf("invalid span baseline key format: %s", key)
		return
	}
	service = parts[0]
	spanName = parts[1]
	h, perr := strconv.Atoi(parts[2])
	if perr != nil {
		err = fmt.Errorf("invalid hour in key: %w", perr)
		return
	}
	hour = h
	dayType = parts[3]
	return
}
