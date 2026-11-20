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
	List(ctx context.Context, userID, filter string, onlyID *string, limit, offset int) ([]Conversation, error)
	Membership(ctx context.Context, conversationID, userID string) (Membership, error)
	Messages(ctx context.Context, conversationID string, clearedAt *time.Time, beforeID *string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, senderID, body string) (Message, string, error)
	MarkRead(ctx context.Context, conversationID, userID string) (*time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// List returns the user's conversations. Every row is scoped through the
// caller's own participant row, so foreign conversations are unreachable.
func (r *PGRepository) List(ctx context.Context, userID, filter string, onlyID *string, limit, offset int) ([]Conversation, error) {
	var condition string
	switch filter {
	case FilterNew:
		condition = `c.last_message_at IS NULL`
	case FilterActive:
		condition = `lm.id IS NOT NULL`
	case "any":
		condition = `TRUE`
	default:
		condition = `(c.last_message_at IS NULL OR lm.id IS NOT NULL)`
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT c.id, c.match_id, other.user_id, c.created_at, other.last_read_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages m
		         WHERE m.conversation_id = c.id AND m.sender_id <> $1
		           AND m.created_at > COALESCE(me.last_read_at, '-infinity')
		           AND m.created_at > COALESCE(me.cleared_at, '-infinity')) AS unread
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
		LEFT JOIN LATERAL (
			SELECT m.id, m.sender_id, m.body, m.created_at
			FROM messages m
			WHERE m.conversation_id = c.id AND m.created_at > COALESCE(me.cleared_at, '-infinity')
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1 AND ($4::uuid IS NULL OR c.id = $4) AND `+condition+`
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = me.user_id AND b.blocked_id = other.user_id)
			   OR (b.blocker_id = other.user_id AND b.blocked_id = me.user_id)
		  )
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset, onlyID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Conversation, error) {
		var c Conversation
		var msgID, senderID, body *string
		var msgAt *time.Time
		if err := row.Scan(&c.ID, &c.MatchID, &c.OtherUserID, &c.CreatedAt, &c.OtherLastReadAt,
			&msgID, &senderID, &body, &msgAt, &c.UnreadCount); err != nil {
			return c, err
		}
		if msgID != nil {
			c.LastMessage = &Message{ID: *msgID, ConversationID: c.ID, SenderID: *senderID, Body: *body, CreatedAt: *msgAt}
		}
		return c, nil
	})
}

func (r *PGRepository) Membership(ctx context.Context, conversationID, userID string) (Membership, error) {
	m := Membership{ConversationID: conversationID}
	err := r.dbPool.QueryRow(ctx, `
		SELECT other.user_id, me.cleared_at, other.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = me.user_id AND b.blocked_id = other.user_id)
			   OR (b.blocker_id = other.user_id AND b.blocked_id = me.user_id)
		  )
	`, conversationID, userID).Scan(&m.OtherUserID, &m.ClearedAt, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrRepositoryNotFound
	}
	return m, err
}

func (r *PGRepository) Messages(ctx context.Context, conversationID string, clearedAt *time.Time, beforeID *string, limit int) ([]Message, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND created_at > COALESCE($2::timestamptz, '-infinity')
		  AND ($3::uuid IS NULL OR (created_at, id) < (
			SELECT b.created_at, b.id FROM messages b WHERE b.id = $3 AND b.conversation_id = $1
		  ))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, clearedAt, beforeID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
		return m, err
	})
}

// CreateMessage stores a message, bumps the conversation and marks it read
// for the sender. It also returns the sender's first name for notifications.
func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, body string) (Message, string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise writers and readers of one conversation so a message can never
	// get a timestamp older than a read cursor set while it was uncommitted.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM conversations WHERE id = $1 FOR UPDATE`, conversationID); err != nil {
		return Message{}, "", err
	}

	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
		RETURNING id, conversation_id, sender_id, body, created_at
	`, conversationID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, "", ErrRepositoryNotFound
		}
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations SET last_message_at = GREATEST(COALESCE(last_message_at, '-infinity'), $2) WHERE id = $1
	`, conversationID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = GREATEST(COALESCE(last_read_at, '-infinity'), $3)
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	var firstName string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT first_name FROM profiles WHERE user_id = $1), '')`, senderID).Scan(&firstName); err != nil {
		return Message{}, "", err
	}
	return m, firstName, tx.Commit(ctx)
}

// MarkRead moves the read cursor to the newest message (not to NOW(), which
// could skip a message committed concurrently).
func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (*time.Time, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Wait for in-flight messages of this conversation (see CreateMessage).
	if _, err := tx.Exec(ctx, `SELECT 1 FROM conversations WHERE id = $1 FOR UPDATE`, conversationID); err != nil {
		return nil, err
	}
	var lastRead *time.Time
	err = tx.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = NULLIF(GREATEST(
			COALESCE(last_read_at, '-infinity'),
			COALESCE((SELECT MAX(created_at) FROM messages WHERE conversation_id = $1), '-infinity')
		), '-infinity'::timestamptz)
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&lastRead)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}
	return lastRead, tx.Commit(ctx)
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE conversation_participants
		SET cleared_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}
