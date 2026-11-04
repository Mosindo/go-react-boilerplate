package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, kind, title, body string, data []byte) (Notification, error)
	// CreateUnique skips creation when an unread notification of the same kind
	// already exists for data[key] (coalesces chat message notifications).
	CreateUnique(ctx context.Context, userID, kind, title, body string, data []byte, key, value string) (*Notification, error)
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const cols = `id, type, title, body, data, read_at IS NOT NULL, created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data []byte) (Notification, error) {
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING `+cols, userID, kind, title, body, data))
}

func (r *PGRepository) CreateUnique(ctx context.Context, userID, kind, title, body string, data []byte, key, value string) (*Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5
		WHERE NOT EXISTS (
		  SELECT 1 FROM notifications WHERE user_id = $1 AND type = $2 AND read_at IS NULL AND data ->> $6 = $7)
		RETURNING `+cols, userID, kind, title, body, data, key, value))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+cols+` FROM notifications
		WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC LIMIT $3
	`, userID, before, limit)
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

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
