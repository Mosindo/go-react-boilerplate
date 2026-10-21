package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTargetNotFound = errors.New("safety: target not found")

type storedBlock struct {
	ID, Name  string
	PhotoID   *string
	BlockedAt time.Time
}

type Repository interface {
	// Block records the block and dissolves any match; it returns the removed conversation ids.
	Block(ctx context.Context, blocker, blocked string) ([]string, error)
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocks(ctx context.Context, blocker string) ([]storedBlock, error)
	CreateReport(ctx context.Context, reporter, reported, reason, details string) (Report, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blocked).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTargetNotFound
	}
	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		WITH gone AS (
			DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) RETURNING id
		)
		SELECT c.id::text FROM conversations c JOIN gone ON gone.id = c.match_id
	`, blocker, blocked)
	if err != nil {
		return nil, err
	}
	var convs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		convs = append(convs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Drop pending notifications between the pair so nothing links them anymore.
	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE (user_id = $1 AND actor_id = $2) OR (user_id = $2 AND actor_id = $1)
	`, blocker, blocked); err != nil {
		return nil, err
	}
	return convs, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blocker string) ([]storedBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id::text, COALESCE(p.first_name, ''), (SELECT id FROM photos WHERE user_id = b.blocked_id AND position = 0), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
	`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedBlock{}
	for rows.Next() {
		var b storedBlock
		if err := rows.Scan(&b.ID, &b.Name, &b.PhotoID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporter, reported, reason, details string) (Report, error) {
	var rep Report
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2
		RETURNING id, created_at
	`, reporter, reported, reason, details).Scan(&rep.ID, &rep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrTargetNotFound
	}
	return rep, err
}
