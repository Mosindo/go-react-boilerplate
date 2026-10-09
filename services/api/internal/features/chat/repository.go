package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("chat: conversation not found")
	ErrBlocked  = errors.New("chat: blocked")
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error)
	ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) ([]Message, bool, error)
	SendMessage(ctx context.Context, userID, conversationID, body string) (SentMessage, error)
	MarkRead(ctx context.Context, userID, conversationID string) (ReadReceipt, error)
	Clear(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.match_id::text, `+profiles.SummarySelect+`,
		       lm.id::text, lm.body, lm.sender_id::text, lm.created_at,
		       COALESCE(un.cnt, 0),
		       COALESCE(lm.created_at, c.created_at)
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		LEFT JOIN LATERAL (
			SELECT id, body, sender_id, created_at FROM messages msg
			WHERE msg.conversation_id = c.id AND (me.cleared_at IS NULL OR msg.created_at > me.cleared_at)
			ORDER BY msg.created_at DESC, msg.id DESC LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS cnt FROM messages msg
			WHERE msg.conversation_id = c.id AND msg.sender_id <> $1 AND msg.created_at > me.last_read_at
			  AND (me.cleared_at IS NULL OR msg.created_at > me.cleared_at)
		) un ON TRUE
		WHERE me.user_id = $1 AND NOT me.hidden
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var cv Conversation
		var lmID, lmBody, lmSender *string
		var lmAt *time.Time
		if err := rows.Scan(&cv.ID, &cv.MatchID, &cv.User.ID, &cv.User.FirstName, &cv.User.Age, &cv.User.MainPhotoURL,
			&lmID, &lmBody, &lmSender, &lmAt, &cv.UnreadCount, &cv.UpdatedAt); err != nil {
			return nil, err
		}
		if lmID != nil {
			cv.LastMessage = &LastMessage{ID: *lmID, Body: *lmBody, SenderID: *lmSender, CreatedAt: *lmAt}
		}
		out = append(out, cv)
	}
	return out, rows.Err()
}

// access describes the caller's standing in a conversation.
type access struct {
	otherID     string
	clearedAt   *time.Time
	lastReadAt  time.Time
	otherReadAt time.Time
	blocked     bool
}

// loadAccess verifies the user participates in an active (not unmatched) conversation.
// Everything that touches a conversation goes through it, so non-participants
// always get ErrNotFound and never learn the conversation exists.
func loadAccess(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, userID, conversationID string, lock bool) (access, error) {
	suffix := ""
	if lock {
		suffix = " FOR SHARE OF me"
	}
	var a access
	err := q.QueryRow(ctx, `
		SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END::text,
		       me.cleared_at, me.last_read_at, other.last_read_at,
		       EXISTS (SELECT 1 FROM blocks b
		               WHERE (b.blocker_id = $1 AND b.blocked_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END)
		                  OR (b.blocked_id = $1 AND b.blocker_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END))
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $2 AND me.user_id = $1`+suffix, userID, conversationID).
		Scan(&a.otherID, &a.clearedAt, &a.lastReadAt, &a.otherReadAt, &a.blocked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return access{}, ErrNotFound
		}
		return access{}, err
	}
	return a, nil
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) ([]Message, bool, error) {
	a, err := loadAccess(ctx, r.db, userID, conversationID, false)
	if err != nil {
		return nil, false, err
	}
	if a.blocked {
		return nil, false, ErrNotFound
	}

	rows, err := r.db.Query(ctx, `
		SELECT msg.id::text, msg.conversation_id::text, msg.sender_id::text, msg.body, msg.created_at
		FROM messages msg
		WHERE msg.conversation_id = $1
		  AND ($2::timestamptz IS NULL OR msg.created_at > $2)
		  AND ($3::uuid IS NULL OR (msg.created_at, msg.id) < (SELECT created_at, id FROM messages WHERE id = $3 AND conversation_id = $1))
		ORDER BY msg.created_at DESC, msg.id DESC
		LIMIT $4
	`, conversationID, a.clearedAt, nullableUUID(beforeID), limit+1)
	if err != nil {
		if isInvalidUUID(err) {
			return nil, false, ErrNotFound
		}
		return nil, false, err
	}
	defer rows.Close()
	var desc []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, false, err
		}
		reader := a.lastReadAt
		if m.SenderID == userID {
			reader = a.otherReadAt
		}
		if !m.CreatedAt.After(reader) {
			t := reader
			m.ReadAt = &t
		}
		desc = append(desc, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(desc) > limit
	if hasMore {
		desc = desc[:limit]
	}
	// Oldest first inside the page.
	out := make([]Message, len(desc))
	for i, m := range desc {
		out[len(desc)-1-i] = m
	}
	return out, hasMore, nil
}

func (r *PGRepository) SendMessage(ctx context.Context, userID, conversationID, body string) (SentMessage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SentMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	a, err := loadAccess(ctx, tx, userID, conversationID, true)
	if err != nil {
		return SentMessage{}, err
	}
	if a.blocked {
		return SentMessage{}, ErrBlocked
	}

	msg := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, created_at
	`, conversationID, userID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return SentMessage{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return SentMessage{}, err
	}
	// Sending implies having read everything so far; a new message resurfaces a locally deleted chat.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants
		SET hidden = FALSE,
		    last_read_at = CASE WHEN user_id = $2 THEN $3 ELSE last_read_at END
		WHERE conversation_id = $1
	`, conversationID, userID, msg.CreatedAt); err != nil {
		return SentMessage{}, err
	}

	var senderName string
	if err := tx.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&senderName); err != nil {
		return SentMessage{}, err
	}
	n, err := notifications.Create(ctx, tx, notifications.NewNotification{
		UserID:    a.otherID,
		Type:      notifications.TypeMessage,
		Title:     fmt.Sprintf("Nouveau message de %s", senderName),
		Body:      preview(body),
		Data:      map[string]string{"conversationId": conversationID, "userId": userID},
		DedupeKey: "msg:" + conversationID,
	})
	if err != nil {
		return SentMessage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SentMessage{}, err
	}
	return SentMessage{Message: msg, RecipientID: a.otherID, Notification: n}, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (ReadReceipt, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ReadReceipt{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	a, err := loadAccess(ctx, tx, userID, conversationID, true)
	if err != nil {
		return ReadReceipt{}, err
	}
	rc := ReadReceipt{OtherUserID: a.otherID}
	if err := tx.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
	`, conversationID, userID).Scan(&rc.ReadAt); err != nil {
		return ReadReceipt{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND dedupe_key = $2 AND is_read = FALSE
	`, userID, "msg:"+conversationID); err != nil {
		return ReadReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReadReceipt{}, err
	}
	return rc, nil
}

func (r *PGRepository) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := loadAccess(ctx, r.db, userID, conversationID, false); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants
		SET cleared_at = clock_timestamp(), hidden = TRUE, last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}

func nullableUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func preview(body string) string {
	runes := []rune(body)
	if len(runes) > 120 {
		return string(runes[:117]) + "..."
	}
	return body
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
