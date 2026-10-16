package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionChecker tells whether a session may still be used. It lets logout, password
// reset and suspension take effect on access tokens that have not expired yet.
type SessionChecker interface {
	Active(ctx context.Context, sessionID string) (bool, error)
}

// PGSessionChecker queries the database and remembers positive answers briefly, so a
// burst of requests costs one lookup. Revocation therefore takes effect within ttl.
type PGSessionChecker struct {
	db  *pgxpool.Pool
	ttl time.Duration

	mu    sync.Mutex
	cache map[string]time.Time
}

func NewPGSessionChecker(db *pgxpool.Pool, ttl time.Duration) *PGSessionChecker {
	return &PGSessionChecker{db: db, ttl: ttl, cache: make(map[string]time.Time)}
}

func (c *PGSessionChecker) Active(ctx context.Context, sessionID string) (bool, error) {
	now := time.Now()
	c.mu.Lock()
	if until, ok := c.cache[sessionID]; ok && now.Before(until) {
		c.mu.Unlock()
		return true, nil
	}
	c.mu.Unlock()

	var active bool
	err := c.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM sessions s JOIN users u ON u.id = s.user_id
		  WHERE s.id = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW() AND u.banned_at IS NULL)
	`, sessionID).Scan(&active)
	if err != nil || !active {
		return false, err
	}

	c.mu.Lock()
	if len(c.cache) > 50_000 {
		for k, until := range c.cache {
			if now.After(until) {
				delete(c.cache, k)
			}
		}
	}
	c.cache[sessionID] = now.Add(c.ttl)
	c.mu.Unlock()
	return true, nil
}
