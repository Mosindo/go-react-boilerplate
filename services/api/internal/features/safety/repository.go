package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoUserNotFound = errors.New("safety: user not found")

type Repository interface {
	// Block records the block and removes any match (and its conversation) between both users.
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrRepoUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM matches
		WHERE user_a = least($1::text, $2::text)::uuid AND user_b = greatest($1::text, $2::text)::uuid`, blockerID, blockedID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200`, blockerID)
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
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2
		RETURNING id`, reporterID, reportedID, reason, details).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepoUserNotFound
		}
		return "", err
	}
	return id, nil
}
