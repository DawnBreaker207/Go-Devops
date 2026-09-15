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

// T53: an unreachable Redis fails an operation quickly (bounded by the
// client's dial/op timeouts) with an error, rather than hanging — the
// service layer treats that error as a cache miss and reads the DB (see
// TestMovies_ReadsSucceedWhenRedisUnreachable in internal/service).
func TestCache_UnreachableFailsFastNotHang(t *testing.T) {
	c := cache.New("127.0.0.1:1", "", 0) // nothing listens on port 1
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()

	start := time.Now()
	_, _, err := c.Get(ctx, "unreachable")
	if err == nil {
		t.Fatal("get against an unreachable redis returned no error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("get took %v, want it bounded by the client's own dial/op timeout", elapsed)
	}
}

func TestCache_Incr(t *testing.T) {
	c := dialOrSkip(t)
	t.Cleanup(func() { _ = c.Close() })
	ctx := context.Background()
	key := "t53:gen:" + time.Now().Format(time.RFC3339Nano)
	t.Cleanup(func() { _ = c.Del(context.Background(), key) })

	first, err := c.Incr(ctx, key)
	if err != nil || first != 1 {
		t.Fatalf("first incr = %d, %v, want 1, nil", first, err)
	}
	second, err := c.Incr(ctx, key)
	if err != nil || second != 2 {
		t.Fatalf("second incr = %d, %v, want 2, nil", second, err)
	}

	var nilCache *cache.Cache
	if n, err := nilCache.Incr(ctx, key); err != nil || n != 0 {
		t.Fatalf("nil cache incr = %d, %v, want 0, nil", n, err)
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