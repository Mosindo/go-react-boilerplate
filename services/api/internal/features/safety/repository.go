package safety

import (
	"context"
	"errors"

	"example.com/api/internal/features/discovery"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Block records the block and dissolves any match (and its conversation) between the two users.
func (r *Repository) Block(ctx context.Context, blocker, blocked string) (*discovery.RemovedMatch, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		if isFK(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rm, err := discovery.DeleteMatchBetween(ctx, tx, blocker, blocked)
	if err != nil {
		return nil, err
	}
	return rm, tx.Commit(ctx)
}

func (r *Repository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *Repository) List(ctx context.Context, blocker string) ([]Block, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Block{}
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) Report(ctx context.Context, reporter, reported, reason, details string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_id, reason) WHERE status = 'open' DO UPDATE SET details = EXCLUDED.details
		RETURNING id`, reporter, reported, reason, details).Scan(&id)
	if isFK(err) {
		return "", ErrNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return id, err
}
