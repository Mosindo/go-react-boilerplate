package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoUnknownUser = errors.New("unknown user")

type Repository interface {
	// Block records the block, then removes the match (messages and notifications cascade) and the
	// swipes between the pair. It returns the id of the removed match, if any.
	Block(ctx context.Context, blocker, blocked string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocked(ctx context.Context, blocker string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporter, reported, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		if isFK(err) {
			return "", ErrRepoUnknownUser
		}
		return "", err
	}
	var matchID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches WHERE (user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)) RETURNING id::text`,
		blocker, blocked).Scan(&matchID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM swipes WHERE (swiper_id = $1 AND target_id = $2) OR (swiper_id = $2 AND target_id = $1)`, blocker, blocked); err != nil {
		return "", err
	}
	return matchID, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blocker string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id::text, COALESCE(p.first_name, ''), b.created_at FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, blocker)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var u BlockedUser
		err := row.Scan(&u.ID, &u.FirstName, &u.BlockedAt)
		return u, err
	})
}

func (r *PGRepository) CreateReport(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)`, reporter, reported, reason, details)
	if isFK(err) {
		return ErrRepoUnknownUser
	}
	return err
}
