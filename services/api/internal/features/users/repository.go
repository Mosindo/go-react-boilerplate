package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error)
	Get(ctx context.Context, organizationID, viewerID, userID string) (User, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// visibleSQL restricts the directory to members with a discoverable profile
// (or the viewer) and hides anyone involved in a block with the viewer.
const visibleSQL = `
	u.organization_id = $1
	AND (u.id = $2 OR p.discoverable)
	AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_user_id = $2 AND b.blocked_user_id = u.id)
		   OR (b.blocker_user_id = u.id AND b.blocked_user_id = $2)
	)`

func scanUser(row pgx.CollectableRow) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.FirstName, &u.CreatedAt)
	return u, err
}

func (r *PGRepository) List(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, p.first_name, u.created_at
		FROM users u
		JOIN profiles p ON p.user_id = u.id
		WHERE `+visibleSQL+`
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $3 OFFSET $4
	`, organizationID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanUser)
}

func (r *PGRepository) Get(ctx context.Context, organizationID, viewerID, userID string) (User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, p.first_name, u.created_at
		FROM users u
		JOIN profiles p ON p.user_id = u.id
		WHERE `+visibleSQL+` AND u.id = $3
	`, organizationID, viewerID, userID)
	if err != nil {
		return User{}, err
	}
	u, err := pgx.CollectExactlyOneRow(rows, scanUser)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}
