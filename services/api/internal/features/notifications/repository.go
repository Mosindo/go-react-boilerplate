package notifications

import (
	"context"
	"encoding/json"
	"time"

	"example.com/api/internal/platform/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// CreateTx inserts a notification inside the caller's transaction so it commits atomically with the
// event that caused it (e.g. a new match).
func CreateTx(ctx context.Context, tx pgx.Tx, n New) (Notification, error) {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return Notification{}, err
	}
	out := Notification{UserID: n.UserID, Type: n.Type, Title: n.Title, Body: n.Body}
	var raw []byte
	err = tx.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, data, read_at, created_at
	`, n.UserID, n.Type, n.Title, n.Body, data).Scan(&out.ID, &raw, &out.ReadAt, &out.CreatedAt)
	out.Data = raw
	return out, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]Notification, error) {
	var before time.Time
	var beforeID string
	if cursor != nil {
		before, beforeID = cursor.Time, cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, body, data, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, userID, nullTime(before), nullStr(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.Data = raw
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
