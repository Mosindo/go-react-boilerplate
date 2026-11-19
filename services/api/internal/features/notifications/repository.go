package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Create(ctx context.Context, userID, kind string, actorID, conversationID *string) (created bool, err error)
	List(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]Stored, error)
	CursorOf(ctx context.Context, userID, id string) (time.Time, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Create inserts a notification. Message notifications are collapsed: at most one
// unread "message" notification exists per conversation, refreshed on each message.
func (r *PGRepository) Create(ctx context.Context, userID, kind string, actorID, conversationID *string) (bool, error) {
	if kind == TypeMessage && conversationID != nil {
		tag, err := r.db.Exec(ctx, `
			UPDATE notifications SET created_at = clock_timestamp()
			WHERE user_id = $1 AND type = 'message' AND conversation_id = $2 AND read_at IS NULL
		`, userID, *conversationID)
		if err != nil {
			return false, err
		}
		if tag.RowsAffected() > 0 {
			return false, nil
		}
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, conversation_id) VALUES ($1, $2, $3, $4)
	`, userID, kind, actorID, conversationID)
	return err == nil, err
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]Stored, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, actor_id, conversation_id, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, userID, before, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Stored, error) {
		var s Stored
		err := row.Scan(&s.ID, &s.Type, &s.ActorID, &s.ConversationID, &s.ReadAt, &s.CreatedAt)
		return s, err
	})
}

func (r *PGRepository) CursorOf(ctx context.Context, userID, id string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `SELECT created_at FROM notifications WHERE id = $1 AND user_id = $2`, id, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return t, err
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
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
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND conversation_id = $2 AND type = 'message' AND read_at IS NULL
	`, userID, conversationID)
	return err
}

func nullUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
