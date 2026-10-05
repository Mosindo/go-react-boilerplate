package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConversationNotFound = errors.New("conversation not found")

type convRow struct {
	ID          string
	MatchID     string
	OtherUserID string
	Last        *Message
	Unread      int
}

// sendResult carries what the service needs to push realtime events after a message is stored.
type sendResult struct {
	Message      Message
	RecipientID  string
	Notification *notifications.Notification
}

type Repository interface {
	Access(ctx context.Context, userID, conversationID string) (otherUserID string, err error)
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]convRow, error)
	ListMessages(ctx context.Context, userID, conversationID, before string, limit int) ([]Message, bool, error)
	SendMessage(ctx context.Context, userID, conversationID, content string) (sendResult, error)
	MarkRead(ctx context.Context, userID, conversationID string) (readAt time.Time, changed int, err error)
	Clear(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// accessSQL resolves the other participant when (and only when) userID belongs to the conversation's match
// and no block exists between them. Every chat operation goes through it: a non-member gets "not found".
const accessSQL = `
	SELECT CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END
	FROM conversations c
	JOIN matches m ON m.id = c.match_id
	WHERE c.id = $1 AND (m.user_a = $2 OR m.user_b = $2)
	  AND NOT EXISTS (
	      SELECT 1 FROM blocks b
	      WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
	         OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))
`

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func access(ctx context.Context, q queryer, userID, conversationID string) (string, error) {
	var other string
	err := q.QueryRow(ctx, accessSQL, conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrConversationNotFound
	}
	return other, err
}

func (r *PGRepository) Access(ctx context.Context, userID, conversationID string) (string, error) {
	return access(ctx, r.db, userID, conversationID)
}

const messageColumns = `id, conversation_id, sender_user_id, content, created_at, read_at`

func scanMessage(row pgx.Row) (Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &m.Content, &m.CreatedAt, &m.ReadAt)
	return m, err
}

// convSelect lists conversations visible to $1: not blocked and holding at least one message newer than the
// user's "clear history" cutoff (conversations without messages live in the matches list instead).
const convSelect = `
	SELECT c.id, m.id,
	       CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END,
	       lm.id, lm.conversation_id, lm.sender_user_id, lm.content, lm.created_at, lm.read_at,
	       (SELECT count(*) FROM messages un
	         WHERE un.conversation_id = c.id AND un.recipient_user_id = $1 AND un.read_at IS NULL)
	FROM conversations c
	JOIN matches m ON m.id = c.match_id
	JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $1
	JOIN LATERAL (
	    SELECT id, conversation_id, sender_user_id, content, created_at, read_at FROM messages
	    WHERE conversation_id = c.id AND created_at > COALESCE(cp.hidden_at, '-infinity'::timestamptz)
	    ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE (m.user_a = $1 OR m.user_b = $1)
	  AND NOT EXISTS (
	      SELECT 1 FROM blocks b
	      WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
	         OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))
`

func scanConv(row pgx.Row) (convRow, error) {
	var c convRow
	var last Message
	if err := row.Scan(&c.ID, &c.MatchID, &c.OtherUserID,
		&last.ID, &last.ConversationID, &last.SenderUserID, &last.Content, &last.CreatedAt, &last.ReadAt, &c.Unread); err != nil {
		return convRow{}, err
	}
	c.Last = &last
	return c, nil
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]convRow, error) {
	rows, err := r.db.Query(ctx, convSelect+` ORDER BY lm.created_at DESC, c.id LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []convRow{}
	for rows.Next() {
		c, err := scanConv(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListMessages returns one page of messages, newest first, older than the "before" message id.
// The boolean tells whether older messages remain.
func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID, before string, limit int) ([]Message, bool, error) {
	if _, err := access(ctx, r.db, userID, conversationID); err != nil {
		return nil, false, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+messageColumns+`
		FROM messages
		WHERE conversation_id = $1
		  AND created_at > COALESCE((SELECT hidden_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2), '-infinity'::timestamptz)
		  AND ($3 = '' OR (created_at, id) < (SELECT created_at, id FROM messages WHERE id = NULLIF($3, '')::uuid AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, conversationID, userID, before, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, false, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

// SendMessage stores a message and, when the recipient had nothing unread yet, one notification, atomically.
func (r *PGRepository) SendMessage(ctx context.Context, userID, conversationID, content string) (sendResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sendResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	recipient, err := access(ctx, tx, userID, conversationID)
	if err != nil {
		return sendResult{}, err
	}

	var hadUnread bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM messages WHERE conversation_id = $1 AND recipient_user_id = $2 AND read_at IS NULL)
	`, conversationID, recipient).Scan(&hadUnread); err != nil {
		return sendResult{}, err
	}

	msg, err := scanMessage(tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING `+messageColumns, conversationID, userID, recipient, content))
	if err != nil {
		return sendResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2, updated_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return sendResult{}, err
	}

	res := sendResult{Message: msg, RecipientID: recipient}
	if !hadUnread {
		var senderName string
		if err := tx.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&senderName); err != nil {
			return sendResult{}, err
		}
		n, err := notifications.InsertTx(ctx, tx, recipient, notifications.TypeMessage, senderName,
			truncateRunes(content, 80), map[string]any{"conversationId": conversationID, "userId": userID})
		if err != nil {
			return sendResult{}, err
		}
		res.Notification = &n
	}
	if err := tx.Commit(ctx); err != nil {
		return sendResult{}, err
	}
	return res, nil
}

// MarkRead marks every message addressed to userID as read, and the matching notification too.
func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, int, error) {
	if _, err := access(ctx, r.db, userID, conversationID); err != nil {
		return time.Time{}, 0, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var readAt time.Time
	if err := tx.QueryRow(ctx, `SELECT NOW()`).Scan(&readAt); err != nil {
		return time.Time{}, 0, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE conversation_id = $1 AND recipient_user_id = $2 AND read_at IS NULL
	`, conversationID, userID, readAt)
	if err != nil {
		return time.Time{}, 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = $3
		WHERE user_id = $1 AND type = 'message' AND is_read = FALSE AND data->>'conversationId' = $2
	`, userID, conversationID, readAt); err != nil {
		return time.Time{}, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, 0, err
	}
	return readAt, int(tag.RowsAffected()), nil
}

// Clear hides the conversation history for this user only. The other participant is unaffected and a new
// message makes the conversation reappear with only the messages sent after the cut.
func (r *PGRepository) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := access(ctx, r.db, userID, conversationID); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
