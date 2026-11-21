package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (hadMatch bool, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockRow, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) (string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) userExistsInTenant(ctx context.Context, q pgx.Tx, actorID, targetID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users t JOIN users a ON a.id = $1
			WHERE t.id = $2 AND t.organization_id = a.organization_id
		)
	`, actorID, targetID).Scan(&ok)
	return ok, err
}

// Block records the block, removes any match (and therefore the conversation
// by cascade) and drops notifications that reference the blocked member.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := blockInTx(ctx, tx, r, blockerID, blockedID); err != nil {
		return false, err
	}
	hadMatch, err := removeMatch(ctx, tx, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	return hadMatch, tx.Commit(ctx)
}

func blockInTx(ctx context.Context, tx pgx.Tx, r *PGRepository, blockerID, blockedID string) error {
	exists, err := r.userExistsInTenant(ctx, tx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_user_id, blocked_user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE (user_id = $1::uuid AND data->>'userId' = $2::text)
		   OR (user_id = $2::uuid AND data->>'userId' = $1::text)
	`, blockerID, blockedID)
	return err
}

func removeMatch(ctx context.Context, tx pgx.Tx, a, b string) (bool, error) {
	low, high := a, b
	if high < low {
		low, high = high, low
	}
	tag, err := tx.Exec(ctx, `DELETE FROM matches WHERE user_low_id = $1 AND user_high_id = $2`, low, high)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockRow, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT blocked_user_id, created_at FROM blocks
		WHERE blocker_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockRow{}
	for rows.Next() {
		var b BlockRow
		if err := rows.Scan(&b.UserID, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateReport snapshots the last messages exchanged (so moderators keep the
// evidence after the conversation is removed), stores the report, blocks the
// reported member and auto-suspends the profile once enough distinct members
// reported it.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	exists, err := r.userExistsInTenant(ctx, tx, reporterID, reportedID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrUserNotFound
	}

	var reportID string
	err = tx.QueryRow(ctx, `
		WITH evidence AS (
			SELECT COALESCE(jsonb_agg(jsonb_build_object(
				'senderUserId', recent.sender_user_id,
				'content', recent.content,
				'createdAt', recent.created_at
			) ORDER BY recent.created_at), '[]'::jsonb) AS items
			FROM (
				SELECT m.sender_user_id, m.content, m.created_at
				FROM messages m
				JOIN conversations c ON c.id = m.conversation_id
				WHERE c.direct_key = LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)
				ORDER BY m.created_at DESC
				LIMIT 30
			) recent
		)
		INSERT INTO reports (reporter_user_id, reported_user_id, reason, details, evidence)
		SELECT $1::uuid, $2::uuid, $3, $4, evidence.items FROM evidence
		ON CONFLICT (reporter_user_id, reported_user_id) WHERE status = 'open'
		DO UPDATE SET details = EXCLUDED.details, reason = EXCLUDED.reason
		RETURNING id
	`, reporterID, reportedID, reason, details).Scan(&reportID)
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE profiles SET is_suspended = TRUE, updated_at = NOW()
		WHERE user_id = $1
		  AND NOT is_suspended
		  AND (
			SELECT COUNT(DISTINCT reporter_user_id) FROM reports
			WHERE reported_user_id = $1 AND status = 'open'
		  ) >= $2
	`, reportedID, threshold); err != nil {
		return "", err
	}

	if err := blockInTx(ctx, tx, r, reporterID, reportedID); err != nil {
		return "", err
	}
	if _, err := removeMatch(ctx, tx, reporterID, reportedID); err != nil {
		return "", err
	}
	return reportID, tx.Commit(ctx)
}
