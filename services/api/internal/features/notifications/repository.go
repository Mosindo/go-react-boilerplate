package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error)
	RefreshUnread(ctx context.Context, userID, kind, dedupeField, dedupeValue, title, body string, data map[string]string) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id, user_id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	if n.Data == nil {
		n.Data = map[string]string{}
	}
	return n, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error) {
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, data))
}

// RefreshUnread updates an existing unread notification of the same kind and
// dedupe key (e.g. one per conversation) instead of stacking duplicates.
// The bool reports whether a new row was inserted.
func (r *PGRepository) RefreshUnread(ctx context.Context, userID, kind, dedupeField, dedupeValue, title, body string, data map[string]string) (Notification, bool, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications
		SET title = $5, body = $6, data = $7, created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE user_id = $1 AND type = $2 AND is_read = FALSE AND data->>$3 = $4
			ORDER BY created_at DESC LIMIT 1
		)
		RETURNING `+columns, userID, kind, dedupeField, dedupeValue, title, body, data))
	if err == nil {
		return n, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, err
	}
	n, err = r.Create(ctx, userID, kind, title, body, data)
	return n, true, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, notificationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}
