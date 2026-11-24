package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, kind, title, body string, data map[string]any) (Notification, error)
	List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id, type, title, body, data, read_at IS NOT NULL, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var data []byte
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &data, &n.IsRead, &n.CreatedAt, &n.ReadAt)
	n.Data = json.RawMessage(data)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, kind, title, body string, data map[string]any) (Notification, error) {
	if data == nil {
		data = map[string]any{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, kind, title, body, raw))
}

// List paginates by keyset on (created_at, id) to stay O(page size).
func (r *PGRepository) List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+`
		FROM notifications n
		WHERE n.user_id = $1
		  AND ($2::uuid IS NULL OR (n.created_at, n.id) < (
				SELECT c.created_at, c.id FROM notifications c WHERE c.id = $2::uuid AND c.user_id = $1))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $3
	`, userID, nullableUUID(beforeID), limit)
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
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func nullableUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
