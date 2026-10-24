package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	// Block records the block and closes any active match; it returns the closed conversation id, if any.
	Block(ctx context.Context, blockerID, blockedID string) (conversationID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		if isFKViolation(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	var convID string
	err = tx.QueryRow(ctx, `
		WITH m AS (
			UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
			WHERE unmatched_at IS NULL
			  AND user_low = LEAST($1::uuid, $2::uuid) AND user_high = GREATEST($1::uuid, $2::uuid)
			RETURNING id
		)
		SELECT c.id FROM m JOIN conversations c ON c.match_id = m.id
	`, blockerID, blockedID).Scan(&convID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return convID, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
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
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateReport is idempotent per (reporter, reported) while a report is still open.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}
