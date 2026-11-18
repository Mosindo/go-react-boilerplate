package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("conversation not found")

type Repository interface {
	Access(ctx context.Context, userID, otherID string) (conversationAccess, error)
	ListChats(ctx context.Context, userID string, limit, offset int) ([]storedChat, error)
	ListMessages(ctx context.Context, conversationID string, limit int, beforeID string) ([]Message, error)
	LastReadAt(ctx context.Context, conversationID, userID string) (*time.Time, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// Access is the single authorisation gate of the chat: a conversation is only
// reachable through an active match between the two members, without any
// block in either direction. Anything else looks like "not found".
func (r *PGRepository) Access(ctx context.Context, userID, otherID string) (conversationAccess, error) {
	var access conversationAccess
	err := r.dbPool.QueryRow(ctx, `
		SELECT m.conversation_id, sender.first_name
		FROM matches m
		JOIN conversation_participants self_cp ON self_cp.conversation_id = m.conversation_id AND self_cp.user_id = $1
		JOIN conversation_participants other_cp ON other_cp.conversation_id = m.conversation_id AND other_cp.user_id = $2
		JOIN profiles sender ON sender.user_id = $1
		WHERE m.user_low_id = LEAST($1::uuid, $2::uuid)
		  AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
		       OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
		  )
	`, userID, otherID).Scan(&access.ConversationID, &access.SenderFirstName)
	if errors.Is(err, pgx.ErrNoRows) {
		return conversationAccess{}, ErrRepositoryNotFound
	}
	return access, err
}

func (r *PGRepository) ListChats(ctx context.Context, userID string, limit, offset int) ([]storedChat, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = p.user_id ORDER BY ph.position LIMIT 1),
		       lm.id, lm.sender_user_id, lm.recipient_user_id, lm.content, lm.created_at,
		       (
		         SELECT COUNT(*) FROM messages um
		         WHERE um.conversation_id = m.conversation_id
		           AND um.sender_user_id <> $1
		           AND (self_cp.last_read_at IS NULL OR um.created_at > self_cp.last_read_at)
		       )::int
		FROM matches m
		JOIN profiles p ON p.user_id = CASE WHEN m.user_low_id = $1 THEN m.user_high_id ELSE m.user_low_id END
		JOIN conversation_participants self_cp ON self_cp.conversation_id = m.conversation_id AND self_cp.user_id = $1
		JOIN LATERAL (
		  SELECT msg.id, msg.sender_user_id, msg.recipient_user_id, msg.content, msg.created_at
		  FROM messages msg
		  WHERE msg.conversation_id = m.conversation_id
		  ORDER BY msg.created_at DESC, msg.id DESC
		  LIMIT 1
		) lm ON TRUE
		WHERE m.user_low_id = $1 OR m.user_high_id = $1
		ORDER BY lm.created_at DESC, lm.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedChat, 0)
	for rows.Next() {
		var c storedChat
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.PhotoID,
			&c.LastMessage.ID, &c.LastMessage.SenderUserID, &c.LastMessage.RecipientUserID, &c.LastMessage.Content, &c.LastMessage.CreatedAt,
			&c.UnreadCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListMessages returns up to limit+1 messages older than beforeID, newest first.
func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, limit int, beforeID string) ([]Message, error) {
	var before any
	if beforeID != "" {
		before = beforeID
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, sender_user_id, recipient_user_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND (
		    $3::uuid IS NULL
		    OR (created_at, id) < (SELECT c.created_at, c.id FROM messages c WHERE c.id = $3::uuid AND c.conversation_id = $1)
		  )
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, conversationID, limit+1, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit+1)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) LastReadAt(ctx context.Context, conversationID, userID string) (*time.Time, error) {
	var readAt *time.Time
	err := r.dbPool.QueryRow(ctx, `
		SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return readAt, err
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_user_id, recipient_user_id, content, created_at
	`, conversationID, senderID, recipientID, content).Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var readAt time.Time
	err := r.dbPool.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, '-infinity'::timestamptz), NOW())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrRepositoryNotFound
	}
	return readAt, err
}
