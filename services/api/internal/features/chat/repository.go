package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ConversationForPair(ctx context.Context, organizationID, userID, otherUserID string) (conversationRef, error)
	ListChats(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error)
	ListMessages(ctx context.Context, conversationID string, beforeID string, limit int) ([]Message, bool, error)
	OtherLastReadAt(ctx context.Context, conversationID, userID string) (*time.Time, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Hide(ctx context.Context, conversationID, userID string) error
	UnreadConversations(ctx context.Context, userID string) (int, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ConversationForPair is the chat authorization gate: a conversation is only
// reachable through an existing match in the caller's tenant (blocks and
// unmatches delete the match, which closes the conversation).
func (r *PGRepository) ConversationForPair(ctx context.Context, organizationID, userID, otherUserID string) (conversationRef, error) {
	var ref conversationRef
	err := r.db.QueryRow(ctx, `
		SELECT m.conversation_id, m.id
		FROM matches m
		JOIN users other ON other.id = $3 AND other.organization_id = $1
		JOIN conversation_participants cp ON cp.conversation_id = m.conversation_id AND cp.user_id = $2
		WHERE m.user_low_id = LEAST($2::uuid, $3::uuid)
		  AND m.user_high_id = GREATEST($2::uuid, $3::uuid)
		  AND m.conversation_id IS NOT NULL
	`, organizationID, userID, otherUserID).Scan(&ref.ConversationID, &ref.MatchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return conversationRef{}, ErrNotMatched
	}
	return ref, err
}

func (r *PGRepository) ListChats(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.conversation_id, other.user_id, m.created_at,
		       lm.id, lm.sender_user_id, lm.content, lm.created_at,
		       (
				SELECT COUNT(*) FROM (
					SELECT 1 FROM messages x
					WHERE x.conversation_id = m.conversation_id
					  AND x.sender_user_id <> $1
					  AND x.created_at > COALESCE(me.last_read_at, 'epoch'::timestamptz)
					LIMIT 99
				) capped
		       ) AS unread
		FROM matches m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $1
		JOIN conversation_participants other ON other.conversation_id = m.conversation_id AND other.user_id <> $1
		JOIN LATERAL (
			SELECT id, sender_user_id, content, created_at
			FROM messages
			WHERE conversation_id = m.conversation_id
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE (m.user_low_id = $1 OR m.user_high_id = $1)
		  AND (me.hidden_at IS NULL OR lm.created_at > me.hidden_at)
		ORDER BY lm.created_at DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedSummary, error) {
		var s storedSummary
		var lm MessagePreview
		err := row.Scan(&s.MatchID, &s.ConversationID, &s.OtherUserID, &s.MatchedAt, &lm.ID, &lm.SenderUserID, &lm.Content, &lm.CreatedAt, &s.UnreadCount)
		s.LastMessage = &lm
		return s, err
	})
}

// ListMessages pages backwards with a keyset cursor (the oldest message id the
// client already has) and returns the page in chronological order.
func (r *PGRepository) ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, bool, error) {
	var rows pgx.Rows
	var err error
	if beforeID == "" {
		rows, err = r.db.Query(ctx, `
			SELECT id, conversation_id, sender_user_id, recipient_user_id, content, created_at
			FROM messages WHERE conversation_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`, conversationID, limit+1)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT m.id, m.conversation_id, m.sender_user_id, m.recipient_user_id, m.content, m.created_at
			FROM messages m
			JOIN messages cursor ON cursor.id = $2 AND cursor.conversation_id = $1
			WHERE m.conversation_id = $1
			  AND (m.created_at, m.id) < (cursor.created_at, cursor.id)
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT $3
		`, conversationID, beforeID, limit+1)
	}
	if err != nil {
		return nil, false, err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt)
		return m, err
	})
	if err != nil {
		return nil, false, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, hasMore, nil
}

func (r *PGRepository) OtherLastReadAt(ctx context.Context, conversationID, userID string) (*time.Time, error) {
	var t *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT last_read_at FROM conversation_participants
		WHERE conversation_id = $1 AND user_id <> $2
		LIMIT 1
	`, conversationID, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, sender_user_id, recipient_user_id, content, created_at
	`, conversationID, senderID, recipientID, content).Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before it.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&t)
	return t, err
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}

func (r *PGRepository) UnreadConversations(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM matches m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $1
		WHERE (m.user_low_id = $1 OR m.user_high_id = $1)
		  AND EXISTS (
			SELECT 1 FROM messages x
			WHERE x.conversation_id = m.conversation_id
			  AND x.sender_user_id <> $1
			  AND x.created_at > COALESCE(me.last_read_at, 'epoch'::timestamptz)
		  )
	`, userID).Scan(&n)
	return n, err
}
