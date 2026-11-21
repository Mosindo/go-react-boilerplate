package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type StoredNotification struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      map[string]string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type Repository interface {
	ListNotifications(ctx context.Context, userID string, limit, offset int) ([]StoredNotification, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]string) (StoredNotification, error)
	UpsertUnreadByRef(ctx context.Context, userID, kind, refKey, refValue, title, body string, data map[string]string) (StoredNotification, error)
	MarkRead(ctx context.Context, userID, notificationID string) (StoredNotification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkReadByRef(ctx context.Context, userID, kind, refKey, refValue string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const notificationColumns = `id, user_id, type, title, body, data, is_read, created_at, read_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanNotification(row scanner) (StoredNotification, error) {
	var n StoredNotification
	var raw []byte
	if err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
		return StoredNotification{}, err
	}
	n.Data = map[string]string{}
	if len(raw) > 0 {
		var generic map[string]any
		if err := json.Unmarshal(raw, &generic); err == nil {
			for k, v := range generic {
				if s, ok := v.(string); ok {
					n.Data[k] = s
				}
			}
		}
	}
	return n, nil
}

func encodeData(data map[string]string) []byte {
	if data == nil {
		data = map[string]string{}
	}
	raw, _ := json.Marshal(data)
	return raw
}

func (r *PGRepository) ListNotifications(ctx context.Context, userID string, limit, offset int) ([]StoredNotification, error) {
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

	notifications := make([]StoredNotification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *PGRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM notifications WHERE user_id = $1 AND is_read = FALSE LIMIT 100
		) unread
	`, userID).Scan(&count)
	return count, err
}

func (r *PGRepository) CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]string) (StoredNotification, error) {
	return scanNotification(r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+notificationColumns, userID, kind, title, body, encodeData(data)))
}

// UpsertUnreadByRef coalesces notifications: if an unread notification of the
// same type and reference exists (e.g. unread messages from the same person)
// it is refreshed instead of creating a new row.
func (r *PGRepository) UpsertUnreadByRef(ctx context.Context, userID, kind, refKey, refValue, title, body string, data map[string]string) (StoredNotification, error) {
	n, err := scanNotification(r.dbPool.QueryRow(ctx, `
		UPDATE notifications
		SET title = $5, body = $6, data = $7, created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data->>$3 = $4
			ORDER BY created_at DESC
			LIMIT 1
		)
		RETURNING `+notificationColumns, userID, kind, refKey, refValue, title, body, encodeData(data)))
	if err == nil {
		return n, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return StoredNotification{}, err
	}
	return r.CreateNotification(ctx, userID, kind, title, body, data)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (StoredNotification, error) {
	n, err := scanNotification(r.dbPool.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE,
		    read_at = COALESCE(read_at, NOW())
		WHERE id = $1
		  AND user_id = $2
		RETURNING `+notificationColumns, notificationID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredNotification{}, ErrNotificationNotFound
		}
		return StoredNotification{}, err
	}
	return n, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *PGRepository) MarkReadByRef(ctx context.Context, userID, kind, refKey, refValue string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data->>$3 = $4
	`, userID, kind, refKey, refValue)
	return err
}
