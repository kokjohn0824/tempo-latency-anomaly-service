package redis

import (
    "context"
    "fmt"

    goRedis "github.com/redis/go-redis/v9"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Client implements store.Store backed by Redis.
type Client struct {
    rdb *goRedis.Client
}

// New creates a new Redis client using application config and verifies connectivity.
func New(cfg config.RedisConfig) (store.Store, error) {
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    rdb := goRedis.NewClient(&goRedis.Options{
        Addr:     addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    if err := rdb.Ping(context.Background()).Err(); err != nil {
        return nil, fmt.Errorf("redis ping: %w", err)
    }

    return &Client{rdb: rdb}, nil
}

// Close releases Redis resources.
func (c *Client) Close() error {
    if c == nil || c.rdb == nil {
        return nil
    }
    return c.rdb.Close()
}

