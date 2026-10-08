package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotParticipant = errors.New("not a participant")

type convRow struct {
	ID, MatchID, OtherID string
	LastMessageAt        time.Time
	Unread               int
	Last                 *Message
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, afterAt *time.Time, afterID string) ([]convRow, error)
	ListMessages(ctx context.Context, userID, convID string, limit int, beforeAt *time.Time, beforeID string) ([]Message, time.Time, error)
	// Send stores the message and returns it with the other participant's id.
	Send(ctx context.Context, userID, convID, body string) (Message, string, error)
	MarkRead(ctx context.Context, userID, convID string) (otherID string, readAt time.Time, err error)
	Hide(ctx context.Context, userID, convID string) error
	Unread(ctx context.Context, userID string) (UnreadResponse, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, afterAt *time.Time, afterID string) ([]convRow, error) {
	var afterIDArg any
	if afterID != "" {
		afterIDArg = afterID
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id,
		       CASE WHEN m.user_a_id = $1::uuid THEN m.user_b_id ELSE m.user_a_id END,
		       c.last_message_at,
		       (SELECT count(*) FROM messages x WHERE x.conversation_id = c.id AND x.sender_id <> $1::uuid AND x.created_at > me.last_read_at),
		       lm.id, lm.sender_id, lm.body, lm.created_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN LATERAL (SELECT id, sender_id, body, created_at FROM messages
		              WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1) lm ON TRUE
		WHERE me.user_id = $1::uuid
		  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		  AND ($3::timestamptz IS NULL OR (c.last_message_at, c.id) < ($3::timestamptz, $4::uuid))
		ORDER BY c.last_message_at DESC, c.id DESC
		LIMIT $2`, userID, limit, afterAt, afterIDArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []convRow
	for rows.Next() {
		var c convRow
		var lm Message
		if err := rows.Scan(&c.ID, &c.MatchID, &c.OtherID, &c.LastMessageAt, &c.Unread, &lm.ID, &lm.SenderID, &lm.Body, &lm.CreatedAt); err != nil {
			return nil, err
		}
		lm.ConversationID = c.ID
		c.Last = &lm
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) participantOther(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID, convID string) (string, error) {
	var other string
	err := q.QueryRow(ctx, `
		SELECT p.user_id FROM conversation_participants me
		JOIN conversation_participants p ON p.conversation_id = me.conversation_id AND p.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotParticipant
	}
	return other, err
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, convID string, limit int, beforeAt *time.Time, beforeID string) ([]Message, time.Time, error) {
	other, err := r.participantOther(ctx, r.db, userID, convID)
	if err != nil {
		return nil, time.Time{}, err
	}
	var readAt time.Time
	if err := r.db.QueryRow(ctx, `SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2`, convID, other).Scan(&readAt); err != nil {
		return nil, time.Time{}, err
	}
	var beforeIDArg any
	if beforeID != "" {
		beforeIDArg = beforeID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3::timestamptz, $4::uuid))
		ORDER BY created_at DESC, id DESC LIMIT $2`, convID, limit, beforeAt, beforeIDArg)
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
	return out, readAt, rows.Err()
}

func (r *PGRepository) Send(ctx context.Context, userID, convID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	other, err := r.participantOther(ctx, tx, userID, convID)
	if err != nil {
		return Message{}, "", err
	}
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at`, convID, userID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// The sender has read their own message; a new message resurfaces a conversation hidden by either side.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants
		SET last_read_at = CASE WHEN user_id = $2 THEN GREATEST(last_read_at, $3) ELSE last_read_at END, hidden_at = NULL
		WHERE conversation_id = $1`, convID, userID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	return m, other, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, convID string) (string, time.Time, error) {
	other, err := r.participantOther(ctx, r.db, userID, convID)
	if err != nil {
		return "", time.Time{}, err
	}
	var readAt time.Time
	err = r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = GREATEST(last_read_at, NOW())
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at`, convID, userID).Scan(&readAt)
	return other, readAt, err
}

func (r *PGRepository) Hide(ctx context.Context, userID, convID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotParticipant
	}
	return nil
}

func (r *PGRepository) Unread(ctx context.Context, userID string) (UnreadResponse, error) {
	var u UnreadResponse
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(sum(n), 0), count(*) FILTER (WHERE n > 0) FROM (
		  SELECT (SELECT count(*) FROM messages x
		           WHERE x.conversation_id = me.conversation_id AND x.sender_id <> me.user_id AND x.created_at > me.last_read_at) AS n
		  FROM conversation_participants me WHERE me.user_id = $1) t`, userID).Scan(&u.UnreadMessages, &u.UnreadConversations)
	return u, err
}
