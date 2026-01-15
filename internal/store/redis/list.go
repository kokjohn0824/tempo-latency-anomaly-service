package redis

import (
    "context"
    "strconv"
)

// ListBaselineKeys returns all baseline keys that have sufficient samples.
// It scans for keys matching "base:*" pattern and filters by minSamples.
func (c *Client) ListBaselineKeys(ctx context.Context, minSamples int) ([]string, error) {
    var result []string
    var cursor uint64
    
    // Use SCAN to iterate through all keys matching "base:*"
    for {
        keys, nextCursor, err := c.rdb.Scan(ctx, cursor, "base:*", 100).Result()
        if err != nil {
            return nil, err
        }
        
        // Check each key's sample count
        for _, key := range keys {
            sampleCountStr, err := c.rdb.HGet(ctx, key, fieldSampleCount).Result()
            if err != nil {
                // Skip keys that don't have sampleCount field or have errors
                continue
            }
            
            sampleCount, err := strconv.Atoi(sampleCountStr)
            if err != nil {
                continue
            }
            
            // Only include keys with sufficient samples
            if sampleCount >= minSamples {
                result = append(result, key)
            }
        }
        
        cursor = nextCursor
        if cursor == 0 {
            break
        }
    }
    
    return result, nil
}
