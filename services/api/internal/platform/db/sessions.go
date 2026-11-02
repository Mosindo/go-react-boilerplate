package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionChecker implements middleware.SessionChecker on top of the pool. It
// also refreshes users.last_active_at at most every five minutes.
type SessionChecker struct{ pool *pgxpool.Pool }

func NewSessionChecker(pool *pgxpool.Pool) *SessionChecker { return &SessionChecker{pool: pool} }

func (s *SessionChecker) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sessions
			WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		)`, sessionID, userID).Scan(&ok)
	if err != nil || !ok {
		return false, err
	}
	_, _ = s.pool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'`, userID)
	return true, nil
}
