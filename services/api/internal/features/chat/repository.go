package chat

import (
	"context"
	"errors"
	"strconv"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNotFound = platformerrors.NotFound("conversation not found")

type Repository interface {
	Participant(ctx context.Context, conversationID, userID string) (participant, error)
	ListConversations(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]ConversationItem, error)
	ListMessages(ctx context.Context, conversationID, userID string, beforeID int64, limit int) ([]Message, *time.Time, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Participant is the authorization check used by every conversation endpoint:
// the caller must be a participant, otherwise the conversation "does not exist".
func (r *PGRepository) Participant(ctx context.Context, conversationID, userID string) (participant, error) {
	var p participant
	err := r.db.QueryRow(ctx, `
		SELECT other.user_id::text, me.hidden_at
		FROM conversation_participants me
		JOIN conversation_participants other
		  ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&p.OtherUserID, &p.HiddenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return participant{}, errNotFound
	}
	return p, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]ConversationItem, error) {
	var at *time.Time
	cursorID := "00000000-0000-0000-0000-000000000000"
	if cursor != nil {
		at = &cursor.At
		cursorID = cursor.ID
	}
	// One query: last message and unread count come from LATERAL subqueries,
	// so cost is O(page size), not O(conversations).
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, other.user_id::text, pf.first_name,
		       CASE WHEN ph.id IS NULL THEN NULL ELSE '/photos/' || ph.id::text || '/file' END,
		       lm.id::text, lm.sender_id::text, lm.body, lm.created_at,
		       COALESCE(unread.n, 0),
		       COALESCE(c.last_message_at, c.created_at) AS sort_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
		LEFT JOIN profiles pf ON pf.user_id = other.user_id
		LEFT JOIN photos ph ON ph.user_id = other.user_id AND ph.position = 0
		LEFT JOIN LATERAL (
		  SELECT m.id, m.sender_id, m.body, m.created_at FROM messages m
		  WHERE m.conversation_id = c.id AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)
		  ORDER BY m.id DESC LIMIT 1) lm ON TRUE
		LEFT JOIN LATERAL (
		  SELECT count(*) AS n FROM messages m
		  WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id
		    AND (me.last_read_at IS NULL OR m.created_at > me.last_read_at)
		    AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)) unread ON TRUE
		WHERE me.user_id = $1
		  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		  AND ($2::timestamptz IS NULL OR (COALESCE(c.last_message_at, c.created_at), c.id) < ($2::timestamptz, $3::uuid))
		ORDER BY sort_at DESC, c.id DESC
		LIMIT $4
	`, userID, at, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ConversationItem{}
	for rows.Next() {
		var (
			item     ConversationItem
			msgID    *string
			senderID *string
			body     *string
			msgAt    *time.Time
		)
		if err := rows.Scan(&item.ID, &item.User.UserID, &item.User.FirstName, &item.User.PhotoURL,
			&msgID, &senderID, &body, &msgAt, &item.UnreadCount, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if msgID != nil {
			item.LastMessage = &Message{ID: *msgID, ConversationID: item.ID, SenderID: *senderID, Body: *body, CreatedAt: *msgAt}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID, userID string, beforeID int64, limit int) ([]Message, *time.Time, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text, m.sender_id::text, m.body, m.created_at
		FROM messages m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $2
		WHERE m.conversation_id = $1
		  AND ($3::bigint = 0 OR m.id < $3)
		  AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)
		ORDER BY m.id DESC LIMIT $4
	`, conversationID, userID, beforeID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		m := Message{ConversationID: conversationID}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	rows.Close()

	var otherRead *time.Time
	err = r.db.QueryRow(ctx, `
		SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id <> $2
	`, conversationID, userID).Scan(&otherRead)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	return out, otherRead, nil
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	m := Message{ConversationID: conversationID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, created_at
	`, conversationID, senderID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything up to now.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
	`, conversationID, userID).Scan(&at)
	return at, err
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	return err
}

func parseMessageCursor(raw string) (int64, bool) {
	decoded, ok := httpx.DecodeCursor(raw)
	if !ok {
		return 0, false
	}
	if decoded == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(decoded, 10, 64)
	return id, err == nil && id > 0
}
