package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryUserNotFound = errors.New("user not found")

type blockedRow struct {
	BlockedUser
	photoID *string
}

type Repository interface {
	// Block records the block and ends every active match between the two
	// users; it returns the ended match ids.
	Block(ctx context.Context, blocker, blocked string) ([]string, error)
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocks(ctx context.Context, blocker string) ([]blockedRow, error)
	CreateReport(ctx context.Context, reporter, reported, reason, details string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blocker, blocked); err != nil {
		return nil, mapErr(err)
	}

	rows, err := tx.Query(ctx, `
		UPDATE matches SET ended_at = NOW()
		WHERE ended_at IS NULL AND user_a = least($1::uuid, $2::uuid) AND user_b = greatest($1::uuid, $2::uuid)
		RETURNING id
	`, blocker, blocked)
	if err != nil {
		return nil, err
	}
	ended := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ended = append(ended, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ended) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE match_id = ANY($1::uuid[])`, ended); err != nil {
			return nil, err
		}
	}
	return ended, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return mapErr(err)
}

func (r *PGRepository) ListBlocks(ctx context.Context, blocker string) ([]blockedRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at, ph.id
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		LEFT JOIN LATERAL (SELECT id FROM photos WHERE user_id = b.blocked_id ORDER BY position LIMIT 1) ph ON TRUE
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []blockedRow{}
	for rows.Next() {
		var br blockedRow
		var at time.Time
		if err := rows.Scan(&br.UserID, &br.FirstName, &at, &br.photoID); err != nil {
			return nil, err
		}
		br.BlockedAt = at
		out = append(out, br)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
	`, reporter, reported, reason, details)
	return mapErr(err)
}

func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		return ErrRepositoryUserNotFound
	}
	return err
}
