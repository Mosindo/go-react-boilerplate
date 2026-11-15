package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Membership(ctx context.Context, userID, conversationID string) (Membership, error)
	ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int, onlyID string) ([]ConversationRow, error)
	ListMessages(ctx context.Context, conversationID string, after *time.Time, cursor *httpx.Cursor, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, userID, conversationID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error)
	Clear(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Membership is the authorisation gate of every chat operation: it only
// returns a row when userID participates in the conversation.
func (r *PGRepository) Membership(ctx context.Context, userID, conversationID string) (Membership, error) {
	var m Membership
	err := r.db.QueryRow(ctx, `
		SELECT me.conversation_id, other.user_id, me.cleared_at, other.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants other
		  ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&m.ConversationID, &m.OtherUserID, &m.ClearedAt, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrConversationNotFound
	}
	return m, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int, onlyID string) ([]ConversationRow, error) {
	var cursorAt *time.Time
	var cursorID *string
	if cursor != nil {
		cursorAt, cursorID = &cursor.At, &cursor.ID
	}
	var only *string
	if onlyID != "" {
		only = &onlyID
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, m.created_at, other.user_id, other.last_read_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages um
		         WHERE um.conversation_id = c.id AND um.sender_id <> $1
		           AND um.created_at > COALESCE(me.last_read_at, '-infinity')
		           AND um.created_at > COALESCE(me.cleared_at, '-infinity'))::int,
		       COALESCE(c.last_message_at, c.created_at) AS sort_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
		LEFT JOIN LATERAL (
			SELECT id, sender_id, body, created_at FROM messages
			WHERE conversation_id = c.id AND created_at > COALESCE(me.cleared_at, '-infinity')
			ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND (me.cleared_at IS NULL OR COALESCE(c.last_message_at, c.created_at) > me.cleared_at)
		  AND ($2::uuid IS NULL OR c.id = $2)
		  AND ($3::timestamptz IS NULL OR (COALESCE(c.last_message_at, c.created_at), c.id) < ($3, $4::uuid))
		ORDER BY sort_at DESC, c.id DESC
		LIMIT $5
	`, userID, only, cursorAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ConversationRow, error) {
		var c ConversationRow
		var lmID, lmSender, lmBody *string
		var lmAt *time.Time
		err := row.Scan(&c.ID, &c.MatchID, &c.MatchedAt, &c.OtherUserID, &c.OtherLastReadAt, &lmID, &lmSender, &lmBody, &lmAt, &c.UnreadCount, &c.SortAt)
		if lmID != nil {
			c.LastMessage = &Message{ID: *lmID, ConversationID: c.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		return c, err
	})
}

// ListMessages returns messages newest first (keyset pagination going back in
// time), hiding anything the user cleared.
func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, after *time.Time, cursor *httpx.Cursor, limit int) ([]Message, error) {
	var cursorAt *time.Time
	var cursorID *string
	if cursor != nil {
		cursorAt, cursorID = &cursor.At, &cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5
	`, conversationID, after, cursorAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
		return m, err
	})
}

func (r *PGRepository) CreateMessage(ctx context.Context, userID, conversationID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
		RETURNING id, conversation_id, sender_id, body, created_at
	`, conversationID, userID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrConversationNotFound
	}
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrConversationNotFound
	}
	return at, err
}

func (r *PGRepository) Clear(ctx context.Context, userID, conversationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = clock_timestamp(), last_read_at = clock_timestamp()
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
