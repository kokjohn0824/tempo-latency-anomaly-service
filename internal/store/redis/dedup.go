package redis

import (
    "context"
    "fmt"
    "time"
)

// IsDuplicateOrMark uses SETNX with TTL semantics to deduplicate trace IDs.
// Returns true if the traceID has been seen (duplicate), false if newly marked.
func (c *Client) IsDuplicateOrMark(ctx context.Context, traceID string, ttl time.Duration) (bool, error) {
    key := fmt.Sprintf("seen:%s", traceID)
    ok, err := c.rdb.SetNX(ctx, key, "1", ttl).Result()
    if err != nil {
        return false, err
    }
    // ok == true => it was not present and is now set => not a duplicate
    return !ok, nil
}

