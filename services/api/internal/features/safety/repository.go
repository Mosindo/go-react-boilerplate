package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	UserInOrganization(ctx context.Context, organizationID, userID string) (bool, error)
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) UserInOrganization(ctx context.Context, organizationID, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND organization_id = $2)`, userID, organizationID).Scan(&ok)
	return ok, err
}

// Block records the block and tears down any match and conversation between
// the pair, so neither side can see or message the other anymore.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_user_id, blocked_user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", err
	}

	var matchID string
	var conversationID *string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
		RETURNING id, conversation_id
	`, blockerID, blockedID).Scan(&matchID, &conversationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if conversationID != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, *conversationID); err != nil {
			return "", err
		}
	}
	// Make sure the blocker's decision is a pass so an unblock never resurrects a like.
	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_user_id, target_user_id, decision) VALUES ($1, $2, 'pass')
		ON CONFLICT (actor_user_id, target_user_id) DO UPDATE SET decision = 'pass', updated_at = NOW()
	`, blockerID, blockedID); err != nil {
		return "", err
	}
	return matchID, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT blocked_user_id, created_at FROM blocks
		WHERE blocker_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
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

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	// One open report per pair: repeated reports are absorbed rather than duplicated.
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_user_id, reported_user_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_user_id, reported_user_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details)
	return err
}
