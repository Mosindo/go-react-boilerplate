package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	// Block records the block and ends any active match with that user. It
	// returns true when an active match was ended.
	Block(ctx context.Context, blockerID, blockedID string) (matchEnded bool, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	// CreateReport stores a report unless an identical one from the same
	// reporter was filed in the last 24 hours (idempotent).
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isFKViolation(err) {
			return false, ErrUserNotFound
		}
		return false, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL
	`, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
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

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1::uuid, $2::uuid, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM reports
			WHERE reporter_id = $1::uuid AND reported_id = $2::uuid AND reason = $3
			  AND created_at > NOW() - INTERVAL '24 hours'
		)
	`, reporterID, reportedID, reason, details)
	if err != nil && isFKViolation(err) {
		return ErrUserNotFound
	}
	return err
}
