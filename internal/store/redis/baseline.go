package redis

import (
    "context"
    "strconv"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

const (
    fieldP50         = "p50"
    fieldP95         = "p95"
    fieldMAD         = "mad"
    fieldSampleCount = "sampleCount"
    fieldUpdatedAt   = "updatedAt"
    timeLayout       = time.RFC3339Nano
)

// GetBaseline reads baseline stats from Redis hash at key.
func (c *Client) GetBaseline(ctx context.Context, key string) (*store.Baseline, error) {
    m, err := c.rdb.HGetAll(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    if len(m) == 0 {
        return nil, nil
    }

    // Parse fields, missing values default to zero.
    parseFloat := func(k string) float64 {
        if v, ok := m[k]; ok {
            if f, err := strconv.ParseFloat(v, 64); err == nil {
                return f
            }
        }
        return 0
    }
    parseInt := func(k string) int {
        if v, ok := m[k]; ok {
            if n, err := strconv.Atoi(v); err == nil {
                return n
            }
        }
        return 0
    }
    parseTime := func(k string) time.Time {
        if v, ok := m[k]; ok {
            if t, err := time.Parse(timeLayout, v); err == nil {
                return t
            }
        }
        return time.Time{}
    }

    b := &store.Baseline{
        P50:         parseFloat(fieldP50),
        P95:         parseFloat(fieldP95),
        MAD:         parseFloat(fieldMAD),
        SampleCount: parseInt(fieldSampleCount),
        UpdatedAt:   parseTime(fieldUpdatedAt),
    }
    return b, nil
}

// SetBaseline writes baseline stats to Redis hash at key.
func (c *Client) SetBaseline(ctx context.Context, key string, b store.Baseline) error {
    // Ensure UpdatedAt is set; if zero, set to now.
    if b.UpdatedAt.IsZero() {
        b.UpdatedAt = time.Now().UTC()
    }

    fields := map[string]interface{}{
        fieldP50:         strconv.FormatFloat(b.P50, 'f', -1, 64),
        fieldP95:         strconv.FormatFloat(b.P95, 'f', -1, 64),
        fieldMAD:         strconv.FormatFloat(b.MAD, 'f', -1, 64),
        fieldSampleCount: strconv.Itoa(b.SampleCount),
        fieldUpdatedAt:   b.UpdatedAt.Format(timeLayout),
    }
    return c.rdb.HSet(ctx, key, fields).Err()
}

