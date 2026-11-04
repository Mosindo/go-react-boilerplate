package safety

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type blockedRow struct {
	UserID    string
	FirstName string
	PhotoID   *string
	BlockedAt time.Time
}

type Repository interface {
	Block(ctx context.Context, blocker, blocked string) error
	Unblock(ctx context.Context, blocker, blocked string) error
	ListBlocked(ctx context.Context, blocker string) ([]blockedRow, error)
	CreateReport(ctx context.Context, reporter, reported, reason, details string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func (r *PGRepository) Block(ctx context.Context, blocker, blocked string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blocker, blocked)
	if isFKViolation(err) {
		return ErrUserNotFound
	}
	return err
}

func (r *PGRepository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blocker string) ([]blockedRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), ph.id, b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		LEFT JOIN LATERAL (SELECT id FROM photos WHERE user_id = b.blocked_id ORDER BY position LIMIT 1) ph ON TRUE
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT 200`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []blockedRow{}
	for rows.Next() {
		var b blockedRow
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.PhotoID, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateReport is idempotent while a previous report of the same pair is still open.
func (r *PGRepository) CreateReport(ctx context.Context, reporter, reported, reason, details string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open')`,
		reporter, reported, reason, details)
	if isFKViolation(err) {
		return ErrUserNotFound
	}
	return err
}
