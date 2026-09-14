// Package ratelimit provides an in-memory token bucket (zero dependencies)
// to throttle sensitive endpoints: brute-force on /auth, seat-hold bots.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Limiter is a token bucket keyed by client (default: IP).
type Limiter struct {
	mu       sync.Mutex
	capacity float64
	refill   float64 // tokens added per second
	buckets  map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New builds a Limiter with a burst capacity and a refill rate (tokens/sec).
func New(capacity int, refillPerSecond float64) *Limiter {
	return &Limiter{
		capacity: float64(capacity),
		refill:   refillPerSecond,
		buckets:  make(map[string]*bucket),
	}
}

// Allow reports whether the key still has tokens, consuming one if it does.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.maybePrune(now)
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}

	b.tokens = math.Min(l.capacity, b.tokens+now.Sub(b.last).Seconds()*l.refill)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ponytail: the bucket map never shrinks; fine for a single instance, needs a
// shared store (Redis) for a multi-instance setup.
func (l *Limiter) maybePrune(now time.Time) {
	if len(l.buckets) < 10_000 {
		return
	}
	for key, b := range l.buckets {
		if now.Sub(b.last) > 10*time.Minute {
			delete(l.buckets, key)
		}
	}
}