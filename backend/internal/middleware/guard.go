package middleware

import (
	"context"
	"errors"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/database"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

func RateLimit(lim *ratelimit.Limiter) gin.HandlerFunc {
	return rateLimitBy(lim, func(c *gin.Context) string { return c.ClientIP() })
}

// RateLimitByUser must run after Auth.
func RateLimitByUser(lim *ratelimit.Limiter) gin.HandlerFunc {
	return rateLimitBy(lim, func(c *gin.Context) string {
		if id := CurrentUserID(c); id != "" {
			return "user:" + id
		}
		return c.ClientIP()
	})
}

func rateLimitBy(lim *ratelimit.Limiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !lim.Allow(key(c)) {
			wait := int(math.Ceil(lim.RetryAfter().Seconds()))
			if wait < 1 {
				wait = 1
			}
			response.Abort(c, apperrors.TooManyRequests("too many requests, try again later").
				WithDetails(map[string]string{"retry_after_seconds": strconv.Itoa(wait)}))
			return
		}
		c.Next()
	}
}

const (
	dbGuardFreshFor   = time.Second
	dbGuardSlowProbes = 3
)

func DBGuard(db *gorm.DB, timeout time.Duration) gin.HandlerFunc {
	return NewDBGuard(func(ctx context.Context) error { return database.Ping(ctx, db) }, timeout)
}

// Probe results are cached and only one probe runs at a time, so a busy pool causes
// neither extra pings nor false 503s. A refused DB is down at once, a slow one after several timeouts.
func NewDBGuard(probe func(ctx context.Context) error, timeout time.Duration) gin.HandlerFunc {
	return newDBGuard(probe, timeout, dbGuardFreshFor)
}

func newDBGuard(probe func(ctx context.Context) error, timeout, freshFor time.Duration) gin.HandlerFunc {
	g := &dbGuard{probe: probe, timeout: timeout, freshFor: freshFor}
	return func(c *gin.Context) {
		if g.down() {
			response.Abort(c, apperrors.ServiceUnavailable("service busy, try again later"))
			return
		}
		c.Next()
	}
}

type dbGuard struct {
	probe    func(ctx context.Context) error
	timeout  time.Duration
	freshFor time.Duration
	probing  atomic.Bool

	mu        sync.Mutex
	checkedAt time.Time
	isDown    bool
	slow      int
}

func (g *dbGuard) down() bool {
	g.mu.Lock()
	fresh, isDown := !g.checkedAt.IsZero() && time.Since(g.checkedAt) < g.freshFor, g.isDown
	g.mu.Unlock()
	if fresh || !g.probing.CompareAndSwap(false, true) {
		return isDown
	}
	defer g.probing.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	err := g.probe(ctx)
	cancel()

	g.mu.Lock()
	defer g.mu.Unlock()
	g.checkedAt = time.Now()
	switch {
	case err == nil:
		g.isDown, g.slow = false, 0
	case errors.Is(err, context.DeadlineExceeded):
		g.slow++
		if g.slow >= dbGuardSlowProbes {
			g.isDown = true
		}
	default:
		g.isDown, g.slow = true, 0
	}
	return g.isDown
}
