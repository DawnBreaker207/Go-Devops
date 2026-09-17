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

// dialTimeout/opTimeout bound a Redis brownout: TCP up but not answering
// must fail fast into the DB fallback, not stall the request.
const (
	dialTimeout = 500 * time.Millisecond
	opTimeout   = 300 * time.Millisecond
)

func New(addr, password string, db int) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{
		Addr: addr, Password: password, DB: db,
		DialTimeout: dialTimeout, ReadTimeout: opTimeout, WriteTimeout: opTimeout,
		// A cache miss must fail once and fall through to the DB, not retry a
		// dead connection several times first (go-redis retries 3 times by
		// default, multiplying the worst-case latency of a brownout).
		MaxRetries: -1,
	})}
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

// Incr atomically increments key (creating it at 1 if absent) and returns the
// new value; a nil cache always reads as generation 0, which still makes a
// stable (if unshared) cache key. Used as a generation counter: every list
// cache key embeds it, so bumping it invalidates every list key at once
// without a SCAN — the old ones just age out via TTL, unread.
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	if c == nil {
		return 0, nil
	}
	return c.rdb.Incr(ctx, key).Result()
}
// ZAdd appends a member to a sorted set (virtual queue's FIFO order); a nil
// Cache is a no-op returning ok=false so the caller can fail open.
func (c *Cache) ZAdd(ctx context.Context, key string, score float64, member string) (bool, error) {
	if c == nil {
		return false, nil
	}
	if err := c.rdb.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err(); err != nil {
		return false, err
	}
	return true, nil
}

// ZRank returns a member's 0-based rank (lowest score first) and whether it
// is present at all.
func (c *Cache) ZRank(ctx context.Context, key, member string) (int64, bool, error) {
	if c == nil {
		return 0, false, nil
	}
	rank, err := c.rdb.ZRank(ctx, key, member).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return rank, true, nil
}

// ZRem removes a member (leaving the queue, or having been admitted).
func (c *Cache) ZRem(ctx context.Context, key, member string) error {
	if c == nil {
		return nil
	}
	return c.rdb.ZRem(ctx, key, member).Err()
}

// Expire sets a TTL on an existing key so an abandoned queue does not live forever.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	return c.rdb.Expire(ctx, key, ttl).Err()
}
