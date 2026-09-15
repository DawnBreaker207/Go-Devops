package ratelimit

import (
	"sync"
	"time"
)

// FailureLimiter locks a key (e.g. email+IP) out for a while after max
// consecutive failures: 5 wrong passwords -> 429 (T14). A success resets the
// key; failures older than the lockout window no longer count.
type FailureLimiter struct {
	max     int
	lockout time.Duration
	now     func() time.Time

	mu      sync.Mutex
	entries map[string]*failureEntry
}

type failureEntry struct {
	count       int
	last        time.Time
	lockedUntil time.Time
}

// NewFailureLimiter builds a limiter; a nil clock uses time.Now.
func NewFailureLimiter(max int, lockout time.Duration, now func() time.Time) *FailureLimiter {
	if max <= 0 {
		max = 5
	}
	if lockout <= 0 {
		lockout = 5 * time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &FailureLimiter{max: max, lockout: lockout, now: now, entries: make(map[string]*failureEntry)}
}

// Blocked reports whether key is locked out and for how much longer.
func (l *FailureLimiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok {
		return false, 0
	}
	now := l.now()
	if now.Before(e.lockedUntil) {
		return true, e.lockedUntil.Sub(now)
	}
	return false, 0
}

// Fail records a failure and reports whether it started a lockout.
func (l *FailureLimiter) Fail(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= 10_000 {
			l.prune(now)
		}
		e = &failureEntry{}
		l.entries[key] = e
	}
	if now.Sub(e.last) > l.lockout {
		e.count = 0
	}
	e.last = now
	e.count++
	if e.count >= l.max {
		e.count = 0
		e.lockedUntil = now.Add(l.lockout)
		return true
	}
	return false
}

// Reset forgets the failures of key (successful login).
func (l *FailureLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

func (l *FailureLimiter) prune(now time.Time) {
	for key, e := range l.entries {
		if now.Sub(e.last) > l.lockout && !now.Before(e.lockedUntil) {
			delete(l.entries, key)
		}
	}
}
