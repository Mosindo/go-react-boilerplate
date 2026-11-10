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
	Create(ctx context.Context, userID, kind string, actorID, matchID *string, body string) (Notification, error)
	UpsertMessage(ctx context.Context, userID, actorID, matchID, body string) (Notification, error)
	List(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkMatchRead(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

const columns = `id, type, actor_id, match_id, body, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.ActorID, &n.MatchID, &n.Body, &n.CreatedAt, &n.ReadAt)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind string, actorID, matchID *string, body string) (Notification, error) {
	return scan(r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, actorID, matchID, body))
}

// UpsertMessage keeps a single unread "message" notification per conversation.
func (r *PGRepository) UpsertMessage(ctx context.Context, userID, actorID, matchID, body string) (Notification, error) {
	return scan(r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id, body)
		VALUES ($1, 'message', $2, $3, $4)
		ON CONFLICT (user_id, match_id) WHERE type = 'message' AND read_at IS NULL
		DO UPDATE SET body = EXCLUDED.body, created_at = NOW()
		RETURNING `+columns, userID, actorID, matchID, body))
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, userID, before, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkMatchRead(ctx context.Context, userID, matchID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND match_id = $2 AND read_at IS NULL
	`, userID, matchID)
	return err
}

func nullUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
