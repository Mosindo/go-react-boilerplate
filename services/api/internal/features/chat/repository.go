package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func notFound(err error) bool {
	var pgErr *pgconn.PgError
	return errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02")
}

// Conversations lists the caller's visible conversations, newest activity first. A conversation the
// user deleted locally stays hidden until a newer message arrives.
func (r *Repository) Conversations(ctx context.Context, userID string, limit, offset int) ([]conversationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END,
		       c.last_message_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at, lm.read_at,
		       (SELECT count(*) FROM messages x
		         WHERE x.conversation_id = c.id AND x.sender_id <> $1 AND x.read_at IS NULL
		           AND x.created_at > COALESCE(cp.hidden_at, '-infinity'))
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at, read_at FROM messages
		  WHERE conversation_id = c.id AND created_at > COALESCE(cp.hidden_at, '-infinity')
		  ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE cp.user_id = $1 AND (cp.hidden_at IS NULL OR cp.hidden_at < c.last_message_at)
		ORDER BY c.last_message_at DESC, c.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conversationRow
	for rows.Next() {
		var c conversationRow
		var mid, msender, mbody *string
		var mat, mread *time.Time
		if err := rows.Scan(&c.ID, &c.MatchID, &c.OtherID, &c.LastMessageAt, &mid, &msender, &mbody, &mat, &mread, &c.Unread); err != nil {
			return nil, err
		}
		if mid != nil {
			c.Last = &Message{ID: *mid, ConversationID: c.ID, SenderID: *msender, Body: *mbody, CreatedAt: *mat, ReadAt: mread}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// OtherParticipant returns the other user of a conversation when userID is a participant.
// Non-members get ErrNotFound, indistinguishable from a non-existent conversation.
func (r *Repository) OtherParticipant(ctx context.Context, userID, conversationID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		SELECT other.user_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $2 AND me.user_id = $1
	`, userID, conversationID).Scan(&other)
	if notFound(err) {
		return "", ErrNotFound
	}
	return other, err
}

func (r *Repository) Messages(ctx context.Context, userID, conversationID string, limit int, before *time.Time) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at, m.read_at
		FROM messages m
		JOIN conversation_participants cp ON cp.conversation_id = m.conversation_id AND cp.user_id = $1
		WHERE m.conversation_id = $2
		  AND m.created_at > COALESCE(cp.hidden_at, '-infinity')
		  AND ($4::timestamptz IS NULL OR m.created_at < $4)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3
	`, userID, conversationID, limit, before)
	if err != nil {
		if notFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Send stores a message after re-checking membership inside the transaction.
func (r *Repository) Send(ctx context.Context, userID, conversationID, body string) (Message, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var other string
	err = tx.QueryRow(ctx, `
		SELECT other.user_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		JOIN conversations c ON c.id = me.conversation_id
		WHERE me.conversation_id = $2 AND me.user_id = $1
		FOR UPDATE OF c
	`, userID, conversationID).Scan(&other)
	if notFound(err) {
		return Message{}, "", ErrNotFound
	}
	if err != nil {
		return Message{}, "", err
	}
	m := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, userID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, "", err
	}
	return m, other, nil
}

// MarkRead marks the other participant's messages as read and returns how many changed.
func (r *Repository) MarkRead(ctx context.Context, userID, conversationID string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE messages SET read_at = NOW()
		WHERE conversation_id = $2 AND sender_id <> $1 AND read_at IS NULL
		  AND EXISTS (SELECT 1 FROM conversation_participants cp WHERE cp.conversation_id = $2 AND cp.user_id = $1)
	`, userID, conversationID)
	if err != nil {
		if notFound(err) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Hide deletes the conversation locally for userID only.
func (r *Repository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NOW() WHERE conversation_id = $2 AND user_id = $1`, userID, conversationID)
	if err != nil {
		if notFound(err) {
			return ErrNotFound
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
