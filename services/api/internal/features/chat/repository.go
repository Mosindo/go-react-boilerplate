package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotMember = errors.New("not a member of this conversation")
	ErrRepoBlocked   = errors.New("blocked")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type ConversationRow struct {
	ID             string
	MatchID        string
	PeerID         string
	UpdatedAt      time.Time
	PeerLastReadAt *time.Time
	Unread         int
	Last           *Message
}

// ListConversations returns the user's visible conversations with last message and unread
// count in one query (LATERAL join), newest activity first. Blocked pairs are excluded.
func (r *Repository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, peer.user_id, COALESCE(lm.created_at, c.created_at), peer.last_read_at,
		       (SELECT count(*) FROM messages m
		         WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id
		           AND m.created_at > COALESCE(me.last_read_at, '-infinity'::timestamptz))::int,
		       lm.id, lm.sender_id, lm.content, lm.created_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants peer ON peer.conversation_id = c.id AND peer.user_id <> me.user_id
		LEFT JOIN LATERAL (
			SELECT id, sender_id, content, created_at FROM messages
			WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1 AND me.hidden_at IS NULL
		  AND NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = me.user_id AND b.blocked_id = peer.user_id)
		           OR (b.blocker_id = peer.user_id AND b.blocked_id = me.user_id))
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ConversationRow
	for rows.Next() {
		var row ConversationRow
		var mid, msender, mcontent *string
		var mcreated *time.Time
		if err := rows.Scan(&row.ID, &row.MatchID, &row.PeerID, &row.UpdatedAt, &row.PeerLastReadAt, &row.Unread, &mid, &msender, &mcontent, &mcreated); err != nil {
			return nil, err
		}
		if mid != nil {
			row.Last = &Message{ID: *mid, ConversationID: row.ID, SenderID: *msender, Content: *mcontent, CreatedAt: *mcreated}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// Peer returns the other participant if userID belongs to the conversation.
func (r *Repository) Peer(ctx context.Context, conversationID, userID string) (string, error) {
	var peer string
	err := r.pool.QueryRow(ctx, `
		SELECT peer.user_id
		FROM conversation_participants me
		JOIN conversation_participants peer ON peer.conversation_id = me.conversation_id AND peer.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&peer)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotMember
	}
	return peer, err
}

func (r *Repository) IsBlocked(ctx context.Context, a, b string) (bool, error) {
	var blocked bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))
	`, a, b).Scan(&blocked)
	return blocked, err
}

// Messages pages backwards (newest first) using a keyset cursor on (created_at, id).
func (r *Repository) Messages(ctx context.Context, conversationID string, before *string, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::uuid IS NULL OR (created_at, id) < (
		        SELECT created_at, id FROM messages WHERE id = $2::uuid AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Insert stores a message, bumps the conversation and un-hides it for both sides.
func (r *Repository) Insert(ctx context.Context, conversationID, senderID, content string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	m := Message{ConversationID: conversationID, SenderID: senderID, Content: content}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, content) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, senderID, content).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NULL WHERE conversation_id = $1 AND hidden_at IS NOT NULL`, conversationID); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything so far.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *Repository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var at time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrRepoNotMember
	}
	return at, err
}

func (r *Repository) Hide(ctx context.Context, conversationID, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotMember
	}
	return nil
}
