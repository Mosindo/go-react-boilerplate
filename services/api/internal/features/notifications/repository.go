package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, typ, title, body string, conversationID *string) (Notification, error)
	// UpsertMessage keeps a single unread message notification per conversation.
	UpsertMessage(ctx context.Context, userID, conversationID, title, body string) (Notification, error)
	List(ctx context.Context, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const cols = `id, type, title, body, conversation_id, read_at, created_at`

func scanOne(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.ConversationID, &n.ReadAt, &n.CreatedAt)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, typ, title, body string, conversationID *string) (Notification, error) {
	return scanOne(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, conversation_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+cols, userID, typ, title, body, conversationID))
}

func (r *PGRepository) UpsertMessage(ctx context.Context, userID, conversationID, title, body string) (Notification, error) {
	n, err := scanOne(r.db.QueryRow(ctx, `
		UPDATE notifications SET title = $3, body = $4, created_at = NOW()
		WHERE id = (SELECT id FROM notifications
		            WHERE user_id = $1 AND conversation_id = $2 AND type = 'message' AND read_at IS NULL
		            ORDER BY created_at DESC LIMIT 1)
		RETURNING `+cols, userID, conversationID, title, body))
	if errors.Is(err, pgx.ErrNoRows) {
		return r.Create(ctx, userID, "message", title, body, &conversationID)
	}
	return n, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	if beforeTime == nil {
		rows, err = r.db.Query(ctx, `SELECT `+cols+` FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `SELECT `+cols+` FROM notifications WHERE user_id = $1
			AND (created_at, id) < ($2, $3::uuid) ORDER BY created_at DESC, id DESC LIMIT $4`, userID, *beforeTime, beforeID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		n, err := scanOne(rows)
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
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrNotFound
	}
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND conversation_id = $2 AND type = 'message' AND read_at IS NULL`, userID, conversationID)
	return err
}
