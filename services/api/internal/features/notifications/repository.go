package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("notification not found")

type Repository interface {
	Upsert(ctx context.Context, userID, kind, title, body string, data json.RawMessage, dedupeKey *string) (Notification, error)
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkReadByDedupeKey(ctx context.Context, userID, dedupeKey string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const columns = `id, user_id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	return n, err
}

// Upsert inserts a notification, or refreshes the existing unread one that
// shares the same dedupe key so that a burst of messages yields one entry.
func (r *PGRepository) Upsert(ctx context.Context, userID, kind, title, body string, data json.RawMessage, dedupeKey *string) (Notification, error) {
	return scan(r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, dedupe_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, dedupe_key) WHERE NOT is_read AND dedupe_key IS NOT NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, data = EXCLUDED.data, created_at = NOW()
		RETURNING `+columns, userID, kind, title, body, data, dedupeKey))
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND NOT is_read`, userID).Scan(&count)
	return count, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	var exists bool
	err := r.dbPool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
			WHERE id = $1 AND user_id = $2
			RETURNING 1
		)
		SELECT EXISTS (SELECT 1 FROM updated)
	`, notificationID, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND NOT is_read
	`, userID)
	return err
}

func (r *PGRepository) MarkReadByDedupeKey(ctx context.Context, userID, dedupeKey string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND dedupe_key = $2 AND NOT is_read
	`, userID, dedupeKey)
	return err
}
