package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrForbidden covers "not a participant", "unmatched", "blocked" and
	// "does not exist" on purpose: callers must not learn which one applies.
	ErrForbidden = errors.New("conversation not available")
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int) ([]storedConversation, error)
	// Participant reports the other user's id if userID may use the conversation.
	Participant(ctx context.Context, userID, conversationID string) (otherUserID string, err error)
	ListMessages(ctx context.Context, conversationID string, before *time.Time, limit int) ([]Message, error)
	OtherLastRead(ctx context.Context, conversationID, userID string) (*time.Time, error)
	InsertMessage(ctx context.Context, conversationID, senderID, otherUserID, senderName, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) error
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// accessSQL is the single authorization rule for conversations: the user is a
// participant, the match is active, and no block exists in either direction.
const accessSQL = `
	FROM conversations c
	JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
	JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
	JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
	LEFT JOIN LATERAL (
		SELECT body, sender_id, created_at FROM messages
		WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = other.user_id) OR (b.blocker_id = other.user_id AND b.blocked_id = $1))`

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, other.user_id, c.created_at,
		       lm.body, lm.sender_id, lm.created_at,
		       (SELECT COUNT(*) FROM messages mm WHERE mm.conversation_id = c.id AND mm.sender_id <> $1 AND mm.created_at > me.last_read_at)
		`+accessSQL+`
		  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedConversation{}
	for rows.Next() {
		var s storedConversation
		if err := rows.Scan(&s.ID, &s.MatchID, &s.OtherUserID, &s.CreatedAt, &s.LastBody, &s.LastSenderID, &s.LastMessageAt, &s.UnreadCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) Participant(ctx context.Context, userID, conversationID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `SELECT other.user_id `+accessSQL+` AND c.id = $2`, userID, conversationID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", ErrForbidden
	}
	return other, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, before *time.Time, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3`, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) OtherLastRead(ctx context.Context, conversationID, userID string) (*time.Time, error) {
	var t *time.Time
	err := r.db.QueryRow(ctx, `SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id <> $2`, conversationID, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, otherUserID, senderName, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1,$2,$3)
		RETURNING id, conversation_id, sender_id, body, created_at`, conversationID, senderID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies the sender has read everything so far.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3, hidden_at = NULL WHERE conversation_id = $1 AND user_id = $2`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NULL WHERE conversation_id = $1 AND user_id = $2`, conversationID, otherUserID); err != nil {
		return Message{}, err
	}
	// One unread "message" notification per conversation, refreshed instead of duplicated.
	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2`, otherUserID, conversationID); err != nil {
		return Message{}, err
	}
	if err := notifications.Insert(ctx, tx, otherUserID, notifications.TypeMessage, senderName, preview(body),
		map[string]string{"conversationId": conversationID, "userId": senderID}); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2`, userID, conversationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Hide removes the conversation from the user's list only; the other side keeps it.
// It reappears when a new message arrives.
func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	return err
}

func preview(body string) string {
	runes := []rune(body)
	if len(runes) > 80 {
		return string(runes[:80]) + "…"
	}
	return body
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
