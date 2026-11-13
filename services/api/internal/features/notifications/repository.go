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
	Create(ctx context.Context, n Notification, dedupeKey string) (Notification, bool, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkPeerMessagesRead(ctx context.Context, userID, peerUserID string) error
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
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND NOT is_read`, userID).Scan(&count)
	return count, err
}

// Create inserts a notification. With a dedupe key, nothing is inserted when
// an unread notification of the same type and key already exists (e.g. one
// "new messages" notification per conversation until it is read).
func (r *PGRepository) Create(ctx context.Context, n Notification, dedupeKey string) (Notification, bool, error) {
	data := n.Data
	if data == nil {
		data = map[string]string{}
	}
	if dedupeKey != "" {
		data["dedupe"] = dedupeKey
	}
	created, err := scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5
		WHERE $6 = '' OR NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2 AND NOT is_read AND data->>'dedupe' = $6
		)
		RETURNING `+columns, n.UserID, n.Type, n.Title, n.Body, data, dedupeKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, false, nil
		}
		return Notification{}, false, err
	}
	return created, true, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, notificationID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotificationNotFound
		}
		return Notification{}, err
	}
	return n, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND NOT is_read
	`, userID)
	return err
}

// MarkPeerMessagesRead clears message notifications for a conversation once
// the user has opened it.
func (r *PGRepository) MarkPeerMessagesRead(ctx context.Context, userID, peerUserID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'peerUserId' = $2
	`, userID, peerUserID)
	return err
}
