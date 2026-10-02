package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Row, error)
	Summary(ctx context.Context, userID string) (Summary, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Row, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, type, match_id::text, actor_id::text, created_at, read_at FROM notifications
		WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Row, error) {
		var n Row
		err := row.Scan(&n.ID, &n.Type, &n.MatchID, &n.ActorID, &n.CreatedAt, &n.ReadAt)
		return n, err
	})
}

func (r *PGRepository) Summary(ctx context.Context, userID string) (Summary, error) {
	var s Summary
	err := r.db.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL)::int,
		       (SELECT COUNT(*) FROM messages x JOIN matches m ON m.id = x.match_id
		         WHERE (m.user_a = $1 OR m.user_b = $1) AND x.sender_id <> $1 AND x.read_at IS NULL)::int`, userID).
		Scan(&s.UnreadNotifications, &s.UnreadMessages)
	return s, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
