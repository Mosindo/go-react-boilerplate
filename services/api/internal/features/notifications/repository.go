package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error)
	Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	UnreadCount(ctx context.Context, userID string) (int, error)
	CreateCollapsed(ctx context.Context, userID, kind, title, body string, data map[string]string, collapseKey string) (Notification, bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const columns = `id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	if n.Data == nil {
		n.Data = map[string]string{}
	}
	return n, err
}

// List uses keyset pagination on (created_at, id) for stable pages.
func (r *PGRepository) List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+columns+`
		FROM notifications n
		WHERE n.user_id = $1
		  AND ($2 = '' OR (n.created_at, n.id) < (
		        SELECT created_at, id FROM notifications WHERE id::text = $2 AND user_id = $1))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $3
	`, userID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error) {
	if data == nil {
		data = map[string]string{}
	}
	return scan(r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, data))
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scan(r.dbPool.QueryRow(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, notificationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

// CreateCollapsed inserts a notification unless an unread one with the same type and
// collapse key already exists (prevents one notification per chat message).
func (r *PGRepository) CreateCollapsed(ctx context.Context, userID, kind, title, body string, data map[string]string, collapseKey string) (Notification, bool, error) {
	if data == nil {
		data = map[string]string{}
	}
	data["collapseKey"] = collapseKey
	n, err := scan(r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data->>'collapseKey' = $6
		)
		RETURNING `+columns, userID, kind, title, body, data, collapseKey))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	return n, err == nil, err
}
