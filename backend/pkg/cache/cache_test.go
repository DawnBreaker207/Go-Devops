package cache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
)

func redisAddr() string {
	if a := os.Getenv("TEST_REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6379"
}

func dialOrSkip(t *testing.T) *cache.Cache {
	t.Helper()
	c := cache.New(redisAddr(), "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx); err != nil {
		t.Skipf("redis unavailable at %s: %v", redisAddr(), err)
	}
	return c
}

// T53: a cache round-trips values live in Redis and honours the TTL and Deletes.
func TestCache_SetGetExpireDel(t *testing.T) {
	c := dialOrSkip(t)
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	v, ok, err := c.Get(ctx, "t53:miss")
	if err != nil || ok || v != "" {
		t.Fatalf("miss = (%q,%v,%v), want none", v, ok, err)
	}

	if err := c.Set(ctx, "t53:key", `{"hello":"world"}`, 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, ok, err = c.Get(ctx, "t53:key")
	if err != nil || !ok || v != `{"hello":"world"}` {
		t.Fatalf("read = (%q,%v,%v)", v, ok, err)
	}

	if err := c.Set(ctx, "t53:ttl", "brief", 60*time.Millisecond); err != nil {
		t.Fatalf("set ttl: %v", err)
	}
	<-time.After(120 * time.Millisecond)
	if _, ok, err := c.Get(ctx, "t53:ttl"); err != nil || ok {
		t.Fatalf("expired value still readable (ok=%v err=%v)", ok, err)
	}

	if err := c.Del(ctx, "t53:key", "t53:ttl"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, ok, err := c.Get(ctx, "t53:key"); err != nil || ok {
		t.Fatalf("deleted value still readable (ok=%v err=%v)", ok, err)
	}
}

func TestCache_NilIsNoop(t *testing.T) {
	var c *cache.Cache
	ctx := context.Background()
	if err := c.Set(ctx, "k", "v", 0); err != nil {
		t.Fatalf("nil set: %v", err)
	}
	if v, ok, err := c.Get(ctx, "k"); err != nil || ok || v != "" {
		t.Fatalf("nil get = (%q,%v,%v)", v, ok, err)
	}
	if err := c.Del(ctx, "k"); err != nil {
		t.Fatalf("nil del: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("nil close: %v", err)
	}
}