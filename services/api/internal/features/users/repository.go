package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListUsers(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error)
	GetUserByID(ctx context.Context, organizationID, viewerID, userID string) (User, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// notBlocked hides members involved in a block with the viewer, in either direction.
const notBlocked = `
	NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_user_id = $2 AND b.blocked_user_id = u.id)
		   OR (b.blocker_user_id = u.id AND b.blocked_user_id = $2)
	)`

func (r *PGRepository) ListUsers(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT u.id, u.display_name, u.organization_id, u.created_at
		FROM users u
		WHERE u.organization_id = $1
		  AND `+notBlocked+`
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $3 OFFSET $4
	`, organizationID, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.DisplayName, &user.OrganizationID, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *PGRepository) GetUserByID(ctx context.Context, organizationID, viewerID, userID string) (User, error) {
	var user User
	err := r.dbPool.QueryRow(ctx, `
		SELECT u.id, u.display_name, u.organization_id, u.created_at
		FROM users u
		WHERE u.organization_id = $1
		  AND u.id = $3
		  AND `+notBlocked+`
	`, organizationID, viewerID, userID).Scan(&user.ID, &user.DisplayName, &user.OrganizationID, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return user, nil
}
