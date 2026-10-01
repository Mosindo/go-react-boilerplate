package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	// CreateUnique inserts unless an unread notification with the same type and data exists.
	CreateUnique(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) (int, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const columns = `id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var data []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &data, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
		return Notification{}, err
	}
	n.Data = map[string]string{}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &n.Data)
	}
	return n, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Notification, 0, limit)
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) CreateUnique(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, bool, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, false, err
	}
	n, err := scan(r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5::jsonb
		WHERE NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data = $5::jsonb)
		RETURNING `+columns, userID, kind, title, body, string(payload)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	if err != nil {
		return Notification{}, false, err
	}
	return n, true, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scan(r.pool.QueryRow(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2 RETURNING `+columns, notificationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) (int, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE`, userID)
	return int(tag.RowsAffected()), err
}
