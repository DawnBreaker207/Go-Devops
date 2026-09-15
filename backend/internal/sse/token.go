package sse

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const DefaultTokenTTL = 30 * time.Second

type tokenEntry struct {
	userID     string
	showtimeID string
	hallID     string
	expires    time.Time
}

// TokenStore issues showtime-bound stream tokens so no JWT goes in the URL. Tokens stay
// reusable until expiry so EventSource reconnects with the same URL still work.
type TokenStore struct {
	ttl time.Duration
	now func() time.Time

	mu        sync.Mutex
	tokens    map[string]tokenEntry
	nextPrune time.Time
}

// NewTokenStore uses DefaultTokenTTL when ttl <= 0 and time.Now when now is nil.
func NewTokenStore(ttl time.Duration, now func() time.Time) *TokenStore {
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	if now == nil {
		now = time.Now
	}
	return &TokenStore{ttl: ttl, now: now, tokens: make(map[string]tokenEntry)}
}

// Issue sweeps expired tokens at most once per TTL, not on every call.
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

// size is used by tests.
func (s *TokenStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tokens)
}
