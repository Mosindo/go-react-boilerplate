package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before time.Time, limit int) ([]storedConversation, error)
	ListMessages(ctx context.Context, userID, conversationID string, before time.Time, limit int) ([]Message, error)
	SendMessage(ctx context.Context, userID, conversationID, body string) (Message, string, error)
	MarkRead(ctx context.Context, userID, conversationID string) (otherUserID string, err error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before time.Time, limit int) ([]storedConversation, error) {
	var cursor any
	if !before.IsZero() {
		cursor = before
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id::text, c.match_id::text, op.user_id::text, op.display_name,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = op.user_id ORDER BY ph.position LIMIT 1),
		       c.last_activity_at,
		       lm.id::text, lm.sender_id::text, lm.body, lm.created_at,
		       (other.last_read_at IS NOT NULL AND other.last_read_at >= lm.created_at),
		       un.n
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN profiles op ON op.user_id = CASE WHEN m.user_a = me.user_id THEN m.user_b ELSE m.user_a END
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id = op.user_id
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at FROM messages
		  WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		CROSS JOIN LATERAL (
		  SELECT COUNT(*)::int AS n FROM messages
		  WHERE conversation_id = c.id AND sender_id <> me.user_id
		    AND (me.last_read_at IS NULL OR created_at > me.last_read_at)
		) un
		WHERE me.user_id = $1
		  AND (me.hidden_at IS NULL OR c.last_activity_at > me.hidden_at)
		  AND ($2::timestamptz IS NULL OR c.last_activity_at < $2)
		ORDER BY c.last_activity_at DESC, c.id DESC
		LIMIT $3`, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []storedConversation
	for rows.Next() {
		var c storedConversation
		var msgID, msgSender, msgBody *string
		var msgAt *time.Time
		var read *bool
		if err := rows.Scan(&c.ID, &c.MatchID, &c.OtherUserID, &c.OtherName, &c.OtherPhotoID, &c.LastActivityAt,
			&msgID, &msgSender, &msgBody, &msgAt, &read, &c.UnreadCount); err != nil {
			return nil, err
		}
		if msgID != nil {
			c.LastMessage = &Message{ID: *msgID, ConversationID: c.ID, SenderID: *msgSender, Body: *msgBody, CreatedAt: *msgAt, ReadByOther: read != nil && *read}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) isParticipant(ctx context.Context, userID, conversationID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)`, conversationID, userID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, before time.Time, limit int) ([]Message, error) {
	ok, err := r.isParticipant(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var cursor any
	if !before.IsZero() {
		cursor = before
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id::text, m.conversation_id::text, m.sender_id::text, m.body, m.created_at,
		       m.sender_id = $2 AND other.last_read_at IS NOT NULL AND other.last_read_at >= m.created_at
		FROM messages m
		JOIN conversation_participants other ON other.conversation_id = m.conversation_id AND other.user_id <> $2
		WHERE m.conversation_id = $1 AND ($3::timestamptz IS NULL OR m.created_at < $3)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4`, conversationID, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadByOther); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SendMessage stores the message, advances the conversation, marks it read for
// the sender and raises one coalesced notification for the recipient.
func (r *PGRepository) SendMessage(ctx context.Context, userID, conversationID, body string) (Message, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer tx.Rollback(ctx)

	var recipient, senderName string
	err = tx.QueryRow(ctx, `
		SELECT other.user_id::text, p.display_name
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		JOIN profiles p ON p.user_id = me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, conversationID, userID).Scan(&recipient, &senderName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, "", ErrNotFound
	}
	if err != nil {
		return Message{}, "", err
	}

	msg := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, created_at`, conversationID, userID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_activity_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID, msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, 'message', 'Nouveau message', $2 || ' vous a écrit.',
		       jsonb_build_object('conversationId', $3::text, 'userId', $4::text)
		WHERE NOT EXISTS (
		  SELECT 1 FROM notifications
		  WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'conversationId' = $3::text)`,
		recipient, senderName, conversationID, userID); err != nil {
		return Message{}, "", err
	}
	return msg, recipient, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var other string
	err = tx.QueryRow(ctx, `
		UPDATE conversation_participants me SET last_read_at = NOW()
		FROM conversation_participants o
		WHERE me.conversation_id = $1 AND me.user_id = $2 AND o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		RETURNING o.user_id::text`, conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'conversationId' = $2::text`, userID, conversationID); err != nil {
		return "", err
	}
	return other, tx.Commit(ctx)
}

// Hide removes the conversation from the caller's list only. It reappears when
// a new message arrives, and the other participant is unaffected.
func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
