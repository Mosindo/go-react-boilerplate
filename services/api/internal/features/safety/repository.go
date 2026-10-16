package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type StoredBlock struct {
	BlockedID string
	CreatedAt time.Time
}

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	Blocks(ctx context.Context, blockerID string) ([]StoredBlock, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func userExists(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// Block hides both people from each other everywhere and dissolves any match, which
// also deletes the shared conversation. It is one transaction: no half-blocked state.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Same pair lock as swipes: a like in flight cannot create a match across a new block.
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::uuid, $2::uuid)::text || GREATEST($1::uuid, $2::uuid)::text, 11))
	`, blockerID, blockedID); err != nil {
		return err
	}
	if ok, err := userExists(ctx, tx, blockedID); err != nil {
		return err
	} else if !ok {
		return ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
	`, blockerID, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE (user_id = $1 AND data->>'userId' = $3) OR (user_id = $2 AND data->>'userId' = $4)
	`, blockerID, blockedID, blockedID, blockerID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) Blocks(ctx context.Context, blockerID string) ([]StoredBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT blocked_id::text, created_at FROM blocks WHERE blocker_id = $1 ORDER BY created_at DESC LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredBlock
	for rows.Next() {
		var b StoredBlock
		if err := rows.Scan(&b.BlockedID, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Report stores the report once per (reporter, reported, reason) while it is still
// open, so repeated taps do not flood moderators.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if ok, err := userExists(ctx, r.db, reportedID); err != nil {
		return err
	} else if !ok {
		return ErrUserNotFound
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details, context)
		SELECT $1, $2, $3, $4, COALESCE((
		  SELECT jsonb_agg(jsonb_build_object('sender', t.sender_id, 'body', t.body, 'at', t.created_at) ORDER BY t.created_at)
		  FROM (SELECT msg.sender_id, msg.body, msg.created_at
		        FROM messages msg JOIN matches m ON m.id = msg.match_id
		        WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
		        ORDER BY msg.created_at DESC LIMIT 20) t), '[]'::jsonb)
		WHERE NOT EXISTS (
		  SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3 AND status = 'open')
	`, reporterID, reportedID, reason, details)
	return err
}
