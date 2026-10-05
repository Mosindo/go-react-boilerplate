package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type blockRow struct {
	UserID    string
	BlockedAt time.Time
}

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (matchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]blockRow, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Block is idempotent. It returns the id of the match between the two users (empty if none) so the caller
// can tell the other side's client to drop the conversation. Likes and match rows are kept: blocked pairs are
// filtered out of discovery, matches, chat and profile views by their queries.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Same pair lock as matching.Swipe: a block and a reciprocal like cannot interleave into a match.
	low, high := blockerID, blockedID
	if low > high {
		low, high = high, low
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, low+high); err != nil {
		return "", err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
		return "", err
	}
	if !exists {
		return "", ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", err
	}
	var matchID *string
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT id::text FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid))
	`, blockerID, blockedID).Scan(&matchID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if matchID == nil {
		return "", nil
	}
	return *matchID, nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]blockRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT blocked_id, created_at FROM blocks WHERE blocker_id = $1
		ORDER BY created_at DESC, blocked_id LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []blockRow{}
	for rows.Next() {
		var b blockRow
		if err := rows.Scan(&b.UserID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateReport stores a report unless the same reporter already sent an identical one in the last 24 hours.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, reportedID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrUserNotFound
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
		    SELECT 1 FROM reports
		    WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3 AND created_at > NOW() - INTERVAL '24 hours')
	`, reporterID, reportedID, reason, details)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}
