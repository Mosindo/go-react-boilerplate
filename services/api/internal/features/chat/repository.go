package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID string, before *time.Time, limit int) ([]Message, error)
	// CreateMessage stores a message and returns it with the recipient's ID and
	// whether a new unread-message notification was created for them.
	CreateMessage(ctx context.Context, userID, conversationID, content string) (msg Message, recipientID string, notified bool, err error)
	// MarkRead marks the peer's messages as read and returns the peer's ID and the count.
	MarkRead(ctx context.Context, userID, conversationID string) (peerID string, count int, readAt time.Time, err error)
	// Hide clears the conversation history for userID only.
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// visibleSince filters messages a participant can see: everything newer than the
// point where they cleared the conversation.
const visibleSince = `(me.hidden_at IS NULL OR msg.created_at > me.hidden_at)`

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, o.user_id, o.first_name,
		       EXTRACT(YEAR FROM age(CURRENT_DATE, o.birthdate))::int,
		       ph.id, ph.position, ph.width, ph.height, left(md5(ph.storage_key), 8),
		       lm.id, lm.sender_id, lm.content, lm.created_at, lm.read_at,
		       (SELECT count(*) FROM messages msg
		         WHERE msg.conversation_id = c.id AND msg.sender_id <> $1 AND msg.read_at IS NULL AND `+visibleSince+`)
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN profiles o ON o.user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		LEFT JOIN photos ph ON ph.user_id = o.user_id AND ph.position = 0
		JOIN LATERAL (
			SELECT msg.id, msg.sender_id, msg.content, msg.created_at, msg.read_at
			FROM messages msg
			WHERE msg.conversation_id = c.id AND `+visibleSince+`
			ORDER BY msg.created_at DESC, msg.id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND ($2::timestamptz IS NULL OR lm.created_at < $2)
		ORDER BY lm.created_at DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Conversation, 0, limit)
	for rows.Next() {
		var c Conversation
		var photoID *string
		var pos, w, h *int
		var version *string
		var last Message
		if err := rows.Scan(&c.ID, &c.MatchID, &c.Peer.UserID, &c.Peer.FirstName, &c.Peer.Age,
			&photoID, &pos, &w, &h, &version,
			&last.ID, &last.SenderID, &last.Content, &last.CreatedAt, &last.ReadAt, &c.UnreadCount); err != nil {
			return nil, err
		}
		if photoID != nil {
			c.Peer.Photo = &profiles.PhotoRef{ID: *photoID, Position: *pos, Width: *w, Height: *h, Version: *version}
		}
		last.ConversationID = c.ID
		c.LastMessage = &last
		c.LastMessageAt = last.CreatedAt
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, before *time.Time, limit int) ([]Message, error) {
	var participant bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
	`, conversationID, userID).Scan(&participant); err != nil {
		return nil, err
	}
	if !participant {
		return nil, ErrConversationNotFound
	}

	rows, err := r.pool.Query(ctx, `
		SELECT msg.id, msg.conversation_id, msg.sender_id, msg.content, msg.created_at, msg.read_at
		FROM messages msg
		JOIN conversation_participants me ON me.conversation_id = msg.conversation_id AND me.user_id = $2
		WHERE msg.conversation_id = $1
		  AND `+visibleSince+`
		  AND ($3::timestamptz IS NULL OR msg.created_at < $3)
		ORDER BY msg.created_at DESC, msg.id DESC
		LIMIT $4
	`, conversationID, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	// Return chronological order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *PGRepository) CreateMessage(ctx context.Context, userID, conversationID, content string) (Message, string, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Message{}, "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var recipientID string
	err = tx.QueryRow(ctx, `
		SELECT other.user_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&recipientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, "", false, ErrConversationNotFound
		}
		return Message{}, "", false, err
	}

	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, content) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, content, created_at, read_at
	`, conversationID, userID, content).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt); err != nil {
		return Message{}, "", false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, "", false, err
	}
	// One unread "message" notification per conversation (see unique partial index).
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_user_id, conversation_id)
		VALUES ($1, 'message', $2, $3)
		ON CONFLICT (user_id, conversation_id) WHERE type = 'message' AND read_at IS NULL DO NOTHING
	`, recipientID, userID, conversationID)
	if err != nil {
		return Message{}, "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, "", false, err
	}
	return m, recipientID, tag.RowsAffected() > 0, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (string, int, time.Time, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", 0, time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var peerID string
	err = tx.QueryRow(ctx, `
		SELECT other.user_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&peerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, time.Time{}, ErrConversationNotFound
		}
		return "", 0, time.Time{}, err
	}

	var readAt time.Time
	if err := tx.QueryRow(ctx, `SELECT NOW()`).Scan(&readAt); err != nil {
		return "", 0, time.Time{}, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, conversationID, userID, readAt)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = $3
		WHERE user_id = $2 AND conversation_id = $1 AND type = 'message' AND read_at IS NULL
	`, conversationID, userID, readAt); err != nil {
		return "", 0, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, time.Time{}, err
	}
	return peerID, int(tag.RowsAffected()), readAt, nil
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
}
