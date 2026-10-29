package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotParticipant = errors.New("not a participant of this conversation")

type SendResult struct {
	Message      Message
	RecipientID  string
	Notification *notifications.Notification
}

type ReadResult struct {
	ReadCount int
	OtherID   string
	ReadAt    time.Time
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID string, before *int64, limit int) ([]Message, error)
	Send(ctx context.Context, userID, conversationID, body string) (SendResult, error)
	MarkRead(ctx context.Context, userID, conversationID string) (ReadResult, error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// One query returns the page of conversations with the other user's card, the
// last visible message and the unread count: no per-conversation round trips.
const listConversationsQuery = `
SELECT c.id::text, m.id::text, m.created_at,
       o.id::text, pr.first_name, ph.id::text,
       COALESCE(lm.created_at, c.created_at) AS sort_at,
       lm.id, lm.sender_id::text, lm.body, lm.created_at, lm.read_at,
       (SELECT COUNT(*) FROM messages um
         WHERE um.conversation_id = c.id AND um.sender_id <> $1 AND um.read_at IS NULL
           AND um.created_at > COALESCE(cp.cleared_at, '-infinity'))
FROM conversation_participants cp
JOIN conversations c ON c.id = cp.conversation_id
JOIN matches m ON m.id = c.match_id
CROSS JOIN LATERAL (SELECT CASE WHEN m.user_low_id = $1 THEN m.user_high_id ELSE m.user_low_id END AS id) o
JOIN profiles pr ON pr.user_id = o.id
LEFT JOIN LATERAL (SELECT id FROM photos WHERE user_id = o.id ORDER BY position LIMIT 1) ph ON TRUE
LEFT JOIN LATERAL (
	SELECT id, sender_id, body, created_at, read_at FROM messages
	WHERE conversation_id = c.id AND created_at > COALESCE(cp.cleared_at, '-infinity')
	ORDER BY id DESC LIMIT 1) lm ON TRUE
WHERE cp.user_id = $1 AND cp.hidden_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM blocks b
                  WHERE (b.blocker_id = $1 AND b.blocked_id = o.id) OR (b.blocker_id = o.id AND b.blocked_id = $1))
  AND ($2::timestamptz IS NULL OR COALESCE(lm.created_at, c.created_at) < $2)
ORDER BY sort_at DESC
LIMIT $3`

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, listConversationsQuery, userID, before, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Conversation, error) {
		var (
			c       Conversation
			photoID *string
			msgID   *int64
			sender  *string
			body    *string
			created *time.Time
			readAt  *time.Time
			unread  int
		)
		err := row.Scan(&c.ID, &c.MatchID, &c.MatchedAt, &c.User.ID, &c.User.FirstName, &photoID,
			&c.SortAt, &msgID, &sender, &body, &created, &readAt, &unread)
		if err != nil {
			return c, err
		}
		if photoID != nil {
			u := "/photos/" + *photoID + "/file"
			c.User.PhotoURL = &u
		}
		if msgID != nil {
			c.LastMessage = &Message{ID: *msgID, ConversationID: c.ID, SenderID: *sender, Body: *body, CreatedAt: *created, ReadAt: readAt}
		}
		c.UnreadCount = unread
		return c, nil
	})
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, before *int64, limit int) ([]Message, error) {
	var cleared *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT cleared_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(&cleared)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotParticipant
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id::text, sender_id::text, body, created_at, read_at FROM messages
		WHERE conversation_id = $1
		  AND ($2::bigint IS NULL OR id < $2)
		  AND ($3::timestamptz IS NULL OR created_at > $3)
		ORDER BY id DESC LIMIT $4
	`, conversationID, before, cleared, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
		return m, err
	})
}

func (r *PGRepository) Send(ctx context.Context, userID, conversationID, body string) (SendResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SendResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// The conversation must exist (it is deleted with its match on block/unmatch),
	// the sender must belong to it, and no block may exist between the two people.
	var recipientID, senderName string
	err = tx.QueryRow(ctx, `
		SELECT o.user_id::text, sp.first_name
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		JOIN profiles sp ON sp.user_id = me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = me.user_id AND b.blocked_id = o.user_id)
		                     OR (b.blocker_id = o.user_id AND b.blocked_id = me.user_id))
	`, conversationID, userID).Scan(&recipientID, &senderName)
	if errors.Is(err, pgx.ErrNoRows) {
		return SendResult{}, ErrNotParticipant
	}
	if err != nil {
		return SendResult{}, err
	}

	msg := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, userID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return SendResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return SendResult{}, err
	}
	// A new message brings a locally deleted conversation back for both people.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NULL WHERE conversation_id = $1 AND hidden_at IS NOT NULL`, conversationID); err != nil {
		return SendResult{}, err
	}

	preview := body
	if r := []rune(preview); len(r) > 80 {
		preview = string(r[:80]) + "…"
	}
	notif, err := notifications.InsertMessageUnlessPending(ctx, tx, recipientID, conversationID, map[string]any{
		"conversationId": conversationID,
		"userId":         userID,
		"firstName":      senderName,
		"preview":        preview,
	})
	if err != nil {
		return SendResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SendResult{}, err
	}
	return SendResult{Message: msg, RecipientID: recipientID, Notification: notif}, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (ReadResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReadResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var otherID string
	err = tx.QueryRow(ctx, `
		SELECT o.user_id::text FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&otherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReadResult{}, ErrNotParticipant
	}
	if err != nil {
		return ReadResult{}, err
	}

	res := ReadResult{OtherID: otherID, ReadAt: time.Now()}
	tag, err := tx.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, conversationID, userID, res.ReadAt)
	if err != nil {
		return ReadResult{}, err
	}
	res.ReadCount = int(tag.RowsAffected())

	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2
	`, userID, conversationID); err != nil {
		return ReadResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReadResult{}, err
	}
	return res, nil
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), cleared_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotParticipant
	}
	return nil
}
