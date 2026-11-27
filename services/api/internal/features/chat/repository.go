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
	ErrSenderRestricted     = errors.New("sender restricted by moderation")
)

type Repository interface {
	GetConversation(ctx context.Context, userID, matchID string) (Conversation, error)
	ListMessages(ctx context.Context, conversationID string, beforeAt *time.Time, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, userID, matchID, content, clientID string) (Message, Conversation, error)
	MarkRead(ctx context.Context, userID, matchID string) (Conversation, time.Time, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// membershipSQL resolves a match's conversation only when the caller is a
// participant: non-members get "not found", never someone else's messages.
const membershipSQL = `
	SELECT c.id, c.match_id, other.user_id, other.last_read_at
	FROM conversations c
	JOIN conversation_participants self ON self.conversation_id = c.id AND self.user_id = $1
	JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
	WHERE c.match_id = $2`

func scanConversation(row pgx.Row) (Conversation, error) {
	var conv Conversation
	if err := row.Scan(&conv.ID, &conv.MatchID, &conv.OtherUserID, &conv.OtherLastReadAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, err
	}
	return conv, nil
}

func (r *PGRepository) GetConversation(ctx context.Context, userID, matchID string) (Conversation, error) {
	return scanConversation(r.dbPool.QueryRow(ctx, membershipSQL, userID, matchID))
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, beforeAt *time.Time, beforeID string, limit int) ([]Message, error) {
	var cursorID any
	if beforeID != "" {
		cursorID = beforeID
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, sender_user_id, content, COALESCE(client_id, ''), created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, beforeAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderUserID, &m.Content, &m.ClientID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateMessage inserts a message; a retried send with the same clientId
// returns the original message instead of creating a duplicate.
func (r *PGRepository) CreateMessage(ctx context.Context, userID, matchID, content, clientID string) (Message, Conversation, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Message{}, Conversation{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	conv, err := scanConversation(tx.QueryRow(ctx, membershipSQL+` FOR SHARE OF c`, userID, matchID))
	if err != nil {
		return Message{}, Conversation{}, err
	}
	var restricted bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1 AND moderation_hidden_at IS NOT NULL)
	`, userID).Scan(&restricted); err != nil {
		return Message{}, Conversation{}, err
	}
	if restricted {
		return Message{}, Conversation{}, ErrSenderRestricted
	}

	var clientArg any
	if clientID != "" {
		clientArg = clientID
	}

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content, client_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (conversation_id, sender_user_id, client_id) WHERE client_id IS NOT NULL DO NOTHING
		RETURNING id, sender_user_id, content, COALESCE(client_id, ''), created_at
	`, conv.ID, userID, conv.OtherUserID, content, clientArg).Scan(&m.ID, &m.SenderUserID, &m.Content, &m.ClientID, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id, sender_user_id, content, COALESCE(client_id, ''), created_at
			FROM messages
			WHERE conversation_id = $1 AND sender_user_id = $2 AND client_id = $3
		`, conv.ID, userID, clientID).Scan(&m.ID, &m.SenderUserID, &m.Content, &m.ClientID, &m.CreatedAt)
		if err != nil {
			return Message{}, Conversation{}, err
		}
		return m, conv, tx.Commit(ctx)
	}
	if err != nil {
		return Message{}, Conversation{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE conversations SET last_message_at = $2, updated_at = NOW() WHERE id = $1
	`, conv.ID, m.CreatedAt); err != nil {
		return Message{}, Conversation{}, err
	}
	// The sender has obviously read the conversation up to their message.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conv.ID, userID, m.CreatedAt); err != nil {
		return Message{}, Conversation{}, err
	}
	return m, conv, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, matchID string) (Conversation, time.Time, error) {
	conv, err := r.GetConversation(ctx, userID, matchID)
	if err != nil {
		return Conversation{}, time.Time{}, err
	}
	var readAt time.Time
	err = r.dbPool.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, 'epoch'::timestamptz), NOW())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conv.ID, userID).Scan(&readAt)
	if err != nil {
		return Conversation{}, time.Time{}, err
	}
	return conv, readAt, nil
}
