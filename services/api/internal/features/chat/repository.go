package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrNotFound covers "no such conversation" and "not a participant" on purpose.
	ErrNotFound = errors.New("chat: conversation not found")
	ErrBlocked  = errors.New("chat: blocked")
)

// Access is what a participant is allowed to know about a conversation.
type Access struct {
	ConversationID string
	MatchID        string
	OtherUserID    string
	Blocked        bool
}

type SendResult struct {
	Message      Message
	Recipient    string
	Notification notifications.Notification
}

type Repository interface {
	Access(ctx context.Context, conversationID, userID string) (Access, error)
	ListMessages(ctx context.Context, conversationID string, before *Cursor, limit int) ([]Message, error)
	Send(ctx context.Context, conversationID, senderID, body string) (SendResult, error)
	MarkRead(ctx context.Context, conversationID, readerID string) (int64, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const accessSQL = `
	WITH acc AS (
	  SELECT c.id AS conversation_id, m.id AS match_id,
	         CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END AS other_id
	  FROM match_conversations c
	  JOIN matches m ON m.id = c.match_id
	  WHERE c.id = $1 AND (m.user_a = $2 OR m.user_b = $2)
	)
	SELECT a.conversation_id, a.match_id, a.other_id,
	       EXISTS (
	         SELECT 1 FROM blocks b
	         WHERE (b.blocker_id = $2 AND b.blocked_id = a.other_id)
	            OR (b.blocker_id = a.other_id AND b.blocked_id = $2))
	FROM acc a`

func access(ctx context.Context, q queryRower, conversationID, userID string) (Access, error) {
	var a Access
	err := q.QueryRow(ctx, accessSQL, conversationID, userID).Scan(&a.ConversationID, &a.MatchID, &a.OtherUserID, &a.Blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return Access{}, ErrNotFound
	}
	return a, err
}

func (r *PGRepository) Access(ctx context.Context, conversationID, userID string) (Access, error) {
	return access(ctx, r.pool, conversationID, userID)
}

func scanMessage(row pgx.Row) (Message, error) {
	var m Message
	if err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
		return Message{}, err
	}
	m.CreatedAt = m.CreatedAt.UTC()
	if m.ReadAt != nil {
		t := m.ReadAt.UTC()
		m.ReadAt = &t
	}
	return m, nil
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, before *Cursor, limit int) ([]Message, error) {
	var (
		rows pgx.Rows
		err  error
	)
	const cols = `id, conversation_id, sender_id, body, created_at, read_at`
	if before == nil {
		rows, err = r.pool.Query(ctx, `
			SELECT `+cols+` FROM match_messages WHERE conversation_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2`, conversationID, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT `+cols+` FROM match_messages
			WHERE conversation_id = $1 AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4`, conversationID, before.CreatedAt, before.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Send authorises, inserts the message, bumps the conversation and upserts the
// recipient's message notification in one transaction. The authorisation and
// block check run inside the transaction so they cannot go stale before the insert.
func (r *PGRepository) Send(ctx context.Context, conversationID, senderID, body string) (SendResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SendResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	acc, err := access(ctx, tx, conversationID, senderID)
	if err != nil {
		return SendResult{}, err
	}
	if acc.Blocked {
		return SendResult{}, ErrBlocked
	}

	msg, err := scanMessage(tx.QueryRow(ctx, `
		INSERT INTO match_messages (conversation_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at, read_at`, conversationID, senderID, body))
	if err != nil {
		return SendResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE match_conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return SendResult{}, err
	}

	var senderName string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT first_name FROM profiles WHERE user_id = $1), '')`, senderID).Scan(&senderName); err != nil {
		return SendResult{}, err
	}
	title := senderName
	if title == "" {
		title = "New message"
	}
	n, err := notifications.UpsertMessage(ctx, tx, notifications.NewNotification{
		UserID: acc.OtherUserID,
		Type:   notifications.TypeMessage,
		Title:  title,
		Body:   preview(body),
		Data:   notifications.Data{MatchID: acc.MatchID, ConversationID: conversationID, UserID: senderID},
	})
	if err != nil {
		return SendResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SendResult{}, err
	}
	return SendResult{Message: msg, Recipient: acc.OtherUserID, Notification: n}, nil
}

func preview(body string) string {
	runes := []rune(body)
	if len(runes) > 100 {
		return string(runes[:99]) + "…"
	}
	return body
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, readerID string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE match_messages SET read_at = $3
		WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL`, conversationID, readerID, time.Now())
	if err != nil {
		return 0, err
	}
	if err := notifications.MarkConversationRead(ctx, tx, readerID, conversationID); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}
