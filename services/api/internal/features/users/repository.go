package users

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListContacts(ctx context.Context, userID string, limit, offset int) ([]User, error)
	GetVisible(ctx context.Context, viewerID, userID string) (User, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func scanUser(row pgx.CollectableRow) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.FirstName, &u.CreatedAt)
	return u, err
}

// ListContacts returns the members the caller is matched with. A dating app
// must not expose a browsable member directory.
func (r *PGRepository) ListContacts(ctx context.Context, userID string, limit, offset int) ([]User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, COALESCE(p.first_name, ''), u.created_at
		FROM matches m
		JOIN users u ON u.id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE m.user_a_id = $1 OR m.user_b_id = $1
		ORDER BY m.created_at DESC, u.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanUser)
}

func (r *PGRepository) GetVisible(ctx context.Context, viewerID, userID string) (User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, COALESCE(p.first_name, ''), u.created_at
		FROM users u
		JOIN users viewer ON viewer.id = $1 AND viewer.organization_id = u.organization_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE u.id = $2
		  AND (u.id = $1 OR (
		    NOT `+profiles.SQLBlockedBetween("$1::uuid", "u.id")+`
		    AND `+profiles.SQLMatched("$1::uuid", "u.id")+`
		  ))
	`, viewerID, userID)
	if err != nil {
		return User{}, err
	}
	u, err := pgx.CollectExactlyOneRow(rows, scanUser)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}
