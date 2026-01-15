package redis

import (
    "context"
    "strconv"
)

// AppendDuration pushes durationMs to the list at key and trims to windowSize.
func (c *Client) AppendDuration(ctx context.Context, key string, durationMs int64, windowSize int) error {
    if err := c.rdb.LPush(ctx, key, durationMs).Err(); err != nil {
        return err
    }
    // Keep only the most recent windowSize items (list is newest at head due to LPUSH)
    return c.rdb.LTrim(ctx, key, 0, int64(windowSize-1)).Err()
}

// GetDurations returns all duration samples (ms) from the list at key.
func (c *Client) GetDurations(ctx context.Context, key string) ([]int64, error) {
    vals, err := c.rdb.LRange(ctx, key, 0, -1).Result()
    if err != nil {
        return nil, err
    }
    out := make([]int64, 0, len(vals))
    for _, s := range vals {
        if s == "" {
            continue
        }
        n, err := strconv.ParseInt(s, 10, 64)
        if err != nil {
            // skip malformed entries rather than failing entire read
            continue
        }
        out = append(out, n)
    }
    return out, nil
}

