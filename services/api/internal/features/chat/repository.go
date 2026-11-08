package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("repository: not found")

type Repository interface {
	// Participant returns the peer of userID in the conversation, or ErrRepositoryNotFound if userID is not a participant.
	Participant(ctx context.Context, conversationID, userID string) (participantInfo, error)
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]storedConversation, error)
	ListMessages(ctx context.Context, conversationID, userID string, before *time.Time, limit int) ([]Message, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
	UnreadTotal(ctx context.Context, userID string) (int, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Participant(ctx context.Context, conversationID, userID string) (participantInfo, error) {
	var p participantInfo
	err := r.db.QueryRow(ctx, `
		SELECT peer.user_id, peer.last_read_at,
		       EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = peer.user_id)
		                                         OR (b.blocker_id = peer.user_id AND b.blocked_id = me.user_id))
		FROM conversation_participants me
		JOIN conversation_participants peer ON peer.conversation_id = me.conversation_id AND peer.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, conversationID, userID).
		Scan(&p.PeerID, &p.PeerLastReadAt, &p.Blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return participantInfo{}, ErrRepositoryNotFound
	}
	return p, err
}

// ListConversations returns the conversations that still have visible messages (newer than the
// user's own "clear" marker), most recently active first. Unread counts are computed in the same query.
func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, peer.user_id, peer.last_read_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages x
		         WHERE x.conversation_id = c.id AND x.sender_id <> me.user_id
		           AND x.created_at > me.last_read_at AND x.created_at > COALESCE(me.hidden_at, '-infinity'))::int
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants peer ON peer.conversation_id = c.id AND peer.user_id <> me.user_id
		JOIN LATERAL (
		  SELECT id, sender_id, body, created_at FROM messages m
		  WHERE m.conversation_id = c.id AND m.created_at > COALESCE(me.hidden_at, '-infinity')
		  ORDER BY m.created_at DESC, m.id DESC LIMIT 1) lm ON TRUE
		WHERE me.user_id = $1
		  AND ($2::timestamptz IS NULL OR lm.created_at < $2)
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = peer.user_id)
		                                            OR (b.blocker_id = peer.user_id AND b.blocked_id = me.user_id))
		ORDER BY lm.created_at DESC, c.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedConversation
	for rows.Next() {
		var s storedConversation
		m := Message{}
		if err := rows.Scan(&s.ID, &s.MatchID, &s.PeerID, &s.PeerLastReadAt, &m.ID, &m.SenderID, &m.Body, &m.CreatedAt, &s.UnreadCount); err != nil {
			return nil, err
		}
		m.ConversationID = s.ID
		s.LastMessage = &m
		s.SortAt = m.CreatedAt
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID, userID string, before *time.Time, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at
		FROM messages m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $2
		WHERE m.conversation_id = $1
		  AND m.created_at > COALESCE(me.hidden_at, '-infinity')
		  AND ($3::timestamptz IS NULL OR m.created_at < $3)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4`, conversationID, userID, before, limit)
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

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	m := Message{ConversationID: conversationID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at`, conversationID, senderID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	// The sender has necessarily read everything so far.
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `
		WITH upd AS (
		  UPDATE conversation_participants SET last_read_at = GREATEST(last_read_at, NOW())
		  WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
		), notif AS (
		  UPDATE notifications SET read_at = NOW()
		  WHERE user_id = $2 AND type = 'message' AND ref_id = $1 AND read_at IS NULL
		)
		SELECT last_read_at FROM upd`, conversationID, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrRepositoryNotFound
	}
	return t, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM messages x
		JOIN conversation_participants me ON me.conversation_id = x.conversation_id AND me.user_id = $1
		JOIN conversation_participants peer ON peer.conversation_id = x.conversation_id AND peer.user_id <> $1
		WHERE x.sender_id <> $1 AND x.created_at > me.last_read_at AND x.created_at > COALESCE(me.hidden_at, '-infinity')
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = peer.user_id)
		                                            OR (b.blocker_id = peer.user_id AND b.blocked_id = $1))`, userID).Scan(&n)
	return n, err
}
