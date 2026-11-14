package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func block(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (string, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isForeignKeyViolation(err) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	var matchID string
	err := tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_a_id = LEAST($1::UUID, $2::UUID) AND user_b_id = GREATEST($1::UUID, $2::UUID)
		RETURNING id
	`, blockerID, blockedID).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return matchID, nil
}

// Block records the block and removes any match (and its conversation).
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	matchID, err := block(ctx, tx, blockerID, blockedID)
	if err != nil {
		return "", err
	}
	return matchID, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
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
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

// Report stores the report, blocks the reported member for the reporter and
// suspends the reported account from discovery past the threshold.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_id) WHERE status = 'open' DO NOTHING
	`, reporterID, reportedID, reason, details); err != nil {
		if isForeignKeyViolation(err) {
			return ErrUserNotFound
		}
		return err
	}
	if _, err := block(ctx, tx, reporterID, reportedID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET suspended_at = NOW()
		WHERE id = $1 AND suspended_at IS NULL
		  AND (SELECT COUNT(DISTINCT reporter_id) FROM reports WHERE reported_id = $1 AND status = 'open') >= $2
	`, reportedID, threshold); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
