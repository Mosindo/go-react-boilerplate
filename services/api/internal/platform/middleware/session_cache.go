package middleware

import (
	"context"
	"sync"
	"time"
)

// CachedSessionChecker memoizes positive answers for a short TTL so the
// session lookup does not add a query to every request. Revocations therefore
// take effect within ttl.
type CachedSessionChecker struct {
	inner SessionChecker
	ttl   time.Duration
	now   func() time.Time

	mu      sync.Mutex
	entries map[string]time.Time
}

func NewCachedSessionChecker(inner SessionChecker, ttl time.Duration) *CachedSessionChecker {
	return &CachedSessionChecker{inner: inner, ttl: ttl, now: time.Now, entries: make(map[string]time.Time)}
}

func (c *CachedSessionChecker) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	if c.ttl <= 0 {
		return c.inner.SessionActive(ctx, sessionID, userID)
	}
	key := sessionID + ":" + userID
	now := c.now()

	c.mu.Lock()
	if until, ok := c.entries[key]; ok && now.Before(until) {
		c.mu.Unlock()
		return true, nil
	}
	c.mu.Unlock()

	active, err := c.inner.SessionActive(ctx, sessionID, userID)
	if err != nil || !active {
		return false, err
	}

	c.mu.Lock()
	if len(c.entries) > 10_000 { // bound memory: drop expired, then everything if still huge
		for k, until := range c.entries {
			if !now.Before(until) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) > 10_000 {
			c.entries = make(map[string]time.Time)
		}
	}
	c.entries[key] = now.Add(c.ttl)
	c.mu.Unlock()
	return true, nil
}
