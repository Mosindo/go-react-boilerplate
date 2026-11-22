package safety

import (
	"context"
	"errors"

	"example.com/api/internal/features/matching"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func userExists(ctx context.Context, tx pgx.Tx, userID string) error {
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	return nil
}

// blockTx records the block and tears down any match and conversation.
func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) error {
	if err := userExists(ctx, tx, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`, blockerID, blockedID); err != nil {
		return err
	}
	return matching.DeleteMatchTx(ctx, tx, blockerID, blockedID, false)
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		return blockTx(ctx, tx, blockerID, blockedID)
	})
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT blocked_id, created_at FROM blocks WHERE blocker_id = $1
		ORDER BY created_at DESC, blocked_id LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedBlock, error) {
		var b storedBlock
		err := row.Scan(&b.UserID, &b.BlockedAt)
		return b, err
	})
}

// Report files a report (one open report per pair), blocks the reported
// member for the reporter and escalates to review past the threshold.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var reportID string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := blockTx(ctx, tx, reporterID, reportedID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO reports (reporter_id, reported_id, reason, details)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (reporter_id, reported_id) WHERE status = 'open' DO UPDATE
			  SET details = CASE WHEN EXCLUDED.details = '' THEN reports.details ELSE EXCLUDED.details END,
			      reason = EXCLUDED.reason
			RETURNING id
		`, reporterID, reportedID, reason, details).Scan(&reportID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE profiles SET moderation_status = 'under_review', updated_at = NOW()
			WHERE user_id = $1
			  AND moderation_status = 'active'
			  AND (SELECT COUNT(DISTINCT reporter_id) FROM reports WHERE reported_id = $1 AND status = 'open') >= $2
		`, reportedID, autoReviewThreshold)
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return reportID, err
}
