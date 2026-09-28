package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionChecker implements middleware.SessionChecker on top of the sessions
// table: the session must be unrevoked and unexpired and its user must exist.
type SessionChecker struct{ pool *pgxpool.Pool }

func NewSessionChecker(pool *pgxpool.Pool) *SessionChecker { return &SessionChecker{pool: pool} }

func (s *SessionChecker) SessionActive(ctx context.Context, sessionID, userID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sessions s
			JOIN users u ON u.id = s.user_id
			WHERE s.id = $1 AND s.user_id = $2
			  AND s.revoked_at IS NULL AND s.expires_at > NOW()
		)`, sessionID, userID).Scan(&ok)
	return ok, err
}
