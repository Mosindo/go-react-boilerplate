package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error)
	// Peer returns the other participant when userID belongs to an active, unblocked conversation.
	Peer(ctx context.Context, conversationID, userID string) (peerID string, err error)
	ListMessages(ctx context.Context, conversationID, userID, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, userID, peerID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (int64, error)
	Hide(ctx context.Context, conversationID, userID string) error
	UnreadTotal(ctx context.Context, userID string) (int, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// activeConversation restricts to conversations the user participates in, whose match is
// still active and where neither side blocked the other. $1 = conversation id, $2 = user id.
const activeConversation = `
SELECT CASE WHEN m.user_a_id = $2 THEN m.user_b_id ELSE m.user_a_id END::text
FROM conversations c JOIN matches m ON m.id = c.match_id
WHERE c.id = $1 AND m.unmatched_at IS NULL AND (m.user_a_id = $2 OR m.user_b_id = $2)
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = m.user_a_id AND b.blocked_id = m.user_b_id)
                                            OR (b.blocker_id = m.user_b_id AND b.blocked_id = m.user_a_id))`

func (r *PGRepository) Peer(ctx context.Context, conversationID, userID string) (string, error) {
	var peer string
	if err := r.pool.QueryRow(ctx, activeConversation, conversationID, userID).Scan(&peer); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return peer, nil
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id::text, m.id::text, peer.user_id::text, peer.first_name,
		       (SELECT ph.id::text FROM photos ph WHERE ph.user_id = peer.user_id ORDER BY ph.position LIMIT 1),
		       c.created_at,
		       lm.id::text, lm.sender_id::text, lm.body, lm.created_at, lm.read_at,
		       (SELECT count(*) FROM messages um WHERE um.conversation_id = c.id AND um.sender_id <> $1
		          AND um.read_at IS NULL AND um.created_at > COALESCE(h.hidden_at, '-infinity')),
		       GREATEST(c.created_at, COALESCE(lm.created_at, c.created_at))
		FROM conversations c
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL AND (m.user_a_id = $1 OR m.user_b_id = $1)
		JOIN profiles peer ON peer.user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		LEFT JOIN conversation_hidden h ON h.conversation_id = c.id AND h.user_id = $1
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at, read_at FROM messages
		  WHERE conversation_id = c.id AND created_at > COALESCE(h.hidden_at, '-infinity')
		  ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = m.user_a_id AND b.blocked_id = m.user_b_id)
		                                           OR (b.blocker_id = m.user_b_id AND b.blocked_id = m.user_a_id))
		  -- a locally deleted conversation stays hidden until a new message arrives
		  AND (h.hidden_at IS NULL OR lm.id IS NOT NULL)
		  AND ($2::timestamptz IS NULL OR GREATEST(c.created_at, COALESCE(lm.created_at, c.created_at)) < $2)
		ORDER BY GREATEST(c.created_at, COALESCE(lm.created_at, c.created_at)) DESC, c.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var cv Conversation
		var photoID, lmID, lmSender, lmBody *string
		var lmAt, lmRead *time.Time
		if err := rows.Scan(&cv.ID, &cv.MatchID, &cv.Peer.UserID, &cv.Peer.FirstName, &photoID, &cv.CreatedAt,
			&lmID, &lmSender, &lmBody, &lmAt, &lmRead, &cv.UnreadCount, &cv.SortAt); err != nil {
			return nil, err
		}
		if photoID != nil {
			cv.Peer.PhotoURL = "/photos/" + *photoID
		}
		if lmID != nil {
			cv.LastMessage = &Message{ID: *lmID, ConversationID: cv.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt, ReadAt: lmRead}
		}
		out = append(out, cv)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID, userID, beforeID string, limit int) ([]Message, error) {
	if _, err := r.Peer(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	var cursor *string
	if beforeID != "" {
		cursor = &beforeID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id::text, m.conversation_id::text, m.sender_id::text, m.body, m.created_at, m.read_at
		FROM messages m
		LEFT JOIN conversation_hidden h ON h.conversation_id = m.conversation_id AND h.user_id = $2
		WHERE m.conversation_id = $1 AND m.created_at > COALESCE(h.hidden_at, '-infinity')
		  AND ($3::uuid IS NULL OR (m.created_at, m.id) < (SELECT created_at, id FROM messages WHERE id = $3 AND conversation_id = $1))
		ORDER BY m.created_at DESC, m.id DESC LIMIT $4`, conversationID, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, userID, peerID, body string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var msg Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, conversation_id::text, sender_id::text, body, created_at, read_at`,
		conversationID, userID, body).Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Body, &msg.CreatedAt, &msg.ReadAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return Message{}, err
	}
	// One pending "message" notification per match: avoids flooding while the peer is away.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id)
		SELECT $2, 'message', $3, c.match_id FROM conversations c
		WHERE c.id = $1 AND NOT EXISTS (
		  SELECT 1 FROM notifications n WHERE n.user_id = $2 AND n.match_id = c.match_id AND n.type = 'message' AND n.read_at IS NULL)`,
		conversationID, peerID, userID); err != nil {
		return Message{}, err
	}
	return msg, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE messages SET read_at = NOW()
		WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL`, conversationID, userID)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $2 AND type = 'message' AND read_at IS NULL
		  AND match_id = (SELECT match_id FROM conversations WHERE id = $1)`, conversationID, userID); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO conversation_hidden (conversation_id, user_id) VALUES ($1, $2)
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET hidden_at = NOW()`, conversationID, userID)
	return err
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM messages msg
		JOIN conversations c ON c.id = msg.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL AND (m.user_a_id = $1 OR m.user_b_id = $1)
		LEFT JOIN conversation_hidden h ON h.conversation_id = c.id AND h.user_id = $1
		WHERE msg.sender_id <> $1 AND msg.read_at IS NULL AND msg.created_at > COALESCE(h.hidden_at, '-infinity')
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = m.user_a_id AND b.blocked_id = m.user_b_id)
		                                           OR (b.blocker_id = m.user_b_id AND b.blocked_id = m.user_a_id))`, userID).Scan(&n)
	return n, err
}
