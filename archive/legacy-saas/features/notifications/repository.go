package notifications

import (
	"context"
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
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type Repository interface {
	ListNotifications(ctx context.Context, userID string, limit, offset int) ([]StoredNotification, error)
	CreateNotification(ctx context.Context, userID, kind, title, body string) (StoredNotification, error)
	MarkRead(ctx context.Context, userID, notificationID string) (StoredNotification, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListNotifications(ctx context.Context, userID string, limit, offset int) ([]StoredNotification, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, user_id, type, title, body, is_read, created_at, read_at
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
		var notification StoredNotification
		var readAt *time.Time
		if err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Title,
			&notification.Body,
			&notification.IsRead,
			&notification.CreatedAt,
			&readAt,
		); err != nil {
			return nil, err
		}
		notification.ReadAt = readAt
		notifications = append(notifications, notification)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return notifications, nil
}

func (r *PGRepository) CreateNotification(ctx context.Context, userID, kind, title, body string) (StoredNotification, error) {
	var notification StoredNotification
	var readAt *time.Time
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, type, title, body, is_read, created_at, read_at
	`, userID, kind, title, body).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.Type,
		&notification.Title,
		&notification.Body,
		&notification.IsRead,
		&notification.CreatedAt,
		&readAt,
	)
	if err != nil {
		return StoredNotification{}, err
	}
	notification.ReadAt = readAt
	return notification, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (StoredNotification, error) {
	var notification StoredNotification
	var readAt *time.Time
	err := r.dbPool.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE,
		    read_at = COALESCE(read_at, NOW())
		WHERE id = $1
		  AND user_id = $2
		RETURNING id, user_id, type, title, body, is_read, created_at, read_at
	`, notificationID, userID).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.Type,
		&notification.Title,
		&notification.Body,
		&notification.IsRead,
		&notification.CreatedAt,
		&readAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredNotification{}, ErrNotificationNotFound
		}
		return StoredNotification{}, err
	}
	notification.ReadAt = readAt
	return notification, nil
}
