package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	Create(ctx context.Context, userID, kind, title, body string, data []byte, dedupeKey string) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkReadByConversation(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const notificationColumns = `id, type, title, body, data, is_read, created_at, read_at`

func scanNotification(row pgx.Row) (Notification, error) {
	var n Notification
	var data []byte
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	n.Data = data
	return n, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+notificationColumns+`
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Notification, 0, limit)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&count)
	return count, err
}

// Create inserts a notification. When dedupeKey is set and an unread
// notification with the same key exists, nothing is inserted (created=false):
// a burst of messages yields a single "new message" notification.
func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data []byte, dedupeKey string) (Notification, bool, error) {
	row := r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5
		WHERE $6 = '' OR NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND is_read = FALSE AND data->>'dedupeKey' = $6
		)
		RETURNING `+notificationColumns, userID, kind, title, body, data, dedupeKey)
	n, err := scanNotification(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, false, nil
		}
		return Notification{}, false, err
	}
	return n, true, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scanNotification(r.dbPool.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+notificationColumns, notificationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrRepositoryNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *PGRepository) MarkReadByConversation(ctx context.Context, userID, conversationID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE AND data->>'conversationId' = $2
	`, userID, conversationID)
	return err
}
