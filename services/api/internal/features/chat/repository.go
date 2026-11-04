package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]storedConversation, error)
	Access(ctx context.Context, conversationID, userID string) (access, error)
	ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, before int64, limit int) ([]Message, error)
	PeerLastRead(ctx context.Context, conversationID, userID string) (int64, error)
	AddMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (int64, error)
	Hide(ctx context.Context, conversationID, userID string) error
	Unmatch(ctx context.Context, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, m.id, c.created_at, c.last_message_at,
		       o.user_id, p.first_name, ph.id,
		       lm.id::text, lm.sender_id, lm.body, lm.created_at,
		       COALESCE(un.n, 0)
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN LATERAL (SELECT CASE WHEN m.user_a_id = cp.user_id THEN m.user_b_id ELSE m.user_a_id END AS user_id) o ON TRUE
		JOIN profiles p ON p.user_id = o.user_id
		LEFT JOIN LATERAL (SELECT id FROM photos WHERE user_id = o.user_id ORDER BY position LIMIT 1) ph ON TRUE
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at FROM messages
		  WHERE conversation_id = c.id AND (cp.hidden_at IS NULL OR created_at > cp.hidden_at)
		  ORDER BY id DESC LIMIT 1) lm ON TRUE
		LEFT JOIN LATERAL (
		  SELECT COUNT(*)::int AS n FROM messages
		  WHERE conversation_id = c.id AND id > cp.last_read_message_id AND sender_id <> cp.user_id) un ON TRUE
		WHERE cp.user_id = $1
		  AND (cp.hidden_at IS NULL OR lm.id IS NOT NULL)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = o.user_id) OR (b.blocker_id = o.user_id AND b.blocked_id = $1))
		  AND ($2::timestamptz IS NULL OR c.last_message_at < $2)
		ORDER BY c.last_message_at DESC, c.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedConversation{}
	for rows.Next() {
		var sc storedConversation
		var lmID, lmSender, lmBody *string
		var lmAt *time.Time
		if err := rows.Scan(&sc.ID, &sc.MatchID, &sc.CreatedAt, &sc.SortAt,
			&sc.With.UserID, &sc.With.FirstName, &sc.PhotoID,
			&lmID, &lmSender, &lmBody, &lmAt, &sc.Unread); err != nil {
			return nil, err
		}
		if lmID != nil {
			sc.LastMessage = &Message{ID: *lmID, ConversationID: sc.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (r *PGRepository) Access(ctx context.Context, conversationID, userID string) (access, error) {
	var a access
	err := r.db.QueryRow(ctx, `
		SELECT o.user_id, cp.hidden_at,
		       EXISTS (SELECT 1 FROM blocks b
		               WHERE (b.blocker_id = cp.user_id AND b.blocked_id = o.user_id)
		                  OR (b.blocker_id = o.user_id AND b.blocked_id = cp.user_id))
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN LATERAL (SELECT CASE WHEN m.user_a_id = cp.user_id THEN m.user_b_id ELSE m.user_a_id END AS user_id) o ON TRUE
		WHERE cp.conversation_id = $1 AND cp.user_id = $2`, conversationID, userID).Scan(&a.OtherUserID, &a.HiddenAt, &a.Blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return access{}, ErrNotFound
	}
	return a, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, hiddenAt *time.Time, before int64, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1 AND ($2::bigint = 0 OR id < $2) AND ($3::timestamptz IS NULL OR created_at > $3)
		ORDER BY id DESC LIMIT $4`, conversationID, before, hiddenAt, limit)
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

func (r *PGRepository) PeerLastRead(ctx context.Context, conversationID, userID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(last_read_message_id), 0) FROM conversation_participants
		WHERE conversation_id = $1 AND user_id <> $2`, conversationID, userID).Scan(&id)
	return id, err
}

func (r *PGRepository) AddMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, conversation_id, sender_id, body, created_at`, conversationID, senderID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything so far.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_message_id = $3
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, senderID, m.ID); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants cp
		SET last_read_message_id = GREATEST(cp.last_read_message_id,
		    COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = cp.conversation_id), 0))
		WHERE cp.conversation_id = $1 AND cp.user_id = $2
		RETURNING cp.last_read_message_id`, conversationID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Unmatch deletes the match; conversation, participants and messages cascade.
func (r *PGRepository) Unmatch(ctx context.Context, conversationID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM matches WHERE id = (SELECT match_id FROM conversations WHERE id = $1)`, conversationID)
	return err
}
