package mocks

import (
    "context"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
    "github.com/stretchr/testify/mock"
)

// MockStore implements store.Store using testify/mock for unit tests.
type MockStore struct {
    mock.Mock
}

// DurationOps
func (m *MockStore) AppendDuration(ctx context.Context, key string, durationMs int64, windowSize int) error {
    args := m.Called(ctx, key, durationMs, windowSize)
    return args.Error(0)
}

func (m *MockStore) GetDurations(ctx context.Context, key string) ([]int64, error) {
    args := m.Called(ctx, key)
    if v, ok := args.Get(0).([]int64); ok {
        return v, args.Error(1)
    }
    return nil, args.Error(1)
}

// BaselineOps
func (m *MockStore) GetBaseline(ctx context.Context, key string) (*store.Baseline, error) {
    args := m.Called(ctx, key)
    if v, ok := args.Get(0).(*store.Baseline); ok {
        return v, args.Error(1)
    }
    return nil, args.Error(1)
}

func (m *MockStore) SetBaseline(ctx context.Context, key string, b store.Baseline) error {
    args := m.Called(ctx, key, b)
    return args.Error(0)
}

func (m *MockStore) GetBaselines(ctx context.Context, keys []string) (map[string]*store.Baseline, error) {
    args := m.Called(ctx, keys)
    if v, ok := args.Get(0).(map[string]*store.Baseline); ok {
        return v, args.Error(1)
    }
    return nil, args.Error(1)
}

// DedupOps
func (m *MockStore) IsDuplicateOrMark(ctx context.Context, traceID string, ttl time.Duration) (bool, error) {
    args := m.Called(ctx, traceID, ttl)
    return args.Bool(0), args.Error(1)
}

// DirtyOps
func (m *MockStore) MarkDirty(ctx context.Context, key string) error {
    args := m.Called(ctx, key)
    return args.Error(0)
}

func (m *MockStore) PopDirtyBatch(ctx context.Context, count int64) ([]string, error) {
    args := m.Called(ctx, count)
    if v, ok := args.Get(0).([]string); ok {
        return v, args.Error(1)
    }
    return nil, args.Error(1)
}

// ListOps
func (m *MockStore) ListBaselineKeys(ctx context.Context, minSamples int) ([]string, error) {
    args := m.Called(ctx, minSamples)
    if v, ok := args.Get(0).([]string); ok {
        return v, args.Error(1)
    }
    return nil, args.Error(1)
}

// Close
func (m *MockStore) Close() error {
    args := m.Called()
    return args.Error(0)
}

