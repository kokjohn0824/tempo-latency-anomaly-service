package service

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    smocks "github.com/alexchang/tempo-latency-anomaly-service/internal/store/mocks"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func ingestCfg() *config.Config {
    return &config.Config{
        Timezone:   "Asia/Taipei",
        WindowSize: 500,
        Dedup:      config.DedupConfig{TTL: 1 * time.Hour},
    }
}

func TestIngest_Trace_DedupSkip(t *testing.T) {
    ctx := context.Background()
    m := new(smocks.MockStore)

    // Duplicate event → early return, no other calls
    m.On("IsDuplicateOrMark", mock.Anything, "trace-1", ingestCfg().Dedup.TTL).Return(true, nil)

    ing := NewIngest(m, ingestCfg())
    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 12, 0, 0, 0, loc)
    ev := domain.TraceEvent{
        TraceID:           "trace-1",
        RootServiceName:   "svcX",
        RootTraceName:     "GET /a",
        StartTimeUnixNano: fmt.Sprintf("%d", ts.UnixNano()),
        DurationMs:        123,
    }

    err := ing.Trace(ctx, ev)
    assert.NoError(t, err)
    m.AssertNotCalled(t, "AppendDuration", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
    m.AssertNotCalled(t, "MarkDirty", mock.Anything, mock.Anything)
    m.AssertExpectations(t)
}

func TestIngest_Trace_ProcessAndMarkDirty(t *testing.T) {
    ctx := context.Background()
    cfg := ingestCfg()
    m := new(smocks.MockStore)

    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 16, 45, 0, 0, loc) // Tue weekday hour=16
    bucket, _ := domain.ParseTimeBucket(fmt.Sprintf("%d", ts.UnixNano()), cfg.Timezone)

    ev := domain.TraceEvent{
        TraceID:           "trace-2",
        RootServiceName:   "svcY",
        RootTraceName:     "POST /b",
        StartTimeUnixNano: fmt.Sprintf("%d", ts.UnixNano()),
        DurationMs:        250,
    }

    durKey := domain.MakeDurationKey(ev.RootServiceName, ev.RootTraceName, bucket)
    baseKey := domain.MakeBaselineKey(ev.RootServiceName, ev.RootTraceName, bucket)

    m.On("IsDuplicateOrMark", mock.Anything, ev.TraceID, cfg.Dedup.TTL).Return(false, nil)
    m.On("AppendDuration", mock.Anything, durKey, int64(250), cfg.WindowSize).Return(nil)
    m.On("MarkDirty", mock.Anything, baseKey).Return(nil)

    ing := NewIngest(m, cfg)
    err := ing.Trace(ctx, ev)
    assert.NoError(t, err)
    m.AssertExpectations(t)
}

