package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, n New) (Notification, bool, error)
	List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

const columns = `id, user_id, type, title, body, data, is_read, created_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
		return Notification{}, err
	}
	if err := json.Unmarshal(raw, &n.Data); err != nil {
		n.Data = map[string]string{}
	}
	return n, nil
}

// Create inserts a notification. The bool is false when it was skipped by dedupe.
func (r *PGRepository) Create(ctx context.Context, n New) (Notification, bool, error) {
	data := n.Data
	if data == nil {
		data = map[string]string{}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, false, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, $2, $3, $4, $5::jsonb
		WHERE NOT ($6 AND EXISTS (
			SELECT 1 FROM notifications x
			WHERE x.user_id = $1 AND x.type = $2 AND x.is_read = FALSE
			  AND x.data->>'matchId' IS NOT DISTINCT FROM ($5::jsonb)->>'matchId'
		))
		RETURNING `+columns, n.UserID, n.Type, n.Title, n.Body, string(payload), n.Dedupe)
	created, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	return created, err == nil, err
}

func (r *PGRepository) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1 AND (NOT $2 OR is_read = FALSE)
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4
	`, userID, unreadOnly, limit, offset)
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

func (r *PGRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&count)
	return count, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	n, err := scan(r.pool.QueryRow(ctx, `
		UPDATE notifications
		SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}
