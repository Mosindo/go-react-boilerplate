package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepoNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID, conversationID string, limit, offset int) ([]Conversation, error)
	Membership(ctx context.Context, conversationID, userID string) (membership, error)
	ListMessages(ctx context.Context, conversationID string, after *time.Time, beforeID string, limit int) ([]Message, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
	UnreadTotal(ctx context.Context, userID string) (int, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ListConversations returns the user's active conversations (one per active
// match), most recent activity first. Every per-row value comes from indexed
// lookups, so the cost is proportional to the page size.
func (r *PGRepository) ListConversations(ctx context.Context, userID, conversationID string, limit, offset int) ([]Conversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, m.id, m.created_at, other.user_id, pr.first_name, other.last_read_at,
		       lm.id, lm.body, lm.sender_id, lm.created_at,
		       (SELECT COUNT(*) FROM messages x
		         WHERE x.conversation_id = c.id AND x.sender_id <> $1
		           AND x.created_at > COALESCE(GREATEST(my.last_read_at, my.cleared_at), '-infinity'::timestamptz)) AS unread
		FROM conversation_participants my
		JOIN conversations c ON c.id = my.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
		JOIN profiles pr ON pr.user_id = other.user_id
		LEFT JOIN LATERAL (
			SELECT id, body, sender_id, created_at FROM messages
			WHERE conversation_id = c.id AND created_at > COALESCE(my.cleared_at, '-infinity'::timestamptz)
			ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE my.user_id = $1
		  AND ($2 = '' OR c.id::text = $2)
		  AND (my.cleared_at IS NULL OR c.last_message_at > my.cleared_at)
		ORDER BY COALESCE(c.last_message_at, m.created_at) DESC, c.id
		LIMIT $3 OFFSET $4
	`, userID, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Conversation, error) {
		var conv Conversation
		var msgID, msgBody, msgSender *string
		var msgAt *time.Time
		err := row.Scan(&conv.ID, &conv.MatchID, &conv.MatchedAt, &conv.Peer.UserID, &conv.Peer.FirstName, &conv.PeerLastReadAt,
			&msgID, &msgBody, &msgSender, &msgAt, &conv.UnreadCount)
		if err != nil {
			return conv, err
		}
		if msgID != nil {
			conv.LastMessage = &Message{ID: *msgID, ConversationID: conv.ID, SenderID: *msgSender, Body: *msgBody, CreatedAt: *msgAt}
		}
		return conv, nil
	})
}

// Membership checks that userID participates in an active conversation.
func (r *PGRepository) Membership(ctx context.Context, conversationID, userID string) (membership, error) {
	var m membership
	err := r.db.QueryRow(ctx, `
		SELECT other.user_id, my.cleared_at, other.last_read_at
		FROM conversation_participants my
		JOIN conversations c ON c.id = my.conversation_id
		JOIN matches mt ON mt.id = c.match_id AND mt.ended_at IS NULL
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> my.user_id
		WHERE my.conversation_id = $1 AND my.user_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = my.user_id AND b.blocked_id = other.user_id)
		       OR (b.blocker_id = other.user_id AND b.blocked_id = my.user_id)
		  )
	`, conversationID, userID).Scan(&m.PeerID, &m.ClearedAt, &m.PeerLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return membership{}, errRepoNotFound
	}
	return m, err
}

func scanMessage(row pgx.CollectableRow) (Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	return m, err
}

// ListMessages returns messages newest first (keyset pagination on
// created_at, id), hiding messages older than `after` (local clear).
func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, after *time.Time, beforeID string, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3 = '' OR (created_at, id) < (SELECT created_at, id FROM messages WHERE id::text = $3 AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, after, beforeID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanMessage)
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	var msg Message
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
			RETURNING id, conversation_id, sender_id, body, created_at
		`, conversationID, senderID, body)
		if err != nil {
			return err
		}
		if msg, err = pgx.CollectExactlyOneRow(rows, scanMessage); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
		`, conversationID, senderID, msg.CreatedAt)
		return err
	})
	return msg, err
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&at)
	return at, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_participants my
		JOIN conversations c ON c.id = my.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
		JOIN messages x ON x.conversation_id = c.id
		WHERE my.user_id = $1 AND x.sender_id <> $1
		  AND x.created_at > COALESCE(GREATEST(my.last_read_at, my.cleared_at), '-infinity'::timestamptz)
	`, userID).Scan(&n)
	return n, err
}
