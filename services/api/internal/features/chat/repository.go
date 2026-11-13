package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

type Repository interface {
	CanChat(ctx context.Context, userID, otherUserID string) (bool, error)
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]storedConversation, error)
	ListMessages(ctx context.Context, userID, otherUserID, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, userID, otherUserID, content string) (Message, error)
	MarkRead(ctx context.Context, userID, otherUserID string) (time.Time, error)
	Clear(ctx context.Context, userID, otherUserID string) error
	PeerLastReadAt(ctx context.Context, userID, otherUserID string) (*time.Time, error)
	FirstName(ctx context.Context, userID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func directConversationKey(userID, otherUserID string) string {
	if userID < otherUserID {
		return userID + ":" + otherUserID
	}
	return otherUserID + ":" + userID
}

// CanChat is the single authorization rule for messaging: an active match
// and no block in either direction.
func (r *PGRepository) CanChat(ctx context.Context, userID, otherUserID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM matches m
			WHERE m.user_a_id = LEAST($1::uuid, $2::uuid)
			  AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
			  AND m.unmatched_at IS NULL
		) AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
			   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
		)
	`, userID, otherUserID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.created_at, o.peer_id,
		       lm.id, lm.sender_user_id, lm.recipient_user_id, lm.content, lm.created_at,
		       COALESCE(unread.n, 0)
		FROM matches m
		CROSS JOIN LATERAL (
			SELECT CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END AS peer_id
		) o
		LEFT JOIN conversations c
		  ON c.direct_key = LEAST($1::text, o.peer_id::text) || ':' || GREATEST($1::text, o.peer_id::text)
		LEFT JOIN conversation_participants me
		  ON me.conversation_id = c.id AND me.user_id = $1
		LEFT JOIN LATERAL (
			SELECT msg.id, msg.sender_user_id, msg.recipient_user_id, msg.content, msg.created_at
			FROM messages msg
			WHERE msg.conversation_id = c.id
			  AND msg.created_at > COALESCE(me.cleared_at, '-infinity'::timestamptz)
			ORDER BY msg.created_at DESC, msg.id DESC
			LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS n FROM (
				SELECT 1 FROM messages msg
				WHERE msg.conversation_id = c.id
				  AND msg.sender_user_id <> $1
				  AND msg.created_at > GREATEST(
					COALESCE(me.last_read_at, '-infinity'::timestamptz),
					COALESCE(me.cleared_at, '-infinity'::timestamptz))
				LIMIT 99
			) capped
		) unread ON TRUE
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1)
		  AND m.unmatched_at IS NULL
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = o.peer_id)
			   OR (b.blocker_user_id = o.peer_id AND b.blocked_user_id = $1)
		  )
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]storedConversation, 0, limit)
	for rows.Next() {
		var conv storedConversation
		var msgID, sender, recipient, content *string
		var createdAt *time.Time
		if err := rows.Scan(&conv.MatchID, &conv.MatchedAt, &conv.PeerID,
			&msgID, &sender, &recipient, &content, &createdAt, &conv.UnreadCount); err != nil {
			return nil, err
		}
		if msgID != nil && sender != nil && content != nil && createdAt != nil {
			msg := Message{ID: *msgID, SenderUserID: *sender, Content: *content, CreatedAt: *createdAt}
			if recipient != nil {
				msg.RecipientUserID = *recipient
			}
			conv.LastMessage = &msg
		}
		out = append(out, conv)
	}
	return out, rows.Err()
}

// ListMessages returns up to limit messages older than beforeID (or the
// newest ones), newest first, hiding what the viewer cleared.
func (r *PGRepository) ListMessages(ctx context.Context, userID, otherUserID, beforeID string, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		WITH conv AS (
			SELECT c.id, COALESCE(cp.cleared_at, '-infinity'::timestamptz) AS cleared_at
			FROM conversations c
			LEFT JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $1
			WHERE c.direct_key = $2
		),
		cursor AS (
			SELECT created_at, id FROM messages
			WHERE id = NULLIF($3, '')::uuid AND conversation_id = (SELECT id FROM conv)
		)
		SELECT m.id, m.sender_user_id, COALESCE(m.recipient_user_id::text, ''), m.content, m.created_at
		FROM messages m, conv
		WHERE m.conversation_id = conv.id
		  AND m.created_at > conv.cleared_at
		  AND (
			NULLIF($3, '') IS NULL
			OR (m.created_at, m.id) < (SELECT created_at, id FROM cursor)
		  )
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4
	`, userID, directConversationKey(userID, otherUserID), beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func ensureConversation(ctx context.Context, tx pgx.Tx, userID, otherUserID string) (string, error) {
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

func (r *PGRepository) CreateMessage(ctx context.Context, userID, otherUserID, content string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	conversationID, err := ensureConversation(ctx, tx, userID, otherUserID)
	if err != nil {
		return Message{}, err
	}

	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_user_id, recipient_user_id, content, created_at
	`, conversationID, userID, otherUserID, content).Scan(&m.ID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, otherUserID string) (time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	conversationID, err := ensureConversation(ctx, tx, userID, otherUserID)
	if err != nil {
		return time.Time{}, err
	}
	var readAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt); err != nil {
		return time.Time{}, err
	}
	return readAt, tx.Commit(ctx)
}

func (r *PGRepository) Clear(ctx context.Context, userID, otherUserID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	conversationID, err := ensureConversation(ctx, tx, userID, otherUserID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) PeerLastReadAt(ctx context.Context, userID, otherUserID string) (*time.Time, error) {
	var readAt *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT cp.last_read_at
		FROM conversations c
		JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $1
		WHERE c.direct_key = $2
	`, otherUserID, directConversationKey(userID, otherUserID)).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return readAt, err
}

func (r *PGRepository) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}
