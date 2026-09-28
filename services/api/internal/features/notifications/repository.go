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

// DB is satisfied by *pgxpool.Pool and pgx.Tx, so the creators below run inside
// the caller's transaction.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

const columns = `id, type, title, body, data, read_at, created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &n.Data)
	}
	n.CreatedAt = n.CreatedAt.UTC()
	return n, nil
}

// Create inserts a notification through q.
func Create(ctx context.Context, q DB, n NewNotification) (Notification, error) {
	data, _ := json.Marshal(n.Data)
	return scan(q.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING `+columns, n.UserID, n.Type, n.Title, n.Body, string(data)))
}

// UpsertMessage coalesces message notifications: while the recipient has an
// unread notification for the conversation it is refreshed instead of adding a
// new row per message.
func UpsertMessage(ctx context.Context, q DB, n NewNotification) (Notification, error) {
	updated, err := scan(q.QueryRow(ctx, `
		UPDATE notifications
		SET title = $3, body = $4, created_at = NOW()
		WHERE id = (
			SELECT id FROM notifications
			WHERE user_id = $1 AND type = 'message' AND read_at IS NULL
			  AND data ->> 'conversationId' = $2
			ORDER BY created_at DESC LIMIT 1
		)
		RETURNING `+columns, n.UserID, n.Data.ConversationID, n.Title, n.Body))
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, err
	}
	return Create(ctx, q, n)
}

// MarkConversationRead marks the message notifications of a conversation read.
func MarkConversationRead(ctx context.Context, q DB, userID, conversationID string) error {
	_, err := q.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL
		  AND data ->> 'conversationId' = $2
	`, userID, conversationID)
	return err
}

// DeleteForMatch removes every notification that points at a match (unmatch).
func DeleteForMatch(ctx context.Context, q DB, matchID string) error {
	_, err := q.Exec(ctx, `DELETE FROM notifications WHERE data ->> 'matchId' = $1`, matchID)
	return err
}

type Repository interface {
	List(ctx context.Context, userID string, before *Cursor, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type Cursor struct {
	CreatedAt time.Time `json:"t"`
	ID        string    `json:"id"`
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string, before *Cursor, limit int) ([]Notification, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if before == nil {
		rows, err = r.pool.Query(ctx, `
			SELECT `+columns+` FROM notifications
			WHERE user_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`, userID, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT `+columns+` FROM notifications
			WHERE user_id = $1 AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC
			LIMIT $4`, userID, before.CreatedAt, before.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
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
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead reports whether the notification exists and belongs to userID.
func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	var found bool
	err := r.pool.QueryRow(ctx, `
		WITH target AS (
			SELECT id FROM notifications WHERE id = $1 AND user_id = $2
		), upd AS (
			UPDATE notifications SET read_at = NOW()
			WHERE id IN (SELECT id FROM target) AND read_at IS NULL
		)
		SELECT EXISTS (SELECT 1 FROM target)`, id, userID).Scan(&found)
	return found, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
