package store

import (
    "context"
    "time"
)

// Baseline represents cached baseline statistics for a key.
// Keys stored in Redis hash: p50, p95, mad, sampleCount, updatedAt
type Baseline struct {
    P50         float64   `json:"p50" example:"233.5"`
    P95         float64   `json:"p95" example:"562.0"`
    MAD         float64   `json:"mad" example:"43.0"`
    SampleCount int       `json:"sampleCount" example:"50"`
    UpdatedAt   time.Time `json:"updatedAt" example:"2026-01-15T08:00:00Z"`
}

// DurationOps defines operations for rolling window samples (dur:* lists).
type DurationOps interface {
    // AppendDuration pushes a duration (ms) into the rolling list and trims to windowSize.
    AppendDuration(ctx context.Context, key string, durationMs int64, windowSize int) error
    // GetDurations returns all durations (ms) currently in the rolling list.
    GetDurations(ctx context.Context, key string) ([]int64, error)
}

// BaselineOps defines cached baseline read/write operations (base:* hashes).
type BaselineOps interface {
    // GetBaseline fetches baseline stats for key. Returns (nil, nil) if not found.
    GetBaseline(ctx context.Context, key string) (*Baseline, error)
    // SetBaseline stores baseline stats (including SampleCount and UpdatedAt).
    SetBaseline(ctx context.Context, key string, b Baseline) error
    // GetBaselines fetches baseline stats for multiple keys in one call.
    // Returns a map of key -> Baseline for keys that exist. Missing keys are omitted.
    GetBaselines(ctx context.Context, keys []string) (map[string]*Baseline, error)
}

// DedupOps defines trace ID deduplication (seen:* keys).
type DedupOps interface {
    // IsDuplicateOrMark performs atomic check-and-set with TTL.
    // Returns true if the traceID has been seen before (duplicate), false if marked as new.
    IsDuplicateOrMark(ctx context.Context, traceID string, ttl time.Duration) (bool, error)
}

// DirtyOps defines tracking of keys that need recomputation (dirtyKeys set).
type DirtyOps interface {
    // MarkDirty adds the key to the dirty set.
    MarkDirty(ctx context.Context, key string) error
    // PopDirtyBatch pops up to count keys from the dirty set for processing.
    PopDirtyBatch(ctx context.Context, count int64) ([]string, error)
}

// ListOps defines operations for listing available baselines.
type ListOps interface {
    // ListBaselineKeys returns all baseline keys matching the pattern (base:*).
    // Returns keys that have sufficient samples for anomaly detection.
    ListBaselineKeys(ctx context.Context, minSamples int) ([]string, error)
}

// Store aggregates all storage operations and allows closing resources.
type Store interface {
    DurationOps
    BaselineOps
    DedupOps
    DirtyOps
    ListOps
    Close() error
}
