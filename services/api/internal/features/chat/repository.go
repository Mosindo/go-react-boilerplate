package chat

import (
	"context"
	"errors"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const listConversationsSQL = `
SELECT c.id, m.id, m.created_at,
       COALESCE(c.last_message_at, m.created_at) AS updated_at,
       o.user_id,
       COALESCE(p.first_name, ''),
       CASE WHEN p.show_age AND u.birth_date IS NOT NULL
            THEN EXTRACT(YEAR FROM age(CURRENT_DATE, u.birth_date))::int END,
       ph.id, COALESCE(ph.position, 0),
       COALESCE(unread.n, 0),
       lm.id, lm.sender_user_id, lm.content, lm.created_at,
       CASE WHEN lm.sender_user_id = $1 AND o.last_read_at >= lm.created_at THEN o.last_read_at END
FROM conversation_participants me
JOIN conversations c ON c.id = me.conversation_id AND c.match_id IS NOT NULL
JOIN matches m ON m.id = c.match_id
JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
JOIN users u ON u.id = o.user_id
LEFT JOIN profiles p ON p.user_id = o.user_id
LEFT JOIN LATERAL (
    SELECT id, sender_user_id, content, created_at
    FROM messages WHERE conversation_id = c.id
    ORDER BY created_at DESC, id DESC LIMIT 1
) lm ON TRUE
LEFT JOIN LATERAL (
    SELECT COUNT(*)::int AS n FROM messages x
    WHERE x.conversation_id = c.id AND x.sender_user_id = o.user_id
      AND (me.last_read_at IS NULL OR x.created_at > me.last_read_at)
) unread ON TRUE
LEFT JOIN LATERAL (
    SELECT id, position FROM photos WHERE user_id = o.user_id AND position = 0 LIMIT 1
) ph ON TRUE
WHERE me.user_id = $1
  AND (me.hidden_at IS NULL OR EXISTS (
        SELECT 1 FROM messages h
        WHERE h.conversation_id = c.id AND h.sender_user_id = o.user_id AND h.created_at > me.hidden_at))
  AND NOT EXISTS (
        SELECT 1 FROM blocks b
        WHERE (b.blocker_id = $1 AND b.blocked_id = o.user_id)
           OR (b.blocker_id = o.user_id AND b.blocked_id = $1))
  AND ($2::timestamptz IS NULL
       OR COALESCE(c.last_message_at, m.created_at) < $2
       OR (COALESCE(c.last_message_at, m.created_at) = $2 AND ($3::uuid IS NULL OR c.id < $3)))
ORDER BY COALESCE(c.last_message_at, m.created_at) DESC, c.id DESC
LIMIT $4`

// ListConversations returns one page of my visible conversations in a single set-based query.
func (r *Repository) ListConversations(ctx context.Context, userID string, before *time.Time, beforeID *string, limit int) ([]convoRow, error) {
	rows, err := r.pool.Query(ctx, listConversationsSQL, userID, before, beforeID, limit)
	if err != nil {
		return nil, platformerrors.Wrap("chat.list_conversations", err)
	}
	defer rows.Close()
	out := make([]convoRow, 0, limit)
	for rows.Next() {
		var (
			cr                   convoRow
			lmID, lmSender, lmBd *string
			lmAt, lmRead         *time.Time
		)
		if err := rows.Scan(&cr.ID, &cr.MatchID, &cr.MatchedAt, &cr.UpdatedAt, &cr.OtherID, &cr.FirstName,
			&cr.Age, &cr.PhotoID, &cr.PhotoPosition, &cr.UnreadCount, &lmID, &lmSender, &lmBd, &lmAt, &lmRead); err != nil {
			return nil, platformerrors.Wrap("chat.list_conversations.scan", err)
		}
		if lmID != nil {
			cr.LastMessage = &Message{ID: *lmID, ConversationID: cr.ID, SenderID: *lmSender, Body: *lmBd, CreatedAt: *lmAt, ReadAt: lmRead}
		}
		out = append(out, cr)
	}
	return out, platformerrors.Wrap("chat.list_conversations.rows", rows.Err())
}

// OtherParticipant proves that userID participates in a match conversation and returns the other user.
// It returns ErrNotFound otherwise (callers must map that to 404 so existence never leaks).
func (r *Repository) OtherParticipant(ctx context.Context, conversationID, userID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		SELECT o.user_id
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id AND c.match_id IS NOT NULL
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return other, platformerrors.Wrap("chat.other_participant", err)
}

// ListMessages returns up to limit+1 messages newest first, older than the cursor message.
func (r *Repository) ListMessages(ctx context.Context, conversationID, userID, otherID string, beforeID *string, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.sender_user_id, m.content, m.created_at,
		       CASE WHEN m.sender_user_id = $2 AND o.last_read_at >= m.created_at THEN o.last_read_at END
		FROM messages m
		JOIN conversation_participants o ON o.conversation_id = m.conversation_id AND o.user_id = $3
		WHERE m.conversation_id = $1
		  AND ($4::uuid IS NULL OR (m.created_at, m.id) < (
		        SELECT c.created_at, c.id FROM messages c WHERE c.id = $4 AND c.conversation_id = $1))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $5`, conversationID, userID, otherID, beforeID, limit+1)
	if err != nil {
		return nil, platformerrors.Wrap("chat.list_messages", err)
	}
	defer rows.Close()
	out := make([]Message, 0, limit+1)
	for rows.Next() {
		m := Message{ConversationID: conversationID}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, platformerrors.Wrap("chat.list_messages.scan", err)
		}
		out = append(out, m)
	}
	return out, platformerrors.Wrap("chat.list_messages.rows", rows.Err())
}

// SendMessage runs the whole send in one transaction. The conversation row is locked so that
// concurrent sends are strictly serialised and get strictly increasing created_at values.
func (r *Repository) SendMessage(ctx context.Context, conversationID, senderID, body string) (*sendResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, platformerrors.Wrap("chat.send.begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		recipient string
		firstName string
		blocked   bool
	)
	err = tx.QueryRow(ctx, `
		SELECT o.user_id, COALESCE(sp.first_name, ''),
		       EXISTS (SELECT 1 FROM blocks b
		               WHERE (b.blocker_id = me.user_id AND b.blocked_id = o.user_id)
		                  OR (b.blocker_id = o.user_id AND b.blocked_id = me.user_id))
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id AND c.match_id IS NOT NULL
		JOIN matches m ON m.id = c.match_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
		LEFT JOIN profiles sp ON sp.user_id = me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		FOR UPDATE OF c`, conversationID, senderID).Scan(&recipient, &firstName, &blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, platformerrors.Wrap("chat.send.lock", err)
	}
	if blocked {
		return nil, ErrBlocked
	}

	msg := Message{ConversationID: conversationID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, clock_timestamp(), clock_timestamp())
		RETURNING id, created_at`, conversationID, senderID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return nil, platformerrors.Wrap("chat.send.insert", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2, updated_at = $2 WHERE id = $1`,
		conversationID, msg.CreatedAt); err != nil {
		return nil, platformerrors.Wrap("chat.send.touch", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants
		SET hidden_at = CASE WHEN user_id = $2 THEN NULL ELSE hidden_at END,
		    last_read_at = CASE WHEN user_id = $3 THEN GREATEST(COALESCE(last_read_at, $4), $4) ELSE last_read_at END
		WHERE conversation_id = $1 AND user_id IN ($2, $3)`,
		conversationID, recipient, senderID, msg.CreatedAt); err != nil {
		return nil, platformerrors.Wrap("chat.send.participants", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, platformerrors.Wrap("chat.send.commit", err)
	}
	return &sendResult{Message: msg, RecipientID: recipient, SenderFirstName: firstName}, nil
}

// MarkRead advances my last_read_at (never backwards) and returns the resulting value.
func (r *Repository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var readAt time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, clock_timestamp()), clock_timestamp())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return readAt, platformerrors.Wrap("chat.mark_read", err)
}

// Hide sets hidden_at for me only.
func (r *Repository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	return platformerrors.Wrap("chat.hide", err)
}
