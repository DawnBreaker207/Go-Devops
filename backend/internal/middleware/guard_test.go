package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func guardedEngine(guard gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/x", guard, func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func status(engine *gin.Engine) int {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec.Code
}

// M18: one slow probe under load is not an outage.
func TestDBGuard_SingleTimeoutDoesNotTrip(t *testing.T) {
	var calls atomic.Int32
	engine := guardedEngine(newDBGuard(func(context.Context) error {
		if calls.Add(1) == 1 {
			return context.DeadlineExceeded
		}
		return nil
	}, 50*time.Millisecond, 0))
	for i := 0; i < 3; i++ {
		if got := status(engine); got != http.StatusOK {
			t.Fatalf("request %d: HTTP %d, want 200", i, got)
		}
	}
}

// M18: several timed-out probes in a row mean the database is down.
func TestDBGuard_ThreeTimeoutsTrip(t *testing.T) {
	engine := guardedEngine(newDBGuard(func(context.Context) error { return context.DeadlineExceeded }, 50*time.Millisecond, 0))
	for i := 0; i < 2; i++ {
		if got := status(engine); got != http.StatusOK {
			t.Fatalf("timeout %d: HTTP %d, want 200", i+1, got)
		}
	}
	if got := status(engine); got != http.StatusServiceUnavailable {
		t.Fatalf("third timeout: HTTP %d, want 503", got)
	}
}

// M18: a closed or refused database is down at once, and back as soon as it answers.
func TestDBGuard_HardErrorTripsImmediately(t *testing.T) {
	var broken atomic.Bool
	broken.Store(true)
	engine := guardedEngine(newDBGuard(func(context.Context) error {
		if broken.Load() {
			return sql.ErrConnDone
		}
		return nil
	}, 50*time.Millisecond, 0))
	if got := status(engine); got != http.StatusServiceUnavailable {
		t.Fatalf("closed database: HTTP %d, want 503", got)
	}
	broken.Store(false)
	if got := status(engine); got != http.StatusOK {
		t.Fatalf("database back: HTTP %d, want 200", got)
	}
}

// M18: concurrent requests share one probe instead of one ping each.
func TestDBGuard_ConcurrentRequestsPingOnce(t *testing.T) {
	var calls atomic.Int32
	engine := guardedEngine(newDBGuard(func(context.Context) error {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		return nil
	}, time.Second, time.Minute))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := status(engine); got != http.StatusOK {
				t.Errorf("HTTP %d, want 200", got)
			}
		}()
	}
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Fatalf("probes = %d, want 1", n)
	}
}
