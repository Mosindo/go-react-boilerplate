package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

type Repository interface {
	Access(ctx context.Context, userID, conversationID string) (Access, error)
	ListConversations(ctx context.Context, userID, beforeID string, limit int) ([]StoredConversation, error)
	UnreadConversations(ctx context.Context, userID string) (int, error)
	ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, senderID, content string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// Access is the single authorisation gate of the chat: the caller must be a
// participant, the match must be active and no block may exist between both users.
func (r *PGRepository) Access(ctx context.Context, userID, conversationID string) (Access, error) {
	var a Access
	err := r.dbPool.QueryRow(ctx, `
		SELECT c.id, m.id, m.created_at, other.user_id, me.hidden_at, other.last_read_at
		FROM conversations c
		JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
		WHERE c.id = $2
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = other.user_id)
		         OR (b.blocker_id = other.user_id AND b.blocked_id = $1))
	`, userID, conversationID).Scan(&a.ConversationID, &a.MatchID, &a.MatchedAt, &a.OtherUserID, &a.HiddenAt, &a.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Access{}, ErrConversationNotFound
	}
	return a, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID, beforeID string, limit int) ([]StoredConversation, error) {
	rows, err := r.dbPool.Query(ctx, `
		WITH mine AS (
			SELECT c.id, c.match_id, m.created_at AS matched_at,
			       COALESCE(c.last_message_at, c.created_at) AS activity_at,
			       me.last_read_at, me.hidden_at, other.user_id AS other_id, other.last_read_at AS other_read_at
			FROM conversation_participants me
			JOIN conversations c ON c.id = me.conversation_id
			JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
			JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
			WHERE me.user_id = $1
			  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		)
		SELECT x.id, x.match_id, x.matched_at, x.other_id, x.other_read_at,
		       lm.id, lm.sender_id, lm.content, lm.created_at,
		       (SELECT COUNT(*) FROM messages um
		         WHERE um.conversation_id = x.id AND um.sender_id <> $1
		           AND um.created_at > COALESCE(GREATEST(x.last_read_at, x.hidden_at), '-infinity'::timestamptz)) AS unread
		FROM mine x
		LEFT JOIN LATERAL (
			SELECT id, sender_id, content, created_at FROM messages
			WHERE conversation_id = x.id
			  AND (x.hidden_at IS NULL OR created_at > x.hidden_at)
			ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE $2 = '' OR (x.activity_at, x.id) < (SELECT activity_at, id FROM mine WHERE id::text = $2)
		ORDER BY x.activity_at DESC, x.id DESC
		LIMIT $3
	`, userID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredConversation{}
	for rows.Next() {
		var c StoredConversation
		var lmID, lmSender, lmContent *string
		var lmAt *time.Time
		if err := rows.Scan(&c.ID, &c.MatchID, &c.MatchedAt, &c.OtherUserID, &c.OtherLastReadAt,
			&lmID, &lmSender, &lmContent, &lmAt, &c.UnreadCount); err != nil {
			return nil, err
		}
		if lmID != nil {
			c.LastMessage = &Message{ID: *lmID, ConversationID: c.ID, SenderID: *lmSender, Content: *lmContent, CreatedAt: *lmAt}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadConversations(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.dbPool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
		WHERE me.user_id = $1
		  AND EXISTS (
		      SELECT 1 FROM messages um
		      WHERE um.conversation_id = c.id AND um.sender_id <> $1
		        AND um.created_at > COALESCE(GREATEST(me.last_read_at, me.hidden_at), '-infinity'::timestamptz))
	`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, beforeID string, limit int) ([]Message, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3 = '' OR (created_at, id) < (
		        SELECT created_at, id FROM messages WHERE id::text = $3 AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, hiddenAt, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, content string) (Message, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, content, created_at
	`, conversationID, senderID, content).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var readAt time.Time
	err := r.dbPool.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrConversationNotFound
	}
	return readAt, err
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp(), last_read_at = clock_timestamp()
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
