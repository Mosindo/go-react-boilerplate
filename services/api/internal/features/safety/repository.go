package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Block(ctx context.Context, blocker, blocked string) (*Removed, error)
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocks(ctx context.Context, blocker string) ([]BlockedUser, error)
	Report(ctx context.Context, reporter, reported, reason, details string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Block hides both users from each other and dissolves their match (and with it
// the conversation and its messages) atomically.
func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) (*Removed, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var removed Removed
	err = tx.QueryRow(ctx, `
		SELECT m.id, c.id FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)`, blocker, blocked).
		Scan(&removed.MatchID, &removed.ConversationID)
	switch {
	case err == nil:
		if _, err := tx.Exec(ctx, `DELETE FROM matches WHERE id = $1`, removed.MatchID); err != nil {
			return nil, err
		}
		return &removed, tx.Commit(ctx)
	case errors.Is(err, pgx.ErrNoRows):
		return nil, tx.Commit(ctx)
	default:
		return nil, err
	}
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	if isInvalidUUID(err) {
		return ErrInvalidInput
	}
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blocker string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = b.blocked_id ORDER BY ph.position LIMIT 1)
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt, &b.PhotoID); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Report stores a report together with a snapshot of the most recent messages
// exchanged, so moderators keep evidence even if the reporter then blocks. A
// repeated report for the same reason within 24h is ignored.
func (r *PGRepository) Report(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details, context)
		SELECT $1::uuid, $2::uuid, $3, $4,
		  COALESCE((
		    SELECT jsonb_agg(jsonb_build_object('senderId', s.sender_id, 'body', s.body, 'createdAt', s.created_at) ORDER BY s.created_at)
		    FROM (
		      SELECT msg.sender_id, msg.body, msg.created_at
		      FROM messages msg
		      JOIN conversations c ON c.id = msg.conversation_id
		      JOIN matches m ON m.id = c.match_id
		      WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
		      ORDER BY msg.created_at DESC LIMIT 20) s), '[]'::jsonb)
		WHERE NOT EXISTS (
		    SELECT 1 FROM reports
		    WHERE reporter_id = $1::uuid AND reported_id = $2::uuid AND reason = $3
		      AND created_at > NOW() - INTERVAL '24 hours')
	`, reporter, reported, reason, details)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrUserNotFound
		}
		if isInvalidUUID(err) {
			return ErrInvalidInput
		}
		return err
	}
	return nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
