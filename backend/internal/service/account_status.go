package service

import (
	"context"
	"sync"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// AccountStatusCache keeps account status for ttl; UserService changes drop the entry at once.
// In memory only: the server runs as a single instance.
type AccountStatusCache struct {
	repo repository.UserRepository
	ttl  time.Duration

	mu      sync.Mutex
	entries map[string]accountStatus
}

type accountStatus struct {
	active  bool
	role    string
	expires time.Time
}

const maxAccountStatusEntries = 10000

func NewAccountStatusCache(repo repository.UserRepository, ttl time.Duration) *AccountStatusCache {
	return &AccountStatusCache{repo: repo, ttl: ttl, entries: make(map[string]accountStatus)}
}

func (c *AccountStatusCache) Status(ctx context.Context, userID string) (bool, string, error) {
	now := time.Now()
	c.mu.Lock()
	e, ok := c.entries[userID]
	c.mu.Unlock()
	if ok && now.Before(e.expires) {
		return e.active, e.role, nil
	}

	active, role, found, err := c.repo.StatusByID(ctx, userID)
	if err != nil {
		return false, "", apperrors.ServiceUnavailable("service busy, try again later").Wrap(err)
	}
	if !found {
		return false, "", apperrors.ErrInvalidToken
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= maxAccountStatusEntries {
		for id, old := range c.entries {
			if !now.Before(old.expires) {
				delete(c.entries, id)
			}
		}
	}
	c.entries[userID] = accountStatus{active: active, role: role, expires: now.Add(c.ttl)}
	return active, role, nil
}

func (c *AccountStatusCache) Invalidate(userID string) {
	c.mu.Lock()
	delete(c.entries, userID)
	c.mu.Unlock()
}
