package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepoNotMember = errors.New("not a conversation participant")

type Repository interface {
	Membership(ctx context.Context, conversationID, userID string) (membership, error)
	ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]conversationRow, error)
	ListMessages(ctx context.Context, conversationID string, cursor *httpx.Cursor, limit int) ([]Message, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	UnreadConversations(ctx context.Context, userID string) (int, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// Membership is the authorisation gate of every chat operation: it only returns a row when
// userID is a participant of the conversation and no block exists between the two members.
func (r *PGRepository) Membership(ctx context.Context, conversationID, userID string) (membership, error) {
	var m membership
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.match_id, other.user_id, other.last_read_at, c.created_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = me.user_id AND b.blocked_id = other.user_id)
			   OR (b.blocker_id = other.user_id AND b.blocked_id = me.user_id)
		  )
	`, conversationID, userID).Scan(&m.ConversationID, &m.MatchID, &m.OtherUserID, &m.OtherLastReadAt, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return membership{}, errRepoNotMember
	}
	return m, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]conversationRow, error) {
	args := []any{userID, limit}
	cond := ""
	if cursor != nil {
		cond = "AND (c.last_message_at, c.id) < ($3, $4::uuid)"
		args = append(args, cursor.Time, cursor.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, other.user_id, other.last_read_at, c.created_at, c.last_message_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM (
		          SELECT 1 FROM messages m
		          WHERE m.conversation_id = c.id AND m.sender_id <> $1
		            AND (me.last_read_at IS NULL OR m.created_at > me.last_read_at)
		          LIMIT 99
		       ) unread)::int
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
		LEFT JOIN LATERAL (
			SELECT id, sender_id, body, created_at FROM messages
			WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1 AND c.last_message_at IS NOT NULL `+cond+`
		ORDER BY c.last_message_at DESC, c.id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conversationRow
	for rows.Next() {
		var row conversationRow
		var lmID, lmSender, lmBody *string
		var lmAt *time.Time
		if err := rows.Scan(&row.ConversationID, &row.MatchID, &row.OtherUserID, &row.OtherLastReadAt, &row.CreatedAt,
			&row.LastActivityAt, &lmID, &lmSender, &lmBody, &lmAt, &row.UnreadCount); err != nil {
			return nil, err
		}
		if lmID != nil {
			row.LastMessage = &Message{ID: *lmID, ConversationID: row.ConversationID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, cursor *httpx.Cursor, limit int) ([]Message, error) {
	args := []any{conversationID, limit}
	cond := ""
	if cursor != nil {
		cond = "AND (created_at, id) < ($3, $4::uuid)"
		args = append(args, cursor.Time, cursor.ID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1 `+cond+`
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
		return m, err
	})
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	var m Message
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
			RETURNING id, conversation_id, sender_id, body, created_at
		`, conversationID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
		`, conversationID, senderID, m.CreatedAt)
		return err
	})
	return m, err
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

func (r *PGRepository) UnreadConversations(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		WHERE me.user_id = $1 AND c.last_message_at IS NOT NULL
		  AND EXISTS (
			SELECT 1 FROM messages m
			WHERE m.conversation_id = c.id AND m.sender_id <> $1
			  AND (me.last_read_at IS NULL OR m.created_at > me.last_read_at)
		  )
	`, userID).Scan(&n)
	return n, err
}
