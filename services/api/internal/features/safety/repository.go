package safety

import (
	"context"
	"errors"

	"example.com/api/internal/features/matching"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("not found")

type Repository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
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

// Block records the block and removes any match between the two members
// (which cascades to the conversation and its messages).
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Same per-pair lock as swipes: a like racing with this block either
	// commits first (and its match is deleted below) or sees the block.
	if err := matching.LockPair(ctx, tx, blockerID, blockedID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
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
	tag, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
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

// CreateReport is idempotent while a report from the same author is open.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_user_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_user_id) WHERE status = 'open'
		DO UPDATE SET details = CASE WHEN EXCLUDED.details = '' THEN reports.details ELSE EXCLUDED.details END,
		              reason = EXCLUDED.reason
		RETURNING id
	`, reporterID, reportedID, reason, details).Scan(&id)
	return id, err
}
