package redis

import (
    "context"
    "strconv"
    "time"

    goRedis "github.com/redis/go-redis/v9"

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

// GetBaselines reads multiple baseline hashes using a Redis pipeline for efficiency.
// Returns a map for keys that exist (missing keys are omitted).
func (c *Client) GetBaselines(ctx context.Context, keys []string) (map[string]*store.Baseline, error) {
    if len(keys) == 0 {
        return map[string]*store.Baseline{}, nil
    }

    pipe := c.rdb.Pipeline()
    cmds := make([]*goRedis.MapStringStringCmd, len(keys))

    // Queue HGETALL for each key in the pipeline
    for i, k := range keys {
        cmds[i] = pipe.HGetAll(ctx, k)
    }

    if _, err := pipe.Exec(ctx); err != nil {
        return nil, err
    }

    out := make(map[string]*store.Baseline, len(keys))
    for i, k := range keys {
        m, err := cmds[i].Result()
        if err != nil {
            // If a specific command errors, return the error for visibility.
            return nil, err
        }
        if len(m) == 0 {
            continue
        }

        parseFloat := func(field string) float64 {
            if v, ok := m[field]; ok {
                if f, err := strconv.ParseFloat(v, 64); err == nil {
                    return f
                }
            }
            return 0
        }
        parseInt := func(field string) int {
            if v, ok := m[field]; ok {
                if n, err := strconv.Atoi(v); err == nil {
                    return n
                }
            }
            return 0
        }
        parseTime := func(field string) time.Time {
            if v, ok := m[field]; ok {
                if t, err := time.Parse(timeLayout, v); err == nil {
                    return t
                }
            }
            return time.Time{}
        }

        out[k] = &store.Baseline{
            P50:         parseFloat(fieldP50),
            P95:         parseFloat(fieldP95),
            MAD:         parseFloat(fieldMAD),
            SampleCount: parseInt(fieldSampleCount),
            UpdatedAt:   parseTime(fieldUpdatedAt),
        }
    }

    return out, nil
}
