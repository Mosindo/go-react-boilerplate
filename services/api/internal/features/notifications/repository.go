package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `id, type, title, body, data, is_read, created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	n.Data = map[string]string{}
	_ = json.Unmarshal(raw, &n.Data)
	return n, nil
}

func (r *Repository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+columns+` FROM notifications WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*)::int FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *Repository) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	return scan(r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, payload))
}

// UpsertMessage coalesces message notifications per conversation: while one is still unread,
// further messages refresh it instead of piling up a new row per message.
func (r *Repository) UpsertMessage(ctx context.Context, userID, conversationID, title, body string, data map[string]string) (Notification, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	n, err := scan(r.pool.QueryRow(ctx, `
		UPDATE notifications SET title = $3, body = $4, data = $5, created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE user_id = $1 AND type = 'message' AND is_read = FALSE AND data->>'conversationId' = $2
			ORDER BY created_at DESC LIMIT 1)
		RETURNING `+columns, userID, conversationID, title, body, payload))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.Create(ctx, userID, "message", title, body, data)
	}
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	n, err := scan(r.pool.QueryRow(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2 RETURNING `+columns, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	return n, err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE`, userID)
	return err
}
