package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrInvalidCursor        = errors.New("invalid cursor")
)

type Repository interface {
	Membership(ctx context.Context, matchID, userID string) (membership, error)
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]conversationRow, error)
	ListMessages(ctx context.Context, matchID string, after *time.Time, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, matchID, senderID, body string) (Message, string, error)
	MarkRead(ctx context.Context, matchID, userID string) (time.Time, error)
	Clear(ctx context.Context, matchID, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// Membership is the authorisation gate of every conversation endpoint: it
// only succeeds when userID participates in matchID.
func (r *PGRepository) Membership(ctx context.Context, matchID, userID string) (membership, error) {
	var m membership
	err := r.db.QueryRow(ctx, `
		SELECT other.user_id, me.cleared_at, other.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants other ON other.match_id = me.match_id AND other.user_id <> me.user_id
		WHERE me.match_id = $1 AND me.user_id = $2
	`, matchID, userID).Scan(&m.OtherUserID, &m.ClearedAt, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return membership{}, ErrConversationNotFound
	}
	return m, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]conversationRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, other.user_id, m.created_at, other.last_read_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (
		         SELECT COUNT(*) FROM (
		           SELECT 1 FROM messages x
		           WHERE x.match_id = m.id
		             AND x.sender_id <> $1
		             AND x.created_at > COALESCE(me.last_read_at, '-infinity'::TIMESTAMPTZ)
		             AND x.created_at > COALESCE(me.cleared_at, '-infinity'::TIMESTAMPTZ)
		           LIMIT $4
		         ) capped
		       )
		FROM conversation_participants me
		JOIN matches m ON m.id = me.match_id
		JOIN conversation_participants other ON other.match_id = m.id AND other.user_id <> me.user_id
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at FROM messages
		  WHERE match_id = m.id AND created_at > COALESCE(me.cleared_at, '-infinity'::TIMESTAMPTZ)
		  ORDER BY created_at DESC, id DESC
		  LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		ORDER BY COALESCE(m.last_message_at, m.created_at) DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset, maxUnreadCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]conversationRow, 0, limit)
	for rows.Next() {
		var row conversationRow
		var msgID, senderID, body *string
		var sentAt *time.Time
		if err := rows.Scan(&row.MatchID, &row.OtherUserID, &row.CreatedAt, &row.OtherLastReadAt,
			&msgID, &senderID, &body, &sentAt, &row.UnreadCount); err != nil {
			return nil, err
		}
		if msgID != nil {
			row.LastMessage = &Message{ID: *msgID, MatchID: row.MatchID, SenderID: *senderID, Body: *body, CreatedAt: *sentAt}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListMessages returns newest-first messages, optionally before a cursor
// message id (keyset pagination, stable under concurrent inserts).
func (r *PGRepository) ListMessages(ctx context.Context, matchID string, after *time.Time, beforeID string, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	if beforeID == "" {
		rows, err = r.db.Query(ctx, `
			SELECT id, match_id, sender_id, body, created_at FROM messages
			WHERE match_id = $1 AND created_at > COALESCE($2, '-infinity'::TIMESTAMPTZ)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`, matchID, after, limit)
	} else {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1 AND match_id = $2)`, beforeID, matchID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidCursor
		}
		rows, err = r.db.Query(ctx, `
			SELECT m.id, m.match_id, m.sender_id, m.body, m.created_at FROM messages m
			JOIN messages cur ON cur.id = $3
			WHERE m.match_id = $1
			  AND m.created_at > COALESCE($2, '-infinity'::TIMESTAMPTZ)
			  AND (m.created_at, m.id) < (cur.created_at, cur.id)
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $4
		`, matchID, after, beforeID, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt)
		return m, err
	})
}

// CreateMessage stores the message, bumps the conversation and also marks
// it read for the sender. It returns the sender's first name for previews.
func (r *PGRepository) CreateMessage(ctx context.Context, matchID, senderID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, match_id, sender_id, body, created_at
	`, matchID, senderID, body).Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE match_id = $1 AND user_id = $2
	`, matchID, senderID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	var firstName string
	if err := tx.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, senderID).Scan(&firstName); err != nil {
		return Message{}, "", err
	}
	return m, firstName, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, userID string) (time.Time, error) {
	var readAt time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE match_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, matchID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrConversationNotFound
	}
	return readAt, err
}

func (r *PGRepository) Clear(ctx context.Context, matchID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW(), last_read_at = NOW()
		WHERE match_id = $1 AND user_id = $2
	`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
}
