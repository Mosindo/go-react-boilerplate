package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("safety: user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id)
		SELECT $1, u.id FROM users u WHERE u.id = $2
		ON CONFLICT DO NOTHING`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Either already blocked (idempotent success) or the user does not exist.
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrUserNotFound
		}
	}
	return nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC, b.blocked_id
		LIMIT $2`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		b.BlockedAt = b.BlockedAt.UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2
		RETURNING id`, reporterID, reportedID, reason, details).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return id, err
}
