package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("chat: conversation not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]storedConversation, error)
	// Membership returns the peer id when the user belongs to the conversation, else ErrNotFound.
	Membership(ctx context.Context, userID, conversationID string) (peerID string, err error)
	Messages(ctx context.Context, userID, conversationID string, limit int, before *int64) ([]Message, error)
	PeerInfo(ctx context.Context, conversationID, userID string) (peerRead *time.Time, peer storedConversation, err error)
	Send(ctx context.Context, userID, conversationID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error)
	Clear(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, peer.user_id, pf.first_name,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = peer.user_id ORDER BY ph.position LIMIT 1),
		       c.created_at, COALESCE(c.last_message_at, c.created_at),
		       COALESCE(unread.n, 0),
		       lm.id, lm.sender_id, lm.body, lm.created_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants peer ON peer.conversation_id = c.id AND peer.user_id <> me.user_id
		JOIN profiles pf ON pf.user_id = peer.user_id
		LEFT JOIN LATERAL (
		    SELECT m.id, m.sender_id, m.body, m.created_at FROM messages m
		    WHERE m.conversation_id = c.id AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)
		    ORDER BY m.created_at DESC, m.id DESC LIMIT 1) lm ON TRUE
		LEFT JOIN LATERAL (
		    SELECT COUNT(*)::int AS n FROM messages m
		    WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id AND m.created_at > me.last_read_at
		      AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)) unread ON TRUE
		WHERE me.user_id = $1 AND (me.hidden_at IS NULL OR lm.id IS NOT NULL)
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedConversation{}
	for rows.Next() {
		var s storedConversation
		var mid *int64
		var msender, mbody *string
		var mat *time.Time
		if err := rows.Scan(&s.ID, &s.MatchID, &s.PeerID, &s.PeerName, &s.PhotoID, &s.CreatedAt, &s.SortAt, &s.Unread, &mid, &msender, &mbody, &mat); err != nil {
			return nil, err
		}
		if mid != nil {
			s.Last = &Message{ID: *mid, ConversationID: s.ID, SenderID: *msender, Body: *mbody, CreatedAt: *mat}
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) Membership(ctx context.Context, userID, conversationID string) (string, error) {
	var peer string
	err := r.db.QueryRow(ctx, `
		SELECT peer.user_id FROM conversation_participants me
		JOIN conversation_participants peer ON peer.conversation_id = me.conversation_id AND peer.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, conversationID, userID).Scan(&peer)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return peer, err
}

func (r *PGRepository) Messages(ctx context.Context, userID, conversationID string, limit int, before *int64) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at
		FROM messages m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $2
		WHERE m.conversation_id = $1
		  AND (me.hidden_at IS NULL OR m.created_at > me.hidden_at)
		  AND ($4::bigint IS NULL OR m.id < $4)
		ORDER BY m.id DESC LIMIT $3`, conversationID, userID, limit, before)
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

func (r *PGRepository) PeerInfo(ctx context.Context, conversationID, userID string) (*time.Time, storedConversation, error) {
	var s storedConversation
	var readAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT peer.user_id, pf.first_name, peer.last_read_at,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = peer.user_id ORDER BY ph.position LIMIT 1)
		FROM conversation_participants peer
		JOIN profiles pf ON pf.user_id = peer.user_id
		WHERE peer.conversation_id = $1 AND peer.user_id <> $2`, conversationID, userID).
		Scan(&s.PeerID, &s.PeerName, &readAt, &s.PhotoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, s, ErrNotFound
	}
	return &readAt, s, err
}

func (r *PGRepository) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
		RETURNING id, conversation_id, sender_id, body, created_at`, conversationID, userID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything so far.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at`, conversationID, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return at, err
}

func (r *PGRepository) Clear(ctx context.Context, userID, conversationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
