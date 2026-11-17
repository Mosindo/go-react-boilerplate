package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrBlockNotFound   = errors.New("block not found")
	ErrInvalidReason   = errors.New("invalid report reason")
	ErrDetailsTooLong  = errors.New("report details too long")
	ErrCannotTargetYou = errors.New("cannot target yourself")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*EndedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (*EndedMatch, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (*EndedMatch, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`, blockerID, blockedID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	var ended EndedMatch
	err := tx.QueryRow(ctx, `
		UPDATE matches m SET ended_at = NOW(), ended_by = $1
		FROM conversations c
		WHERE c.match_id = m.id
		  AND m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
		  AND m.ended_at IS NULL
		RETURNING m.id, c.id
	`, blockerID, blockedID).Scan(&ended.MatchID, &ended.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ended, nil
}

// Block hides both members from each other everywhere and ends any active match.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*EndedMatch, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ended, err := blockTx(ctx, tx, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	return ended, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	tag, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBlockNotFound
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
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

// Report stores a moderation report and blocks the reported member in the same
// transaction: a report always protects the reporter immediately.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (*EndedMatch, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, reportedID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_id) WHERE status = 'open'
		DO UPDATE SET details = CASE WHEN EXCLUDED.details = '' THEN reports.details ELSE EXCLUDED.details END,
		              reason = EXCLUDED.reason
	`, reporterID, reportedID, reason, details); err != nil {
		return nil, err
	}
	ended, err := blockTx(ctx, tx, reporterID, reportedID)
	if err != nil {
		return nil, err
	}
	return ended, tx.Commit(ctx)
}
