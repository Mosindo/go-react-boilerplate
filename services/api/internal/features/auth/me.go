package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrUserNotFound is returned when the account does not exist (anymore).
var ErrUserNotFound = errors.New("user not found")

// Querier is satisfied by *pgxpool.Pool and pgx.Tx.
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// LoadMe builds the Me payload with a single query. profileComplete means: profile row AND
// at least one photo AND a location set.
func LoadMe(ctx context.Context, q Querier, userID string, now time.Time) (MeResponse, error) {
	var (
		me    MeResponse
		birth *time.Time
	)
	err := q.QueryRow(ctx, `
		SELECT u.id, u.email, u.birth_date, u.created_at,
		       EXISTS (SELECT 1 FROM profiles p
		               WHERE p.user_id = u.id AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL)
		       AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = u.id)
		FROM users u
		WHERE u.id = $1`, userID).Scan(&me.ID, &me.Email, &birth, &me.CreatedAt, &me.ProfileComplete)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MeResponse{}, ErrUserNotFound
		}
		return MeResponse{}, err
	}
	if birth != nil {
		me.BirthDate = birth.UTC().Format("2006-01-02")
		me.Age = AgeOn(*birth, now)
	}
	return me, nil
}
