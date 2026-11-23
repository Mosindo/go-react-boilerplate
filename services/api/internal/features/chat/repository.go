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
	ErrCursorNotFound       = errors.New("cursor message not found")
)

type Repository interface {
	GetMembership(ctx context.Context, userID, conversationID string) (Membership, error)
	ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error)
	MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// GetMembership only succeeds when the caller participates in the
// conversation, the conversation belongs to an active match and nobody
// blocked the other. Every chat operation goes through it.
func (r *PGRepository) GetMembership(ctx context.Context, userID, conversationID string) (Membership, error) {
	var m Membership
	err := r.db.QueryRow(ctx, `
		SELECT c.id, other.user_id, other.last_read_at
		FROM conversations c
		JOIN matches mt ON mt.conversation_id = c.id
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
		WHERE c.id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = other.user_id)
			   OR (b.blocker_id = other.user_id AND b.blocked_id = $1)
		  )
	`, userID, conversationID).Scan(&m.ConversationID, &m.OtherUserID, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrConversationNotFound
	}
	return m, err
}

// ListMessages returns newest-first messages using keyset pagination.
func (r *PGRepository) ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	if beforeID == "" {
		rows, err = r.db.Query(ctx, `
			SELECT id, conversation_id, sender_user_id, content, created_at
			FROM messages
			WHERE conversation_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`, conversationID, limit)
	} else {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1 AND conversation_id = $2)`, beforeID, conversationID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCursorNotFound
		}
		rows, err = r.db.Query(ctx, `
			SELECT m.id, m.conversation_id, m.sender_user_id, m.content, m.created_at
			FROM messages m, (SELECT created_at, id FROM messages WHERE id = $2) cursor
			WHERE m.conversation_id = $1
			  AND (m.created_at, m.id) < (cursor.created_at, cursor.id)
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $3
		`, conversationID, beforeID, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &m.Content, &m.CreatedAt)
		return m, err
	})
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, sender_user_id, content, created_at
	`, conversationID, senderID, recipientID, content).Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &m.Content, &m.CreatedAt)
	if err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var readAt time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, 'epoch'::timestamptz), NOW())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrConversationNotFound
	}
	return readAt, err
}
