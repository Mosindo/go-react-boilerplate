package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	Create(ctx context.Context, userID, kind, title, body string, data []byte) (Notification, error)
	UpsertMessage(ctx context.Context, userID, matchID, title, body string, data []byte) (Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
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
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND NOT is_read`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data []byte) (Notification, error) {
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, data))
}

// UpsertMessage keeps a single unread "message" notification per
// conversation instead of one per message.
func (r *PGRepository) UpsertMessage(ctx context.Context, userID, matchID, title, body string, data []byte) (Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications
		SET title = $3, body = $4, data = $5, created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'matchId' = $2
			ORDER BY created_at DESC LIMIT 1
		)
		RETURNING `+columns, userID, matchID, title, body, data))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.Create(ctx, userID, TypeMessage, title, body, data)
	}
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND NOT is_read
	`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND NOT is_read AND data->>'matchId' = $2
	`, userID, matchID)
	return err
}
