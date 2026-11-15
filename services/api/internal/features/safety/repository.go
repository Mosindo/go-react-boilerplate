package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (ReportResponse, string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (string, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isForeignKeyViolation(err) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	// Blocking ends the match: conversation and messages cascade.
	var matchID string
	err := tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		RETURNING id
	`, blockerID, blockedID).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return matchID, nil
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	matchID, err := blockTx(ctx, tx, blockerID, blockedID)
	if err != nil {
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
		return ErrBlockNotFound
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2
	`, blockerID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

// Report stores the report and blocks the reported user in one transaction,
// so the reporter is protected immediately.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (ReportResponse, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ReportResponse{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var resp ReportResponse
	err = tx.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, reporterID, reportedID, reason, details).Scan(&resp.ID, &resp.CreatedAt)
	if err != nil {
		if isForeignKeyViolation(err) {
			return ReportResponse{}, "", ErrUserNotFound
		}
		return ReportResponse{}, "", err
	}
	matchID, err := blockTx(ctx, tx, reporterID, reportedID)
	if err != nil {
		return ReportResponse{}, "", err
	}
	return resp, matchID, tx.Commit(ctx)
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
