package chat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListChats(ctx context.Context, organizationID, userID string) ([]ChatSummary, error)
	ListChatsWithPagination(ctx context.Context, organizationID, userID string, limit, offset int) ([]ChatSummary, error)
	ListMessages(ctx context.Context, userID, otherUserID string, limit int) ([]ChatMessage, error)
	CreateMessage(ctx context.Context, userID, otherUserID, content string) (ChatMessage, error)
	UserExists(ctx context.Context, organizationID, userID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListChats(ctx context.Context, organizationID, userID string) ([]ChatSummary, error) {
	return r.ListChatsWithPagination(ctx, organizationID, userID, 100, 0)
}

func (r *PGRepository) ListChatsWithPagination(ctx context.Context, organizationID, userID string, limit, offset int) ([]ChatSummary, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT
			u.id,
			u.email,
			u.created_at,
			lm.content,
			lm.created_at
		FROM conversations c
		JOIN conversation_participants self_cp
		  ON self_cp.conversation_id = c.id
		 AND self_cp.user_id = $1
		JOIN conversation_participants other_cp
		  ON other_cp.conversation_id = c.id
		 AND other_cp.user_id <> $1
		JOIN users u
		  ON u.id = other_cp.user_id
		 AND u.organization_id = $2
		LEFT JOIN LATERAL (
			SELECT m.content, m.created_at
			FROM messages m
			WHERE m.conversation_id = c.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		) lm ON true
		WHERE c.kind = 'direct'
		ORDER BY COALESCE(lm.created_at, c.updated_at, c.created_at) DESC, u.id DESC
		LIMIT $3 OFFSET $4
	`, userID, organizationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := make([]ChatSummary, 0)
	for rows.Next() {
		var chat ChatSummary
		var lastContent sql.NullString
		var lastCreatedAt sql.NullTime
		if err := rows.Scan(&chat.UserID, &chat.UserEmail, &chat.UserCreatedAt, &lastContent, &lastCreatedAt); err != nil {
			return nil, err
		}
		if lastContent.Valid && lastCreatedAt.Valid {
			chat.LastMessage = &ChatMessagePreview{
				Content:   lastContent.String,
				CreatedAt: lastCreatedAt.Time,
			}
		}
		chats = append(chats, chat)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return chats, nil
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, otherUserID string, limit int) ([]ChatMessage, error) {
	conversationID, err := r.findDirectConversationID(ctx, userID, otherUserID)
	if err != nil {
		return nil, err
	}
	if conversationID == "" {
		return []ChatMessage{}, nil
	}

	rows, err := r.dbPool.Query(ctx, `
		SELECT id, sender_user_id, recipient_user_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2
	`, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ChatMessage, 0)
	for rows.Next() {
		var message ChatMessage
		if err := rows.Scan(&message.ID, &message.SenderUserID, &message.RecipientUserID, &message.Content, &message.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return messages, nil
}

func (r *PGRepository) CreateMessage(ctx context.Context, userID, otherUserID, content string) (ChatMessage, error) {
	tx, err := r.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ChatMessage{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	conversationID, err := getOrCreateDirectConversation(ctx, tx, userID, otherUserID)
	if err != nil {
		return ChatMessage{}, err
	}

	var message ChatMessage
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_user_id, recipient_user_id, content, created_at
	`, conversationID, userID, otherUserID, content).Scan(
		&message.ID,
		&message.SenderUserID,
		&message.RecipientUserID,
		&message.Content,
		&message.CreatedAt,
	)
	if err != nil {
		return ChatMessage{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE conversations
		SET updated_at = NOW()
		WHERE id = $1
	`, conversationID); err != nil {
		return ChatMessage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ChatMessage{}, err
	}
	return message, nil
}

func (r *PGRepository) UserExists(ctx context.Context, organizationID, userID string) (bool, error) {
	var exists bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE id = $1
			  AND organization_id = $2
		)
	`, userID, organizationID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *PGRepository) findDirectConversationID(ctx context.Context, userID, otherUserID string) (string, error) {
	var conversationID string
	err := r.dbPool.QueryRow(ctx, `
		SELECT id
		FROM conversations
		WHERE direct_key = $1
	`, directConversationKey(userID, otherUserID)).Scan(&conversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return conversationID, nil
}

func getOrCreateDirectConversation(ctx context.Context, tx pgx.Tx, userID, otherUserID string) (string, error) {
	var conversationID string
	err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, created_by_user_id)
		VALUES ('direct', $1, $2)
		ON CONFLICT (direct_key)
		DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, directConversationKey(userID, otherUserID), userID).Scan(&conversationID)
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, userID, otherUserID); err != nil {
		return "", err
	}

	return conversationID, nil
}

func directConversationKey(userID, otherUserID string) string {
	if userID < otherUserID {
		return userID + ":" + otherUserID
	}
	return otherUserID + ":" + userID
}
