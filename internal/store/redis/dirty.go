package redis

import (
    "context"
)

const dirtySetKey = "dirtyKeys"

// MarkDirty adds key to the global dirty set.
func (c *Client) MarkDirty(ctx context.Context, key string) error {
    return c.rdb.SAdd(ctx, dirtySetKey, key).Err()
}

// PopDirtyBatch pops up to count keys from the dirty set.
func (c *Client) PopDirtyBatch(ctx context.Context, count int64) ([]string, error) {
    if count <= 0 {
        count = 1
    }
    res, err := c.rdb.SPopN(ctx, dirtySetKey, count).Result()
    if err != nil {
        return nil, err
    }
    return res, nil
}

