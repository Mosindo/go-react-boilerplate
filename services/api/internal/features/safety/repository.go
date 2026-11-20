package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*RemovedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, threshold int, window time.Duration) (*RemovedMatch, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (*RemovedMatch, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`, blockerID, blockedID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrRepositoryUserNotFound
		}
		return nil, err
	}
	// Blocking ends the match: the conversation and its messages are removed.
	var matchID, conversationID *string
	if err := tx.QueryRow(ctx, `
		WITH m AS (
			SELECT id FROM matches
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		), conv AS (
			SELECT c.id FROM conversations c WHERE c.match_id IN (SELECT id FROM m)
		), deleted AS (
			DELETE FROM matches WHERE id IN (SELECT id FROM m) RETURNING id
		)
		SELECT (SELECT id FROM deleted), (SELECT id FROM conv)
	`, blockerID, blockedID).Scan(&matchID, &conversationID); err != nil {
		return nil, err
	}
	if matchID == nil {
		return nil, nil
	}
	removed := &RemovedMatch{MatchID: *matchID}
	if conversationID != nil {
		removed.ConversationID = *conversationID
	}
	return removed, nil
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*RemovedMatch, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	removed, err := blockTx(ctx, tx, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

// CreateReport stores the report, blocks the reported member for the
// reporter, and flags the account for review once enough distinct members
// reported it recently.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, threshold int, window time.Duration) (*RemovedMatch, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
	`, reporterID, reportedID, reason, details); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrRepositoryUserNotFound
		}
		return nil, err
	}
	removed, err := blockTx(ctx, tx, reporterID, reportedID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET moderation_status = 'under_review', updated_at = NOW()
		WHERE id = $1 AND moderation_status = 'active'
		  AND (
			SELECT COUNT(DISTINCT reporter_id) FROM reports
			WHERE reported_id = $1 AND status = 'open' AND created_at > NOW() - make_interval(secs => $2)
		  ) >= $3
	`, reportedID, window.Seconds(), threshold); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}
