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

// RateLimit rejects requests over the per-IP frequency cap with 429.
func RateLimit(lim *ratelimit.Limiter) gin.HandlerFunc {
	return rateLimitBy(lim, func(c *gin.Context) string { return c.ClientIP() })
}

// RateLimitByUser rejects requests over the per-user frequency cap with 429;
// anonymous requests are keyed by IP. Must run after Auth.
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
	// dbGuardFreshFor is how long one database probe answers for.
	dbGuardFreshFor = time.Second
	// dbGuardSlowProbes is how many timed-out probes in a row mean "down".
	dbGuardSlowProbes = 3
)

// DBGuard answers 503 fast while the database is down instead of letting every
// request hang or fail one by one.
func DBGuard(db *gorm.DB, timeout time.Duration) gin.HandlerFunc {
	return NewDBGuard(func(ctx context.Context) error { return database.Ping(ctx, db) }, timeout)
}

// NewDBGuard is DBGuard over any probe. The result is reused for a second and
// only one probe runs at a time, so a busy pool does not turn every request
// into an extra ping or into false 503s (M18): a closed or refused database is
// down at once, a slow one only after several timed-out probes in a row.
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
