package chat

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type cursor struct {
	At time.Time
	ID string
}

type access struct {
	OtherUserID     string
	HiddenAt        *time.Time
	OtherLastReadAt time.Time
}

type conversationRow struct {
	ConversationSummary
	sortAt time.Time
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *cursor, limit int) ([]conversationRow, error)
	UnreadTotal(ctx context.Context, userID string) (int, error)
	Access(ctx context.Context, conversationID, userID string) (access, error)
	ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, before *cursor, limit int) ([]Message, error)
	InsertMessage(ctx context.Context, conversationID, senderID, otherID, content string) (msg Message, senderName string, notified bool, err error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *cursor, limit int) ([]conversationRow, error) {
	const q = `
		SELECT c.id, c.match_id, c.created_at, op.user_id, pr.first_name,
		       lm.id, lm.sender_id, lm.content, lm.created_at,
		       (SELECT count(*) FROM messages m
		         WHERE m.conversation_id = c.id AND m.sender_id <> $1
		           AND m.created_at > cp.last_read_at
		           AND (cp.hidden_at IS NULL OR m.created_at > cp.hidden_at)) AS unread,
		       COALESCE(lm.created_at, c.created_at) AS sort_at
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN conversation_participants op ON op.conversation_id = c.id AND op.user_id <> cp.user_id
		JOIN profiles pr ON pr.user_id = op.user_id
		LEFT JOIN LATERAL (
			SELECT m.id, m.sender_id, m.content, m.created_at FROM messages m
			WHERE m.conversation_id = c.id AND (cp.hidden_at IS NULL OR m.created_at > cp.hidden_at)
			ORDER BY m.created_at DESC, m.id DESC LIMIT 1
		) lm ON TRUE
		WHERE cp.user_id = $1
		  AND (cp.hidden_at IS NULL OR lm.id IS NOT NULL)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = cp.user_id AND b.blocked_id = op.user_id)
			   OR (b.blocker_id = op.user_id AND b.blocked_id = cp.user_id))
		  AND ($2::timestamptz IS NULL OR (COALESCE(lm.created_at, c.created_at), c.id) < ($2, $3::uuid))
		ORDER BY sort_at DESC, c.id DESC
		LIMIT $4`
	var at *time.Time
	var id *string
	if before != nil {
		at, id = &before.At, &before.ID
	}
	rows, err := r.db.Query(ctx, q, userID, at, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conversationRow
	for rows.Next() {
		var row conversationRow
		var lmID, lmSender, lmContent *string
		var lmAt *time.Time
		if err := rows.Scan(&row.ID, &row.MatchID, &row.CreatedAt, &row.User.UserID, &row.User.Name,
			&lmID, &lmSender, &lmContent, &lmAt, &row.UnreadCount, &row.sortAt); err != nil {
			return nil, err
		}
		if lmID != nil {
			row.LastMessage = &Message{ID: *lmID, ConversationID: row.ID, SenderID: *lmSender, Content: *lmContent, CreatedAt: *lmAt}
		}
		row.UpdatedAt = row.sortAt
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM conversation_participants cp
		JOIN conversation_participants op ON op.conversation_id = cp.conversation_id AND op.user_id <> cp.user_id
		JOIN messages m ON m.conversation_id = cp.conversation_id
		WHERE cp.user_id = $1 AND m.sender_id <> $1
		  AND m.created_at > cp.last_read_at
		  AND (cp.hidden_at IS NULL OR m.created_at > cp.hidden_at)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = cp.user_id AND b.blocked_id = op.user_id)
			   OR (b.blocker_id = op.user_id AND b.blocked_id = cp.user_id))
	`, userID).Scan(&n)
	return n, err
}

// Access is the single authorization gate for every conversation endpoint: the
// caller must be a participant and neither side may have blocked the other.
// Anything else answers "not found" so conversations cannot be probed.
func (r *PGRepository) Access(ctx context.Context, conversationID, userID string) (access, error) {
	var a access
	err := r.db.QueryRow(ctx, `
		SELECT op.user_id, cp.hidden_at, op.last_read_at
		FROM conversation_participants cp
		JOIN conversation_participants op ON op.conversation_id = cp.conversation_id AND op.user_id <> cp.user_id
		WHERE cp.conversation_id = $1 AND cp.user_id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = cp.user_id AND b.blocked_id = op.user_id)
			   OR (b.blocker_id = op.user_id AND b.blocked_id = cp.user_id))
	`, conversationID, userID).Scan(&a.OtherUserID, &a.HiddenAt, &a.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return access{}, ErrNotFound
	}
	return a, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, before *cursor, limit int) ([]Message, error) {
	var at *time.Time
	var id *string
	if before != nil {
		at, id = &before.At, &before.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5
	`, conversationID, hiddenAt, at, id, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// InsertMessage stores the message, marks the conversation read for
// the sender and creates at most one unread "message" notification per
// conversation for the recipient.
func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, otherID, content string) (Message, string, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	msg := Message{ConversationID: conversationID, SenderID: senderID, Content: content}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, content) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, senderID, content).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, "", false, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, msg.CreatedAt); err != nil {
		return Message{}, "", false, err
	}
	var senderName string
	if err := tx.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, senderID).Scan(&senderName); err != nil {
		return Message{}, "", false, err
	}
	data, _ := json.Marshal(map[string]string{"conversationId": conversationID, "userId": senderID})
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT $1, 'message', $2, $3, $4::jsonb
		WHERE NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'conversationId' = $5)
	`, otherID, "Nouveau message de "+senderName, "Ouvrez la conversation pour le lire.", string(data), conversationID)
	if err != nil {
		return Message{}, "", false, err
	}
	return msg, senderName, tag.RowsAffected() > 0, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var readAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt); err != nil {
		return time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND NOT is_read AND data->>'conversationId' = $2
	`, userID, conversationID); err != nil {
		return time.Time{}, err
	}
	return readAt, tx.Commit(ctx)
}

// Hide removes the conversation from the caller's list only. Messages sent
// before the hide stay invisible to them; the other person is unaffected.
func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}
