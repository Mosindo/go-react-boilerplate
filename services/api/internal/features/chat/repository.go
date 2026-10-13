package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned both when a conversation does not exist and when the
// caller is not allowed to see it, so existence is never leaked.
var ErrNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredConversation, error)
	// Access returns the other participant if userID may use the conversation
	// (participant, match still active, no block in either direction).
	Access(ctx context.Context, userID, conversationID string) (otherUserID string, err error)
	ListMessages(ctx context.Context, userID, conversationID string, before *int64, limit int) ([]Message, error)
	// InsertMessage stores a message and its side effects (unread state and the
	// aggregated notification for the recipient) in one transaction.
	InsertMessage(ctx context.Context, userID, conversationID, body string) (Message, string, error)
	MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const accessSQL = `
	SELECT other.user_id::text
	FROM conversation_participants cp
	JOIN conversations c ON c.id = cp.conversation_id
	JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
	JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> cp.user_id
	WHERE cp.conversation_id = $1::uuid AND cp.user_id = $2::uuid
	  AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = cp.user_id AND b.blocked_id = other.user_id)
		   OR (b.blocker_id = other.user_id AND b.blocked_id = cp.user_id)
	  )`

func (r *PGRepository) Access(ctx context.Context, userID, conversationID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, accessSQL, conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return other, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, other.user_id, c.last_message_at,
		       (SELECT count(*) FROM messages x
		         WHERE x.conversation_id = c.id AND x.sender_id <> $1 AND x.created_at > cp.last_read_at)::int,
		       lm.body, lm.sender_id, lm.created_at
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> cp.user_id
		JOIN LATERAL (
			SELECT body, sender_id, created_at FROM messages
			WHERE conversation_id = c.id ORDER BY id DESC LIMIT 1
		) lm ON TRUE
		WHERE cp.user_id = $1
		  AND c.last_message_at IS NOT NULL
		  AND (cp.hidden_at IS NULL OR c.last_message_at > cp.hidden_at)
		  AND ($2::timestamptz IS NULL OR c.last_message_at < $2)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = cp.user_id AND b.blocked_id = other.user_id)
			   OR (b.blocker_id = other.user_id AND b.blocked_id = cp.user_id)
		  )
		ORDER BY c.last_message_at DESC, c.id
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredConversation{}
	for rows.Next() {
		var c StoredConversation
		if err := rows.Scan(&c.ID, &c.MatchID, &c.OtherUserID, &c.LastMessageAt, &c.UnreadCount,
			&c.LastMessage.Body, &c.LastMessage.SenderID, &c.LastMessage.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, before *int64, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at,
		       CASE WHEN m.sender_id = $2 THEN m.created_at <= other.last_read_at
		            ELSE TRUE END
		FROM messages m
		JOIN conversation_participants other ON other.conversation_id = m.conversation_id AND other.user_id <> $2
		WHERE m.conversation_id = $1 AND ($3::bigint IS NULL OR m.id < $3)
		ORDER BY m.id DESC
		LIMIT $4
	`, conversationID, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.Read); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	// Newest first from the database; the API returns chronological order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

func (r *PGRepository) InsertMessage(ctx context.Context, userID, conversationID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var other string
	err = tx.QueryRow(ctx, accessSQL, conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, "", ErrNotFound
	}
	if err != nil {
		return Message{}, "", err
	}

	msg := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, userID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// Sending implies having read everything so far.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3, hidden_at = NULL
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// One unread notification per conversation: later messages refresh it.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $2::uuid, 'message', p.first_name, $4,
		       jsonb_build_object('conversationId', $1::text, 'userId', $3::text, 'name', p.first_name)
		FROM profiles p WHERE p.user_id = $3::uuid
		ON CONFLICT (user_id, (data->>'conversationId')) WHERE type = 'message' AND read_at IS NULL
		DO UPDATE SET body = EXCLUDED.body, created_at = NOW()
	`, conversationID, other, userID, Preview(body, 80)); err != nil {
		return Message{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, "", err
	}
	return msg, other, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var readAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	if err != nil {
		return time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2
	`, userID, conversationID); err != nil {
		return time.Time{}, err
	}
	return readAt, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
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
