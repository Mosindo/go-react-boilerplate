package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchSummary, error)
	UnreadTotal(ctx context.Context, userID string) (int, error)
	Membership(ctx context.Context, matchID, userID string) (Membership, error)
	ListMessages(ctx context.Context, m Membership, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, m Membership, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, m Membership, userID string) (time.Time, error)
	Hide(ctx context.Context, m Membership, userID string) error
	Unmatch(ctx context.Context, m Membership) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// visibleMatchesSQL is the list of the caller's active, non-hidden matches
// ($1 = caller). A hidden conversation reappears when a newer message arrives.
const matchListSQL = `
	SELECT m.id, m.created_at,
	       peer.user_id, peer.first_name, COALESCE(photo.id::text, ''),
	       lm.id, lm.sender_user_id, lm.content, lm.created_at,
	       (SELECT count(*) FROM messages um
	         WHERE um.conversation_id = m.conversation_id
	           AND um.sender_user_id <> $1
	           AND um.created_at > COALESCE(self_cp.last_read_at, '-infinity'::timestamptz))
	FROM matches m
	JOIN conversation_participants self_cp ON self_cp.conversation_id = m.conversation_id AND self_cp.user_id = $1
	JOIN profiles peer ON peer.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
	JOIN users pu ON pu.id = peer.user_id AND pu.status = 'active'
	LEFT JOIN LATERAL (
	  SELECT id FROM photos WHERE user_id = peer.user_id AND position = 0
	) photo ON TRUE
	LEFT JOIN LATERAL (
	  SELECT id, sender_user_id, content, created_at FROM messages
	  WHERE conversation_id = m.conversation_id
	  ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE (m.user_a = $1 OR m.user_b = $1)
	  AND m.ended_at IS NULL
	  AND (self_cp.hidden_at IS NULL OR lm.created_at > self_cp.hidden_at)
`

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchSummary, error) {
	rows, err := r.pool.Query(ctx, matchListSQL+`
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []MatchSummary{}
	for rows.Next() {
		var s MatchSummary
		var lmID, lmSender, lmBody *string
		var lmAt *time.Time
		if err := rows.Scan(&s.ID, &s.CreatedAt, &s.Peer.UserID, &s.Peer.FirstName, &s.Peer.PrimaryPhoto,
			&lmID, &lmSender, &lmBody, &lmAt, &s.UnreadCount); err != nil {
			return nil, err
		}
		if lmID != nil {
			s.LastMessage = &Message{ID: *lmID, MatchID: s.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM matches m
		JOIN conversation_participants cp ON cp.conversation_id = m.conversation_id AND cp.user_id = $1
		JOIN messages msg ON msg.conversation_id = m.conversation_id
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND m.ended_at IS NULL
		  AND msg.sender_user_id <> $1
		  AND msg.created_at > COALESCE(cp.last_read_at, '-infinity'::timestamptz)
	`, userID).Scan(&total)
	return total, err
}

// Membership is the authorisation gate of every chat operation: it only
// succeeds for a participant of a match that has not ended.
func (r *PGRepository) Membership(ctx context.Context, matchID, userID string) (Membership, error) {
	var m Membership
	err := r.pool.QueryRow(ctx, `
		SELECT m.id, m.conversation_id,
		       CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END,
		       other_cp.last_read_at,
		       COALESCE(me.first_name, '')
		FROM matches m
		JOIN conversation_participants other_cp
		  ON other_cp.conversation_id = m.conversation_id
		 AND other_cp.user_id = CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END
		LEFT JOIN profiles me ON me.user_id = $2
		WHERE m.id = $1 AND m.ended_at IS NULL AND (m.user_a = $2 OR m.user_b = $2)
	`, matchID, userID).Scan(&m.MatchID, &m.ConversationID, &m.OtherUserID, &m.OtherLastRead, &m.SelfName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMatchNotFound
	}
	return m, err
}

// ListMessages returns up to limit+1 messages older than beforeID (newest
// first); the service trims the extra row to compute hasMore.
func (r *PGRepository) ListMessages(ctx context.Context, m Membership, beforeID string, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_user_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::uuid IS NULL OR (created_at, id) < (
		        SELECT created_at, id FROM messages WHERE id = $2::uuid AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, m.ConversationID, nullable(beforeID), limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		msg := Message{MatchID: m.MatchID}
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateMessage(ctx context.Context, m Membership, senderID, body string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	msg := Message{MatchID: m.MatchID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, m.ConversationID, senderID, m.OtherUserID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, m.ConversationID); err != nil {
		return Message{}, err
	}
	// Writing to a conversation un-hides it for the sender; the recipient's
	// hidden conversation resurfaces on its own because the message is newer.
	// Sending also implies the sender has read everything so far.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NULL, last_read_at = GREATEST(COALESCE(last_read_at, '-infinity'::timestamptz), $3)
		WHERE conversation_id = $1 AND user_id = $2
	`, m.ConversationID, senderID, msg.CreatedAt); err != nil {
		return Message{}, err
	}
	return msg, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, m Membership, userID string) (time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var readAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, '-infinity'::timestamptz), NOW())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, m.ConversationID, userID).Scan(&readAt); err != nil {
		return time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND type = 'new_message' AND is_read = FALSE AND data->>'matchId' = $2
	`, userID, m.MatchID); err != nil {
		return time.Time{}, err
	}
	return readAt, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, m Membership, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, m.ConversationID, userID)
	return err
}

func (r *PGRepository) Unmatch(ctx context.Context, m Membership) error {
	_, err := r.pool.Exec(ctx, `UPDATE matches SET ended_at = NOW() WHERE id = $1 AND ended_at IS NULL`, m.MatchID)
	return err
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
