package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Conversation, error)
	GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID string, limit int, beforeAt *time.Time, beforeID string) ([]Message, time.Time, error)
	SendMessage(ctx context.Context, userID, conversationID, body string) (Sent, error)
	MarkRead(ctx context.Context, userID, conversationID string) (peerID string, readAt time.Time, err error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

const conversationSelect = `
	SELECT c.id, c.match_id, c.created_at, COALESCE(c.last_message_at, c.created_at),
	       ` + profiles.CardColumns + `,
	       lm.id, lm.sender_id, lm.body, lm.created_at,
	       (SELECT count(*) FROM messages m
	         WHERE m.conversation_id = c.id AND m.sender_id <> $1 AND m.created_at > cm.last_read_at)
	FROM conversation_members cm
	JOIN conversations c ON c.id = cm.conversation_id
	JOIN conversation_members pm ON pm.conversation_id = c.id AND pm.user_id <> $1
	JOIN profiles p ON p.user_id = pm.user_id
	LEFT JOIN LATERAL (
		SELECT id, sender_id, body, created_at FROM messages
		WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1) lm ON TRUE
	WHERE cm.user_id = $1`

func scanConversation(row profiles.Scanner) (Conversation, error) {
	var c Conversation
	var lmID, lmSender, lmBody *string
	var lmAt *time.Time
	p, err := profiles.ScanCardWithPrefix(row,
		[]any{&c.ID, &c.MatchID, &c.CreatedAt, &c.UpdatedAt},
		&lmID, &lmSender, &lmBody, &lmAt, &c.UnreadCount)
	if err != nil {
		return Conversation{}, err
	}
	c.Profile = p
	if lmID != nil {
		c.LastMessage = &Message{ID: *lmID, ConversationID: c.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
	}
	return c, nil
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, conversationSelect+`
		AND cm.hidden_at IS NULL
		AND ($2::timestamptz IS NULL OR (COALESCE(c.last_message_at, c.created_at), c.id) < ($2, $3::uuid))
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC
		LIMIT $4`, userID, beforeAt, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error) {
	c, err := scanConversation(r.pool.QueryRow(ctx, conversationSelect+` AND c.id = $2`, userID, conversationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, err
	}
	return c, nil
}

// ListMessages returns a page of messages (newest first) after verifying the
// caller is a member; non-members get the same not-found as a missing id.
func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, limit int, beforeAt *time.Time, beforeID string) ([]Message, time.Time, error) {
	var peerRead time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT pm.last_read_at
		FROM conversation_members cm
		JOIN conversation_members pm ON pm.conversation_id = cm.conversation_id AND pm.user_id <> cm.user_id
		WHERE cm.conversation_id = $1 AND cm.user_id = $2`, conversationID, userID).Scan(&peerRead)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return nil, time.Time{}, ErrConversationNotFound
		}
		return nil, time.Time{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4`, conversationID, beforeAt, nullUUID(beforeID), limit)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, time.Time{}, err
		}
		out = append(out, m)
	}
	return out, peerRead, rows.Err()
}

func (r *PGRepository) SendMessage(ctx context.Context, userID, conversationID, body string) (Sent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Sent{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var peerID string
	err = tx.QueryRow(ctx, `
		SELECT pm.user_id
		FROM conversation_members cm
		JOIN conversation_members pm ON pm.conversation_id = cm.conversation_id AND pm.user_id <> cm.user_id
		WHERE cm.conversation_id = $1 AND cm.user_id = $2`, conversationID, userID).Scan(&peerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Sent{}, ErrConversationNotFound
		}
		return Sent{}, err
	}

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at`, conversationID, userID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if err != nil {
		return Sent{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Sent{}, err
	}
	// Sending implies the sender has read everything so far; a new message also
	// resurfaces a conversation that either side had hidden.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_members
		SET hidden_at = NULL,
		    last_read_at = CASE WHEN user_id = $2 THEN $3 ELSE last_read_at END
		WHERE conversation_id = $1`, conversationID, userID, m.CreatedAt); err != nil {
		return Sent{}, err
	}
	added, err := notifications.InsertTx(ctx, tx, peerID, notifications.TypeNewMessage, userID, conversationID)
	if err != nil {
		return Sent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Sent{}, err
	}
	return Sent{Message: m, PeerID: peerID, NotificationAdded: added}, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (string, time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var readAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE conversation_members SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at`, conversationID, userID).Scan(&readAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", time.Time{}, ErrConversationNotFound
		}
		return "", time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = $3
		WHERE user_id = $1 AND conversation_id = $2 AND type = 'new_message' AND read_at IS NULL`,
		userID, conversationID, readAt); err != nil {
		return "", time.Time{}, err
	}
	var peerID string
	if err := tx.QueryRow(ctx, `
		SELECT user_id FROM conversation_members WHERE conversation_id = $1 AND user_id <> $2`,
		conversationID, userID).Scan(&peerID); err != nil {
		return "", time.Time{}, err
	}
	return peerID, readAt, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_members SET hidden_at = NOW(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrConversationNotFound
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConversationNotFound
	}
	return nil
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
