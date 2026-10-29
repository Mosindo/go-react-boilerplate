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

// Execer is satisfied by *pgxpool.Pool and pgx.Tx so other features can create
// notifications inside their own transactions.
type Execer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Insert stores a notification for userID.
func Insert(ctx context.Context, q Execer, userID, typ string, data map[string]any) (Notification, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	n := Notification{Type: typ, Data: data}
	err = q.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, data) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, userID, typ, payload).Scan(&n.ID, &n.CreatedAt)
	return n, err
}

// InsertMessageUnlessPending stores a "message" notification only if the user has
// no unread one for the same conversation, so a burst of messages yields one entry.
func InsertMessageUnlessPending(ctx context.Context, q Execer, userID, conversationID string, data map[string]any) (*Notification, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	n := Notification{Type: TypeMessage, Data: data}
	err = q.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, data)
		SELECT $1, 'message', $3
		WHERE NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2)
		RETURNING id, created_at
	`, userID, conversationID, payload).Scan(&n.ID, &n.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, data, read_at, created_at FROM notifications
		WHERE user_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) {
		var n Notification
		var raw []byte
		if err := row.Scan(&n.ID, &n.Type, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
			return n, err
		}
		n.Data = map[string]any{}
		_ = json.Unmarshal(raw, &n.Data)
		return n, nil
	})
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
