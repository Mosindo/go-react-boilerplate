package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	Create(ctx context.Context, userID, kind, title, body string, data []byte, collapseKey string) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.CollectableRow) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	return n, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+`
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scan)
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&count)
	return count, err
}

// Create inserts a notification. When collapseKey is set and an unread
// notification with the same key exists, nothing is inserted (created=false):
// a burst of messages yields a single "new message" notification.
func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data []byte, collapseKey string) (Notification, bool, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5::jsonb
		WHERE $6 = '' OR NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND is_read = FALSE AND type = $2 AND data->>'collapseKey' = $6
		)
		RETURNING `+columns, userID, kind, title, body, data, collapseKey)
	if err != nil {
		return Notification{}, false, err
	}
	n, err := pgx.CollectExactlyOneRow(rows, scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	if err != nil {
		return Notification{}, false, err
	}
	return n, true, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, notificationID, userID)
	if err != nil {
		return Notification{}, err
	}
	n, err := pgx.CollectExactlyOneRow(rows, scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE AND type = 'message' AND data->>'conversationId' = $2
	`, userID, conversationID)
	return err
}
