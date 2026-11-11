package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Upsert(ctx context.Context, in Input, data []byte) (Notification, error)
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkReadByDedupe(ctx context.Context, userID, dedupeKey string) error
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

// Upsert inserts a notification, or refreshes the unread one that shares the
// same dedupe key (e.g. one "new messages" entry per conversation).
func (r *PGRepository) Upsert(ctx context.Context, in Input, data []byte) (Notification, error) {
	var dedupe any
	if in.DedupeKey != "" {
		dedupe = in.DedupeKey
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, dedupe_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, dedupe_key) WHERE is_read = FALSE AND dedupe_key IS NOT NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, data = EXCLUDED.data, created_at = NOW()
		RETURNING `+columns, in.UserID, in.Type, in.Title, in.Body, data, dedupe))
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+` FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3
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
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
	`, id, userID)
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
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

func (r *PGRepository) MarkReadByDedupe(ctx context.Context, userID, dedupeKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND dedupe_key = $2 AND is_read = FALSE
	`, userID, dedupeKey)
	return err
}
