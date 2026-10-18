package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

// Execer is satisfied by *pgxpool.Pool and pgx.Tx, so other features can create
// notifications inside their own transaction.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Insert stores a notification for userID using the caller's executor.
func Insert(ctx context.Context, q Execer, userID, kind, title, body string, data map[string]string) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, userID, kind, title, body, string(raw))
	return err
}

type Repository interface {
	List(ctx context.Context, userID string, before *cursor, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type cursor struct {
	CreatedAt time.Time
	ID        string
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string, before *cursor, limit int) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	const cols = `id, type, title, body, data, is_read, created_at, read_at`
	if before == nil {
		rows, err = r.db.Query(ctx, `SELECT `+cols+` FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `SELECT `+cols+` FROM notifications WHERE user_id = $1
			AND (created_at, id) < ($2, $3::uuid) ORDER BY created_at DESC, id DESC LIMIT $4`,
			userID, before.CreatedAt, before.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		n.Data = map[string]string{}
		_ = json.Unmarshal(raw, &n.Data)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND NOT is_read`, userID).Scan(&n)
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
		UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND NOT is_read
	`, userID)
	return err
}
