package service

import (
	"context"
	"sync"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// AccountStatusCache tells the auth middleware whether an account is still
// active and with which role. Answers are kept for ttl so authenticated
// requests do not all read the users table; changes made through UserService
// drop the entry at once. In memory: the server runs as a single instance.
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

// maxAccountStatusEntries bounds the cache; expired entries are pruned past it.
const maxAccountStatusEntries = 10000

func NewAccountStatusCache(repo repository.UserRepository, ttl time.Duration) *AccountStatusCache {
	return &AccountStatusCache{repo: repo, ttl: ttl, entries: make(map[string]accountStatus)}
}

// Status implements middleware.AccountChecker.
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

// Invalidate forgets an account so its next request reads the database.
func (c *AccountStatusCache) Invalidate(userID string) {
	c.mu.Lock()
	delete(c.entries, userID)
	c.mu.Unlock()
}
