package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func badID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "22P02" || pgErr.Code == "23503")
}

func (r *Repository) UserExists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1::uuid)`, id).Scan(&ok)
	if badID(err) {
		return false, nil
	}
	return ok, err
}

// Block records the block, removes any match (cascading to its conversation and messages) and
// clears notifications between the two users. It reports whether a match was removed.
func (r *Repository) Block(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)`, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE (user_id = $1 AND actor_id = $2) OR (user_id = $2 AND actor_id = $1)`, blockerID, blockedID); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		if badID(err) {
			return ErrNotFound
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 500
	`, blockerID)
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

// InsertReport stores a report unless the same open report was filed within the last 24 hours.
func (r *Repository) InsertReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
		  SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3
		    AND status = 'open' AND created_at > NOW() - INTERVAL '24 hours')
	`, reporterID, reportedID, reason, details)
	return err
}
