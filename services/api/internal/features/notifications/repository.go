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
	Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) (int, error)
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
		SELECT `+columns+`
		FROM notifications
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
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM (
			SELECT 1 FROM notifications WHERE user_id = $1 AND is_read = FALSE LIMIT 1000
		) capped
	`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error) {
	if data == nil {
		data = map[string]string{}
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, data))
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

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
