package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrTooManyReports = errors.New("too many reports today")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (affectedConversations []string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Block records the block and dissolves any match with that user, which also
// deletes the shared conversation and its messages through foreign keys.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return nil, err
	}
	var conv []string
	rows, err := tx.Query(ctx, `
		DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
		RETURNING (SELECT id::text FROM conversations WHERE match_id = matches.id)`, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id *string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		if id != nil {
			conv = append(conv, *id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conv, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id::text, COALESCE(p.display_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT $2`, blockerID, maxBlocksPerUserLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.DisplayName, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, reportedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('reports:' || $1, 0))`, reporterID); err != nil {
		return err
	}
	var today int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND created_at > NOW() - INTERVAL '1 day'`, reporterID).Scan(&today); err != nil {
		return err
	}
	if today >= maxReportsPerDay {
		return ErrTooManyReports
	}
	if _, err := tx.Exec(ctx, `INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)`, reporterID, reportedID, reason, details); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
