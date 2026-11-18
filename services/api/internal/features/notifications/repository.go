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
	Create(ctx context.Context, n Notification, coalesce bool) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const notificationColumns = `id, user_id, type, title, body, data, is_read, created_at, read_at`

func scanNotification(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
		return Notification{}, err
	}
	n.Data = map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &n.Data)
	}
	return n, nil
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
	out := make([]Notification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&count)
	return count, err
}

// Create inserts a notification. With coalesce, nothing is inserted when an
// unread notification of the same type about the same member already exists,
// so a burst of messages produces a single notification.
func (r *PGRepository) Create(ctx context.Context, n Notification, coalesce bool) (Notification, bool, error) {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return Notification{}, false, err
	}
	row := r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5::jsonb
		WHERE NOT $6::boolean OR NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1
			  AND type = $2
			  AND is_read = FALSE
			  AND data->>'userId' = ($5::jsonb)->>'userId'
		)
		RETURNING `+notificationColumns, n.UserID, n.Type, n.Title, n.Body, string(data), coalesce)
	created, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	if err != nil {
		return Notification{}, false, err
	}
	return created, true, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scanNotification(r.dbPool.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+notificationColumns, notificationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
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
