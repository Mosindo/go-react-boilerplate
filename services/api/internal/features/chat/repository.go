package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]convRow, error)
	UnreadTotal(ctx context.Context, userID string) (int, error)
	Access(ctx context.Context, userID, conversationID string) (access, error)
	ListMessages(ctx context.Context, conversationID string, clearedAt *time.Time, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, senderID, content string) (Message, error)
	MarkRead(ctx context.Context, conversationID, readerID string) (int, time.Time, error)
	Clear(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]convRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, m.id,
		       CASE WHEN m.user_low = $1 THEN m.user_high ELSE m.user_low END,
		       lm.id, lm.sender_user_id, lm.content, lm.created_at, lm.read_at,
		       COALESCE(un.cnt, 0),
		       COALESCE(lm.created_at, m.created_at)
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		LEFT JOIN LATERAL (
			SELECT id, sender_user_id, content, created_at, read_at FROM messages
			WHERE conversation_id = c.id AND (cp.hidden_at IS NULL OR created_at > cp.hidden_at)
			ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM messages
			WHERE conversation_id = c.id AND sender_user_id <> $1 AND read_at IS NULL
			  AND (cp.hidden_at IS NULL OR created_at > cp.hidden_at)
		) un ON TRUE
		WHERE cp.user_id = $1
		  AND (cp.hidden_at IS NULL OR lm.id IS NOT NULL)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = CASE WHEN m.user_low = $1 THEN m.user_high ELSE m.user_low END)
			   OR (b.blocked_id = $1 AND b.blocker_id = CASE WHEN m.user_low = $1 THEN m.user_high ELSE m.user_low END))
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, c.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []convRow{}
	for rows.Next() {
		var row convRow
		var lmID, lmSender, lmContent *string
		var lmAt, lmRead *time.Time
		if err := rows.Scan(&row.ID, &row.MatchID, &row.OtherUserID, &lmID, &lmSender, &lmContent, &lmAt, &lmRead, &row.UnreadCount, &row.UpdatedAt); err != nil {
			return nil, err
		}
		if lmID != nil {
			row.Last = &Message{ID: *lmID, ConversationID: row.ID, SenderID: *lmSender, Content: *lmContent, CreatedAt: *lmAt, ReadAt: lmRead}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN messages msg ON msg.conversation_id = c.id AND msg.sender_user_id <> $1 AND msg.read_at IS NULL
		WHERE cp.user_id = $1 AND (cp.hidden_at IS NULL OR msg.created_at > cp.hidden_at)
	`, userID).Scan(&n)
	return n, err
}

// Access authorizes a caller on a conversation: they must be a participant of an active
// match and not blocked either way. Non-members get ErrNotFound, never a hint that it exists.
func (r *PGRepository) Access(ctx context.Context, userID, conversationID string) (access, error) {
	var a access
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN m.user_low = $1 THEN m.user_high ELSE m.user_low END, cp.hidden_at
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		WHERE cp.conversation_id = $2 AND cp.user_id = $1
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = m.user_low AND b.blocked_id = m.user_high)
			   OR (b.blocker_id = m.user_high AND b.blocked_id = m.user_low))
	`, userID, conversationID).Scan(&a.OtherUserID, &a.ClearedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return access{}, ErrNotFound
	}
	return a, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, clearedAt *time.Time, beforeID string, limit int) ([]Message, error) {
	var before any
	if beforeID != "" {
		before = beforeID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, sender_user_id, content, created_at, read_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::uuid IS NULL OR (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $3 AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, clearedAt, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m := Message{ConversationID: conversationID}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, content string) (Message, error) {
	m := Message{ConversationID: conversationID}
	err := r.db.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, content) VALUES ($1, $2, $3)
		RETURNING id, sender_user_id, content, created_at, read_at
	`, conversationID, senderID, content).Scan(&m.ID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt)
	if err != nil {
		return Message{}, err
	}
	_, _ = r.db.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID)
	return m, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, readerID string) (int, time.Time, error) {
	var count int
	var at time.Time
	err := r.db.QueryRow(ctx, `
		WITH updated AS (
			UPDATE messages SET read_at = NOW()
			WHERE conversation_id = $1 AND sender_user_id <> $2 AND read_at IS NULL
			RETURNING read_at
		)
		SELECT COUNT(*), COALESCE(MAX(read_at), NOW()) FROM updated
	`, conversationID, readerID).Scan(&count, &at)
	return count, at, err
}

// Clear implements local deletion: the conversation (and its past messages) disappears
// for this participant only, and reappears only when a newer message arrives.
func (r *PGRepository) Clear(ctx context.Context, userID, conversationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
