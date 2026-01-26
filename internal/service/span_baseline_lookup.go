package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// SpanBaselineLookup provides baseline retrieval with multi-level fallback strategy for spans.
type SpanBaselineLookup struct {
	store store.Store
	cfg   *config.Config
}

// NewSpanBaselineLookup constructs a new SpanBaselineLookup service.
func NewSpanBaselineLookup(store store.Store, cfg *config.Config) *SpanBaselineLookup {
	return &SpanBaselineLookup{store: store, cfg: cfg}
}

// LookupWithFallback attempts to find an appropriate baseline using the configured
// multi-level fallback flow for span names.
func (bl *SpanBaselineLookup) LookupWithFallback(
	ctx context.Context,
	service, spanName string,
	bucket domain.TimeBucket,
) (*BaselineResult, error) {
	if res := bl.tryExactMatch(ctx, service, spanName, bucket); res != nil {
		return res, nil
	}

	if bl.cfg != nil && bl.cfg.Fallback.NearbyHoursEnabled {
		if res := bl.tryNearbyHours(ctx, service, spanName, bucket); res != nil {
			return res, nil
		}
	}

	if bl.cfg != nil && bl.cfg.Fallback.DayTypeGlobalEnabled {
		if res := bl.tryDayTypeGlobal(ctx, service, spanName, bucket.DayType); res != nil {
			return res, nil
		}
	}

	if bl.cfg != nil && bl.cfg.Fallback.FullGlobalEnabled {
		if res := bl.tryFullGlobal(ctx, service, spanName); res != nil {
			return res, nil
		}
	}

	return &BaselineResult{
		Baseline:        nil,
		Source:          domain.SourceUnavailable,
		FallbackLevel:   5,
		SourceDetails:   "no baseline data available",
		CannotDetermine: true,
	}, nil
}

func (bl *SpanBaselineLookup) tryExactMatch(ctx context.Context, service, spanName string, bucket domain.TimeBucket) *BaselineResult {
	if bl == nil || bl.store == nil || bl.cfg == nil {
		return nil
	}

	key := domain.MakeSpanBaselineKey(service, spanName, bucket)
	b, err := bl.store.GetBaseline(ctx, key)
	if err != nil || b == nil {
		return nil
	}
	if b.SampleCount < bl.cfg.Stats.MinSamples {
		return nil
	}
	detail := fmt.Sprintf("exact match: %d|%s", bucket.Hour, bucket.DayType)
	return &BaselineResult{
		Baseline:      b,
		Source:        domain.SourceExact,
		FallbackLevel: 1,
		SourceDetails: detail,
	}
}

func (bl *SpanBaselineLookup) tryNearbyHours(ctx context.Context, service, spanName string, bucket domain.TimeBucket) *BaselineResult {
	if bl == nil || bl.store == nil || bl.cfg == nil {
		return nil
	}

	r := bl.cfg.Fallback.NearbyHoursRange
	if r <= 0 {
		return nil
	}

	type hk struct {
		key  string
		hour int
	}
	var keys []hk
	seen := make(map[int]bool)
	center := bucket.Hour
	wrap := func(h int) int { return (h%24 + 24) % 24 }
	for d := 1; d <= r; d++ {
		for _, h := range []int{wrap(center - d), wrap(center + d)} {
			if seen[h] {
				continue
			}
			seen[h] = true
			k := domain.MakeSpanBaselineKey(service, spanName, domain.TimeBucket{Hour: h, DayType: bucket.DayType})
			keys = append(keys, hk{key: k, hour: h})
		}
	}
	if len(keys) == 0 {
		return nil
	}

	rawKeys := make([]string, 0, len(keys))
	for _, it := range keys {
		rawKeys = append(rawKeys, it.key)
	}
	m, err := bl.store.GetBaselines(ctx, rawKeys)
	if err != nil || len(m) == 0 {
		return nil
	}

	var totalSamples int
	var sumP50, sumP95, sumMAD float64
	var latest time.Time
	var usedHours []int
	for _, it := range keys {
		b := m[it.key]
		if b == nil || b.SampleCount <= 0 {
			continue
		}
		totalSamples += b.SampleCount
		sumP50 += b.P50 * float64(b.SampleCount)
		sumP95 += b.P95 * float64(b.SampleCount)
		sumMAD += b.MAD * float64(b.SampleCount)
		if b.UpdatedAt.After(latest) {
			latest = b.UpdatedAt
		}
		usedHours = append(usedHours, it.hour)
	}
	if totalSamples < bl.cfg.Fallback.NearbyMinSamples {
		return nil
	}
	if totalSamples == 0 {
		return nil
	}

	detailParts := make([]string, 0, len(usedHours))
	for _, h := range usedHours {
		detailParts = append(detailParts, fmt.Sprintf("%d", h))
	}
	details := fmt.Sprintf("nearby hours: %s (%s)", strings.Join(detailParts, ","), bucket.DayType)

	agg := &store.Baseline{
		P50:         sumP50 / float64(totalSamples),
		P95:         sumP95 / float64(totalSamples),
		MAD:         sumMAD / float64(totalSamples),
		SampleCount: totalSamples,
		UpdatedAt:   latest,
	}
	return &BaselineResult{
		Baseline:      agg,
		Source:        domain.SourceNearby,
		FallbackLevel: 2,
		SourceDetails: details,
	}
}

func (bl *SpanBaselineLookup) tryDayTypeGlobal(ctx context.Context, service, spanName string, dayType string) *BaselineResult {
	if bl == nil || bl.store == nil || bl.cfg == nil {
		return nil
	}

	rawKeys := make([]string, 0, 24)
	for h := 0; h < 24; h++ {
		rawKeys = append(rawKeys, domain.MakeSpanBaselineKey(service, spanName, domain.TimeBucket{Hour: h, DayType: dayType}))
	}
	m, err := bl.store.GetBaselines(ctx, rawKeys)
	if err != nil || len(m) == 0 {
		return nil
	}

	var totalSamples int
	var sumP50, sumP95, sumMAD float64
	var latest time.Time
	var usedHours []int
	for h := 0; h < 24; h++ {
		k := domain.MakeSpanBaselineKey(service, spanName, domain.TimeBucket{Hour: h, DayType: dayType})
		b := m[k]
		if b == nil || b.SampleCount <= 0 {
			continue
		}
		totalSamples += b.SampleCount
		sumP50 += b.P50 * float64(b.SampleCount)
		sumP95 += b.P95 * float64(b.SampleCount)
		sumMAD += b.MAD * float64(b.SampleCount)
		if b.UpdatedAt.After(latest) {
			latest = b.UpdatedAt
		}
		usedHours = append(usedHours, h)
	}
	if totalSamples < bl.cfg.Fallback.DayTypeGlobalMinSamples {
		return nil
	}
	if totalSamples == 0 {
		return nil
	}

	sort.Ints(usedHours)
	parts := make([]string, 0, len(usedHours))
	for _, h := range usedHours {
		parts = append(parts, fmt.Sprintf("%d", h))
	}
	details := fmt.Sprintf("daytype=%s hours=%s", dayType, strings.Join(parts, ","))

	agg := &store.Baseline{
		P50:         sumP50 / float64(totalSamples),
		P95:         sumP95 / float64(totalSamples),
		MAD:         sumMAD / float64(totalSamples),
		SampleCount: totalSamples,
		UpdatedAt:   latest,
	}
	return &BaselineResult{
		Baseline:      agg,
		Source:        domain.SourceDayType,
		FallbackLevel: 3,
		SourceDetails: details,
	}
}

func (bl *SpanBaselineLookup) tryFullGlobal(ctx context.Context, service, spanName string) *BaselineResult {
	if bl == nil || bl.store == nil || bl.cfg == nil {
		return nil
	}

	dayTypes := []string{"weekday", "weekend"}
	rawKeys := make([]string, 0, 48)
	for _, dt := range dayTypes {
		for h := 0; h < 24; h++ {
			rawKeys = append(rawKeys, domain.MakeSpanBaselineKey(service, spanName, domain.TimeBucket{Hour: h, DayType: dt}))
		}
	}
	m, err := bl.store.GetBaselines(ctx, rawKeys)
	if err != nil || len(m) == 0 {
		return nil
	}

	var totalSamples int
	var sumP50, sumP95, sumMAD float64
	var latest time.Time
	for _, k := range rawKeys {
		b := m[k]
		if b == nil || b.SampleCount <= 0 {
			continue
		}
		totalSamples += b.SampleCount
		sumP50 += b.P50 * float64(b.SampleCount)
		sumP95 += b.P95 * float64(b.SampleCount)
		sumMAD += b.MAD * float64(b.SampleCount)
		if b.UpdatedAt.After(latest) {
			latest = b.UpdatedAt
		}
	}
	if totalSamples < bl.cfg.Fallback.FullGlobalMinSamples {
		return nil
	}
	if totalSamples == 0 {
		return nil
	}

	agg := &store.Baseline{
		P50:         sumP50 / float64(totalSamples),
		P95:         sumP95 / float64(totalSamples),
		MAD:         sumMAD / float64(totalSamples),
		SampleCount: totalSamples,
		UpdatedAt:   latest,
	}
	return &BaselineResult{
		Baseline:      agg,
		Source:        domain.SourceGlobal,
		FallbackLevel: 4,
		SourceDetails: "full global across all hours/daytypes",
	}
}
