package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notifications: not found")

// Querier is implemented by *pgxpool.Pool and pgx.Tx, so Create can join the
// caller's transaction (a match and its notifications commit atomically).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Create inserts a notification (or refreshes the unread one sharing its dedupe key).
func Create(ctx context.Context, q Querier, n NewNotification) (Notification, error) {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return Notification{}, err
	}
	var out Notification
	var raw []byte
	var dedupe any
	if n.DedupeKey != "" {
		dedupe = n.DedupeKey
	}
	err = q.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, dedupe_key)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT (user_id, dedupe_key) WHERE is_read = FALSE AND dedupe_key IS NOT NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, data = EXCLUDED.data, created_at = NOW()
		RETURNING id::text, type, title, body, data, is_read, read_at, created_at
	`, n.UserID, n.Type, n.Title, n.Body, string(data), dedupe).
		Scan(&out.ID, &out.Type, &out.Title, &out.Body, &raw, &out.IsRead, &out.ReadAt, &out.CreatedAt)
	if err != nil {
		return Notification{}, err
	}
	if err := json.Unmarshal(raw, &out.Data); err != nil || out.Data == nil {
		out.Data = map[string]string{}
	}
	return out, nil
}

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, type, title, body, data, is_read, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if err := json.Unmarshal(raw, &n.Data); err != nil || n.Data == nil {
			n.Data = map[string]string{}
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var unread int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&unread); err != nil {
		return nil, 0, err
	}
	return out, unread, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		var invalid interface{ SQLState() string }
		if errors.As(err, &invalid) && invalid.SQLState() == "22P02" {
			return ErrNotFound
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE`, userID)
	return err
}
