package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrConversationNotFound is returned when there is no active match between
// the two members (never matched, unmatched, blocked, other tenant or the
// account no longer exists). The same error is used in every case so the API
// does not leak whether an account exists.
var ErrConversationNotFound = errors.New("conversation not found")

type Repository interface {
	ListChats(ctx context.Context, userID string, limit, offset int) ([]ChatRow, error)
	GetConversation(ctx context.Context, userID, otherUserID string) (Conversation, error)
	ListMessages(ctx context.Context, conv Conversation, beforeID *string, limit int) ([]ChatMessage, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (ChatMessage, error)
	IsSuspended(ctx context.Context, userID string) (bool, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// ListChats returns one row per active match of userID, newest activity
// first, with the last visible message and the unread count. The per-row
// lateral lookups use the (conversation_id, created_at) index so the cost is
// O(page size).
func (r *PGRepository) ListChats(ctx context.Context, userID string, limit, offset int) ([]ChatRow, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT m.id,
		       CASE WHEN m.user_low_id = $1 THEN m.user_high_id ELSE m.user_low_id END AS other_id,
		       m.created_at,
		       lm.id, lm.sender_user_id, lm.recipient_user_id, lm.content, lm.created_at,
		       COALESCE(unread.count, 0)
		FROM matches m
		LEFT JOIN conversations c ON c.match_id = m.id
		LEFT JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
		LEFT JOIN LATERAL (
			SELECT msg.id, msg.sender_user_id, msg.recipient_user_id, msg.content, msg.created_at
			FROM messages msg
			WHERE msg.conversation_id = c.id
			  AND (me.cleared_at IS NULL OR msg.created_at > me.cleared_at)
			ORDER BY msg.created_at DESC, msg.id DESC
			LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS count FROM (
				SELECT 1 FROM messages msg
				WHERE msg.conversation_id = c.id
				  AND msg.sender_user_id <> $1
				  AND msg.created_at > COALESCE(me.last_read_at, '-infinity'::timestamptz)
				  AND (me.cleared_at IS NULL OR msg.created_at > me.cleared_at)
				LIMIT 100
			) capped
		) unread ON TRUE
		WHERE (m.user_low_id = $1 OR m.user_high_id = $1)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = m.user_low_id AND b.blocked_user_id = m.user_high_id)
			   OR (b.blocker_user_id = m.user_high_id AND b.blocked_user_id = m.user_low_id)
		  )
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ChatRow{}
	for rows.Next() {
		var row ChatRow
		var msgID, sender, recipient, content *string
		var createdAt *time.Time
		if err := rows.Scan(&row.MatchID, &row.OtherUserID, &row.MatchedAt, &msgID, &sender, &recipient, &content, &createdAt, &row.UnreadCount); err != nil {
			return nil, err
		}
		if msgID != nil && sender != nil && content != nil && createdAt != nil {
			msg := ChatMessage{ID: *msgID, SenderUserID: *sender, Content: *content, CreatedAt: *createdAt}
			if recipient != nil {
				msg.RecipientUserID = *recipient
			}
			row.LastMessage = &msg
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetConversation returns the conversation of an active match, creating it
// if a legacy match has none yet.
func (r *PGRepository) GetConversation(ctx context.Context, userID, otherUserID string) (Conversation, error) {
	if userID == otherUserID {
		return Conversation{}, ErrConversationNotFound
	}
	low, high := userID, otherUserID
	if high < low {
		low, high = high, low
	}

	var matchID string
	err := r.dbPool.QueryRow(ctx, `
		SELECT m.id FROM matches m
		WHERE m.user_low_id = $1 AND m.user_high_id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
			   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
		  )
	`, low, high).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, ErrConversationNotFound
	}
	if err != nil {
		return Conversation{}, err
	}

	var conv Conversation
	conv.MatchID = matchID
	err = r.dbPool.QueryRow(ctx, `SELECT id FROM conversations WHERE match_id = $1`, matchID).Scan(&conv.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := r.dbPool.QueryRow(ctx, `
			INSERT INTO conversations (kind, direct_key, match_id)
			VALUES ('direct', $1 || ':' || $2, $3)
			ON CONFLICT (direct_key) DO UPDATE SET match_id = EXCLUDED.match_id
			RETURNING id
		`, low, high, matchID).Scan(&conv.ID); err != nil {
			return Conversation{}, err
		}
		if _, err := r.dbPool.Exec(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id)
			VALUES ($1, $2), ($1, $3)
			ON CONFLICT (conversation_id, user_id) DO NOTHING
		`, conv.ID, low, high); err != nil {
			return Conversation{}, err
		}
	} else if err != nil {
		return Conversation{}, err
	}

	err = r.dbPool.QueryRow(ctx, `
		SELECT
			(SELECT cleared_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2),
			(SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $3)
	`, conv.ID, userID, otherUserID).Scan(&conv.MyClearedAt, &conv.OtherLastReadAt)
	if err != nil {
		return Conversation{}, err
	}
	return conv, nil
}

func (r *PGRepository) ListMessages(ctx context.Context, conv Conversation, beforeID *string, limit int) ([]ChatMessage, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT m.id, m.sender_user_id, COALESCE(m.recipient_user_id::text, ''), m.content, m.created_at
		FROM messages m
		WHERE m.conversation_id = $1
		  AND ($2::timestamptz IS NULL OR m.created_at > $2)
		  AND (
			$3::uuid IS NULL
			OR (m.created_at, m.id) < (SELECT b.created_at, b.id FROM messages b WHERE b.id = $3 AND b.conversation_id = $1)
		  )
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4
	`, conv.ID, conv.MyClearedAt, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Return chronological order (oldest first) for display.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (ChatMessage, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return ChatMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m ChatMessage
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_user_id, recipient_user_id, content, created_at
	`, conversationID, senderID, recipientID, content).Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// The match (and its conversation) was removed concurrently.
			return ChatMessage{}, ErrConversationNotFound
		}
		return ChatMessage{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return ChatMessage{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return ChatMessage{}, err
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
		return time.Time{}, ErrConversationNotFound
	}
	return readAt, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}

func (r *PGRepository) IsSuspended(ctx context.Context, userID string) (bool, error) {
	var suspended bool
	err := r.dbPool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1 AND is_suspended)`, userID).Scan(&suspended)
	return suspended, err
}
