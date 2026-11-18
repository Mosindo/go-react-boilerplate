package safety

import (
	"context"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (hadMatch bool, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func targetExists(ctx context.Context, tx pgx.Tx, actorID, targetID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users t JOIN users a ON a.id = $1
			WHERE t.id = $2 AND t.organization_id = a.organization_id
		)
	`, actorID, targetID).Scan(&exists)
	return exists, err
}

// blockTx inserts the block and tears down any match and conversation.
func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (bool, error) {
	if err := db.LockPair(ctx, tx, blockerID, blockedID); err != nil {
		return false, err
	}
	exists, err := targetExists(ctx, tx, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, ErrRepositoryNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_user_id, blocked_user_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return false, err
	}
	var conversationID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
		RETURNING conversation_id
	`, blockerID, blockedID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, conversationID); err != nil {
		return false, err
	}
	return true, nil
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tx, err := r.dbPool.Begin(ctx)
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
	tag, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]storedBlock, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT b.blocked_user_id, p.first_name,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = b.blocked_user_id ORDER BY ph.position LIMIT 1),
		       b.created_at
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
	out := make([]storedBlock, 0)
	for rows.Next() {
		var b storedBlock
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.PhotoID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Report files a report, blocks the member and hides their profile from
// discovery once enough distinct members reported them.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := blockTx(ctx, tx, reporterID, reportedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_user_id, reported_user_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_user_id, reported_user_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE profiles SET moderation_hidden = TRUE, updated_at = NOW()
		WHERE user_id = $1
		  AND NOT moderation_hidden
		  AND (
		    SELECT COUNT(DISTINCT r.reporter_user_id)
		    FROM reports r
		    JOIN users reporter ON reporter.id = r.reporter_user_id
		    WHERE r.reported_user_id = $1
		      AND r.status = 'open'
		      AND reporter.created_at < NOW() - $3::interval
		  ) >= $2
	`, reportedID, ModerationThreshold, ModerationMinAccountAge); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
