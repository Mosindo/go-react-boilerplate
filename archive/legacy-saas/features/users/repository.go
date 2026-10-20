package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StoredUser struct {
	ID             string
	Email          string
	OrganizationID string
	CreatedAt      time.Time
}

type Repository interface {
	ListUsers(ctx context.Context, organizationID string, limit, offset int) ([]StoredUser, error)
	GetUserByID(ctx context.Context, organizationID, userID string) (StoredUser, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListUsers(ctx context.Context, organizationID string, limit, offset int) ([]StoredUser, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT u.id, u.email, u.organization_id, u.created_at
		FROM users u
		WHERE u.organization_id = $1
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]StoredUser, 0)
	for rows.Next() {
		var user StoredUser
		if err := rows.Scan(&user.ID, &user.Email, &user.OrganizationID, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return users, nil
}

func (r *PGRepository) GetUserByID(ctx context.Context, organizationID, userID string) (StoredUser, error) {
	var user StoredUser
	err := r.dbPool.QueryRow(ctx, `
		SELECT id, email, organization_id, created_at
		FROM users
		WHERE organization_id = $1
		  AND id = $2
	`, organizationID, userID).Scan(&user.ID, &user.Email, &user.OrganizationID, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrUserNotFound
		}
		return StoredUser{}, err
	}
	return user, nil
}
