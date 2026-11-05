package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrForbidden covers "not a participant", "match ended" and "blocked": the
// caller must not learn which, and must not learn that the conversation exists.
var ErrForbidden = errors.New("conversation not accessible")

type Repository interface {
	List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Conversation, error)
	Messages(ctx context.Context, userID, convID string, beforeID int64, limit int) ([]Message, error)
	Send(ctx context.Context, userID, convID, body string) (msg Message, senderName, recipientID string, err error)
	MarkRead(ctx context.Context, userID, convID string) (otherID string, upTo int64, err error)
	Hide(ctx context.Context, userID, convID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrForbidden
	}
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Conversation, error) {
	q := `
	SELECT c.id, m.id, m.created_at, peer.user_id, COALESCE(pp.first_name, ''),
	       (SELECT ph.id FROM photos ph WHERE ph.user_id = peer.user_id AND ph.position = 0),
	       lm.id, lm.sender_id, lm.body, lm.created_at,
	       (SELECT COUNT(*) FROM messages x WHERE x.conversation_id = c.id AND x.sender_id <> $1
	          AND x.id > me.last_read_message_id AND x.id > COALESCE(me.hidden_before_message_id, 0))::int,
	       COALESCE(lm.created_at, m.created_at) AS activity
	FROM conversation_participants me
	JOIN conversations c ON c.id = me.conversation_id
	JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
	JOIN conversation_participants peer ON peer.conversation_id = c.id AND peer.user_id <> $1
	LEFT JOIN profiles pp ON pp.user_id = peer.user_id
	LEFT JOIN LATERAL (
	  SELECT x.id, x.sender_id, x.body, x.created_at FROM messages x
	  WHERE x.conversation_id = c.id ORDER BY x.id DESC LIMIT 1) lm ON TRUE
	WHERE me.user_id = $1
	  AND (me.hidden_before_message_id IS NULL OR lm.id > me.hidden_before_message_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = peer.user_id)
	                                             OR (b.blocker_id = peer.user_id AND b.blocked_id = $1))`
	args := []any{userID}
	if beforeAt != nil {
		q += ` AND (COALESCE(lm.created_at, m.created_at), c.id) < ($2, $3::uuid)`
		args = append(args, *beforeAt, beforeID)
	}
	q += ` ORDER BY activity DESC, c.id DESC LIMIT ` + itoa(limit)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var cv Conversation
		var lmID *int64
		var lmSender, lmBody *string
		var lmAt *time.Time
		if err := rows.Scan(&cv.ID, &cv.MatchID, &cv.MatchedAt, &cv.Peer.ID, &cv.Peer.FirstName, &cv.Peer.MainPhotoID,
			&lmID, &lmSender, &lmBody, &lmAt, &cv.UnreadCount, &cv.Activity); err != nil {
			return nil, err
		}
		if lmID != nil {
			cv.LastMessage = &MessagePreview{ID: *lmID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		out = append(out, cv)
	}
	return out, rows.Err()
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// access verifies the caller participates in an active, unblocked
// conversation and returns the other participant.
const accessSQL = `
	SELECT peer.user_id, me.last_read_message_id, COALESCE(me.hidden_before_message_id, 0), peer.last_read_message_id
	FROM conversation_participants me
	JOIN conversation_participants peer ON peer.conversation_id = me.conversation_id AND peer.user_id <> me.user_id
	JOIN conversations c ON c.id = me.conversation_id
	JOIN matches m ON m.id = c.match_id AND m.ended_at IS NULL
	WHERE me.conversation_id = $1 AND me.user_id = $2
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = peer.user_id)
	                                             OR (b.blocker_id = peer.user_id AND b.blocked_id = me.user_id))`

type access struct {
	PeerID                   string
	MyRead, Hidden, PeerRead int64
}

func (r *PGRepository) access(ctx context.Context, q pgx.Tx, userID, convID string, lock bool) (access, error) {
	var a access
	sql := accessSQL
	if lock {
		sql += ` FOR SHARE OF m`
	}
	err := q.QueryRow(ctx, sql, convID, userID).Scan(&a.PeerID, &a.MyRead, &a.Hidden, &a.PeerRead)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrForbidden
	}
	return a, mapErr(err)
}

func (r *PGRepository) Messages(ctx context.Context, userID, convID string, beforeID int64, limit int) ([]Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	a, err := r.access(ctx, tx, userID, convID, false)
	if err != nil {
		return nil, err
	}
	if beforeID <= 0 {
		beforeID = 1<<62 - 1
	}
	rows, err := tx.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1 AND id < $2 AND id > $3
		ORDER BY id DESC LIMIT $4`, convID, beforeID, a.Hidden, limit)
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
		m.Read = m.SenderID == userID && m.ID <= a.PeerRead
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Send(ctx context.Context, userID, convID, body string) (Message, string, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	// FOR SHARE on the match row makes a concurrent unmatch/block wait for us
	// (or us wait for it), so no message lands in an ended conversation.
	a, err := r.access(ctx, tx, userID, convID, true)
	if err != nil {
		return Message{}, "", "", err
	}
	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at`, convID, userID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if err != nil {
		return Message{}, "", "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, m.CreatedAt); err != nil {
		return Message{}, "", "", err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_message_id = $3
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID, m.ID); err != nil {
		return Message{}, "", "", err
	}
	var name string
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT first_name FROM profiles WHERE user_id = $1), '')`, userID).Scan(&name); err != nil {
		return Message{}, "", "", err
	}
	return m, name, a.PeerID, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, convID string) (string, int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	a, err := r.access(ctx, tx, userID, convID, false)
	if err != nil {
		return "", 0, err
	}
	var upTo int64
	err = tx.QueryRow(ctx, `
		UPDATE conversation_participants cp
		SET last_read_message_id = GREATEST(cp.last_read_message_id, COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = $1), 0))
		WHERE cp.conversation_id = $1 AND cp.user_id = $2
		RETURNING cp.last_read_message_id`, convID, userID).Scan(&upTo)
	if err != nil {
		return "", 0, err
	}
	return a.PeerID, upTo, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, userID, convID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_before_message_id =
			(SELECT MAX(id) FROM messages WHERE conversation_id = $1)
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}
