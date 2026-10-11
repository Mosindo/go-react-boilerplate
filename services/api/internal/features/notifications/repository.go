package notifications

import (
	"context"
	"encoding/json"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Insert stores a notification using any Querier, so callers can do it inside their own transaction.
func Insert(ctx context.Context, q db.Querier, n NewNotification) (Notification, error) {
	data := n.Data
	if data == nil {
		data = map[string]string{}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	out := Notification{Type: n.Type, Title: n.Title, Body: n.Body, Data: data}
	err = q.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`, n.UserID, n.Type, n.Title, n.Body, payload).Scan(&out.ID, &out.CreatedAt)
	return out, err
}

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

func (r *Repository) List(ctx context.Context, userID string, cur *Cursor, limit int) ([]Notification, error) {
	var before time.Time
	var beforeID string
	hasCursor := cur != nil
	if hasCursor {
		before, beforeID = cur.CreatedAt, cur.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, title, body, data, read_at IS NOT NULL, created_at
		FROM notifications
		WHERE user_id = $1 AND (NOT $2::bool OR (created_at, id) < ($3::timestamptz, $4::uuid))
		ORDER BY created_at DESC, id DESC LIMIT $5`, userID, hasCursor, before, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &n.Data); err != nil || n.Data == nil {
			n.Data = map[string]string{}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func nullUUID(id string) any {
	if id == "" {
		return "00000000-0000-0000-0000-000000000000"
	}
	return id
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	return tag.RowsAffected() > 0, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
