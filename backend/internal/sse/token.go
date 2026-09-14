package sse

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// DefaultTokenTTL is how long a realtime token opens its stream.
const DefaultTokenTTL = 30 * time.Second

type tokenEntry struct {
	userID     string
	showtimeID string
	hallID     string
	expires    time.Time
}

// TokenStore issues realtime tokens: bound to one showtime, reusable until
// they expire so EventSource reconnects with the same URL still work (T20),
// and never a JWT in the URL (R-S2).
type TokenStore struct {
	ttl time.Duration
	now func() time.Time

	mu        sync.Mutex
	tokens    map[string]tokenEntry
	nextPrune time.Time
}

// NewTokenStore builds a store; ttl <= 0 uses DefaultTokenTTL and a nil clock
// uses time.Now.
func NewTokenStore(ttl time.Duration, now func() time.Time) *TokenStore {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	if now == nil {
		now = time.Now
	}
	return &TokenStore{ttl: ttl, now: now, tokens: make(map[string]tokenEntry)}
}

// Issue mints a token for one user watching one showtime. Expired tokens are
// swept at most once per TTL, not on every call (M13).
func (s *TokenStore) Issue(userID, showtimeID, hallID string) (string, time.Duration) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic("read rand: " + err.Error())
	}
	token := hex.EncodeToString(buf)

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if !now.Before(s.nextPrune) {
		for t, e := range s.tokens {
			if !now.Before(e.expires) {
				delete(s.tokens, t)
			}
		}
		s.nextPrune = now.Add(s.ttl)
	}
	s.tokens[token] = tokenEntry{userID: userID, showtimeID: showtimeID, hallID: hallID, expires: now.Add(s.ttl)}
	return token, s.ttl
}

// Validate accepts a live token for the showtime it was issued for and
// returns that showtime's hall and the user the token belongs to.
func (s *TokenStore) Validate(token, showtimeID string) (hallID, userID string, ok bool) {
	if token == "" {
		return "", "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, found := s.tokens[token]
	if !found {
		return "", "", false
	}
	if !s.now().Before(e.expires) {
		delete(s.tokens, token)
		return "", "", false
	}
	if e.showtimeID != showtimeID {
		return "", "", false
	}
	return e.hallID, e.userID, true
}

// size counts stored tokens (tests).
func (s *TokenStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tokens)
}
