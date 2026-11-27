package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	UserInOrganization(ctx context.Context, organizationID, userID string) (bool, error)
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
	AutoHideIfNeeded(ctx context.Context, reportedID string, threshold int, window time.Duration) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) UserInOrganization(ctx context.Context, organizationID, userID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND organization_id = $2)`, userID, organizationID).Scan(&ok)
	return ok, err
}

// Block is idempotent and also ends any match between the two users (the
// conversation cascades) so a blocked person can no longer write.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_user_id, blocked_user_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", err
	}
	var matchID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
		RETURNING id
	`, blockerID, blockedID).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return matchID, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	tag, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT b.blocked_user_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_user_id
		WHERE b.blocker_user_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
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

// CreateReport stores the report with a snapshot of the conversation (if
// any) so moderators keep context after an unmatch or block. Duplicate open
// reports from the same reporter about the same person are ignored.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	_, err := r.dbPool.Exec(ctx, `
		WITH conv AS (
			SELECT c.id
			FROM matches m
			JOIN conversations c ON c.match_id = m.id
			WHERE m.user_low_id = LEAST($1::uuid, $2::uuid)
			  AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
		),
		recent AS (
			SELECT msg.sender_user_id, msg.content, msg.created_at
			FROM messages msg
			WHERE msg.conversation_id = (SELECT id FROM conv)
			ORDER BY msg.created_at DESC
			LIMIT $5
		)
		INSERT INTO reports (reporter_user_id, reported_user_id, reason, details, evidence)
		SELECT $1, $2, $3, $4, jsonb_build_object(
			'messages', COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'from', CASE WHEN sender_user_id = $1::uuid THEN 'reporter' ELSE 'reported' END,
					'content', content,
					'at', created_at
				) ORDER BY created_at)
				FROM recent
			), '[]'::jsonb),
			'capturedAt', NOW()
		)
		ON CONFLICT (reporter_user_id, reported_user_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details, EvidenceMessageLimit)
	return err
}

func (r *PGRepository) AutoHideIfNeeded(ctx context.Context, reportedID string, threshold int, window time.Duration) (bool, error) {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET moderation_hidden_at = NOW()
		WHERE user_id = $1
		  AND moderation_hidden_at IS NULL
		  AND (
			SELECT COUNT(DISTINCT reporter_user_id)
			FROM reports
			WHERE reported_user_id = $1
			  AND status = 'open'
			  AND created_at > NOW() - make_interval(secs => $3)
		  ) >= $2
	`, reportedID, threshold, window.Seconds())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
