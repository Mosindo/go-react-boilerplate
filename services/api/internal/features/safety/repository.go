package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoUserNotFound = errors.New("user not found")
	errRepoDuplicate    = errors.New("duplicate report")
)

type Repository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) (bool, error)
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (bool, error)
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

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return errRepoUserNotFound
	}
	return err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) (bool, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
	`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

// Report stores a report and escalates the reported account to review when enough
// distinct members reported it recently. Returns whether the account was escalated.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (bool, error) {
	var escalated bool
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
		`, reporterID, reportedID, reason, details)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return errRepoDuplicate
			case "23503":
				return errRepoUserNotFound
			}
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE users SET moderation_status = 'under_review', updated_at = NOW()
			WHERE id = $1 AND moderation_status = 'active'
			  AND (SELECT COUNT(DISTINCT reporter_id) FROM reports
			       WHERE reported_id = $1 AND created_at > NOW() - INTERVAL '30 days') >= $2
		`, reportedID, ReviewThreshold)
		if err != nil {
			return err
		}
		escalated = tag.RowsAffected() > 0
		return nil
	})
	return escalated, err
}
