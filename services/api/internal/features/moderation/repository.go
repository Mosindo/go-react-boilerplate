package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSelf     = errors.New("you cannot do this to yourself")
	ErrNotFound = errors.New("user not found")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "22P02" || pgErr.Code == "23503") {
		return ErrNotFound
	}
	return err
}

// Block hides both people from each other and dissolves any match (conversation
// and messages go with it through ON DELETE CASCADE).
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelf
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return mapErr(err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM matches WHERE user_a = least($1::uuid, $2::uuid) AND user_b = greatest($1::uuid, $2::uuid)
	`, blockerID, blockedID); err != nil {
		return mapErr(err)
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return mapErr(err)
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT $2
	`, blockerID, limit)
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

// Report stores a report with a snapshot of the latest messages exchanged by
// the pair (if any) so moderators keep context even if the match is later removed.
// A repeated open report from the same reporter against the same user is a no-op.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return ErrSelf
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, reportedID).Scan(&exists); err != nil {
		return mapErr(err)
	}
	if !exists {
		return ErrNotFound
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details, evidence)
		SELECT $1, $2, $3, $4, COALESCE((
		  SELECT jsonb_agg(jsonb_build_object('from', CASE WHEN sender_id = $1 THEN 'reporter' ELSE 'reported' END,
		                                      'body', body, 'at', created_at) ORDER BY created_at)
		  FROM (
		    SELECT msg.sender_id, msg.body, msg.created_at
		    FROM matches m
		    JOIN conversations c ON c.match_id = m.id
		    JOIN messages msg ON msg.conversation_id = c.id
		    WHERE m.user_a = least($1::uuid, $2::uuid) AND m.user_b = greatest($1::uuid, $2::uuid)
		    ORDER BY msg.created_at DESC LIMIT 20
		  ) recent
		), '[]'::jsonb)
		WHERE NOT EXISTS (
		  SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'
		)
	`, reporterID, reportedID, reason, details)
	return mapErr(err)
}
