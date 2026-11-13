package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	Block(ctx context.Context, blockerID, blockedID string) (hadMatch bool, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold, underageThreshold int) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&ok)
	return ok, err
}

func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (bool, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_user_id, blocked_user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE matches
		SET unmatched_at = NOW(), unmatched_by_user_id = $1
		WHERE user_a_id = LEAST($1::uuid, $2::uuid)
		  AND user_b_id = GREATEST($1::uuid, $2::uuid)
		  AND unmatched_at IS NULL
	`, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Block records the block and ends any active match in one transaction.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	hadMatch, err := blockTx(ctx, tx, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	return hadMatch, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_user_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_user_id
		WHERE b.blocker_user_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockedUser, 0)
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Report files the report, blocks the reported member for the reporter and
// applies the automatic moderation rules, atomically.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold, underageThreshold int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_user_id, reported_user_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_user_id, reported_user_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details); err != nil {
		return err
	}
	if _, err := blockTx(ctx, tx, reporterID, reportedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE profiles SET moderation_hidden = TRUE, updated_at = NOW()
		WHERE user_id = $1
		  AND NOT moderation_hidden
		  AND (
			(SELECT COUNT(DISTINCT reporter_user_id) FROM reports
			 WHERE reported_user_id = $1 AND status = 'open') >= $2
			OR
			(SELECT COUNT(DISTINCT reporter_user_id) FROM reports
			 WHERE reported_user_id = $1 AND status = 'open' AND reason = 'underage') >= $3
		  )
	`, reportedID, threshold, underageThreshold); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
