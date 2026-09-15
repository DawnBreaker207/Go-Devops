// Package cache is a thin Redis-backed key/value store for reads that are
// expensive on PostgreSQL and safe when slightly stale. The interface keeps
// callers healthy with nothing connected: services accept nil and skip caching.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache stores string values with a TTL. nil *Cache is a legal no-op.
type Cache struct {
	rdb *redis.Client
}

func New(addr, password string, db int) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})}
}

// FromClient wraps an already-configured client (tests, composable setups).
func FromClient(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

func (c *Cache) Close() error {
	if c == nil {
		return nil
	}
	return c.rdb.Close()
}

// Ping verifies the Redis connection is alive (used for health checks, skips).
func (c *Cache) Ping(ctx context.Context) error {
	if c == nil {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

// Get returns the value and whether it was present; a miss returns "", false.
func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	if c == nil {
		return "", false, nil
	}
	v, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// Set stores a value with a TTL (0 keeps it forever).
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Del removes keys so the next read refetches.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	if c == nil {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}