package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	// Create inserts a notification unless an unread one with the same type and data exists.
	Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (*Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, title, body, data, is_read, created_at, read_at
		FROM notifications
		WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		n.Data = map[string]string{}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &n.Data)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*)::int FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (*Notification, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	n := Notification{Type: kind, Title: title, Body: body, Data: data}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5::jsonb
		WHERE NOT EXISTS (
		  SELECT 1 FROM notifications
		  WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data = $5::jsonb)
		RETURNING id, is_read, created_at
	`, userID, kind, title, body, string(raw)).Scan(&n.ID, &n.IsRead, &n.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}
