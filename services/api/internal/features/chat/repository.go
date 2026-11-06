package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error)
	UnreadTotal(ctx context.Context, userID string) (int, error)
	ListMessages(ctx context.Context, conversationID, userID string, before *time.Time, limit int) (MessagesResponse, error)
	Send(ctx context.Context, conversationID, senderID, body string) (SendResult, error)
	MarkRead(ctx context.Context, conversationID, userID string) (peerID string, err error)
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func mapErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, m.id, o.user_id, p.first_name, ph.id,
		       lm.body, lm.sender_id, lm.created_at,
		       (SELECT count(*) FROM messages x
		         WHERE x.conversation_id = c.id AND x.sender_id <> $1 AND x.created_at > me.last_read_at),
		       COALESCE(lm.created_at, m.created_at)
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> $1
		JOIN profiles p ON p.user_id = o.user_id
		LEFT JOIN photos ph ON ph.user_id = o.user_id AND ph.position = 0
		LEFT JOIN LATERAL (
		  SELECT body, sender_id, created_at FROM messages
		  WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND (me.hidden_at IS NULL OR lm.created_at > me.hidden_at)
		  AND ($2::timestamptz IS NULL OR COALESCE(lm.created_at, m.created_at) < $2)
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, c.id DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Conversation, 0, limit)
	for rows.Next() {
		var c Conversation
		var body, sender *string
		var at *time.Time
		if err := rows.Scan(&c.ID, &c.MatchID, &c.Peer.ID, &c.Peer.FirstName, &c.Peer.MainPhotoID,
			&body, &sender, &at, &c.UnreadCount, &c.ActivityAt); err != nil {
			return nil, err
		}
		if body != nil && sender != nil && at != nil {
			c.LastMessage = &LastMessage{Body: *body, SenderID: *sender, CreatedAt: *at}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT count(*) FROM conversation_participants me
		JOIN messages x ON x.conversation_id = me.conversation_id
		WHERE me.user_id = $1 AND x.sender_id <> $1 AND x.created_at > me.last_read_at
	`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID, userID string, before *time.Time, limit int) (MessagesResponse, error) {
	var resp MessagesResponse
	// Membership check: a non-participant gets "not found", never the messages.
	err := r.db.QueryRow(ctx, `
		SELECT o.user_id, p.first_name, ph.id, o.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		JOIN profiles p ON p.user_id = o.user_id
		LEFT JOIN photos ph ON ph.user_id = o.user_id AND ph.position = 0
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&resp.Peer.ID, &resp.Peer.FirstName, &resp.Peer.MainPhotoID, &resp.PeerLastReadAt)
	if err != nil {
		return resp, mapErr(err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1 AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC LIMIT $3
	`, conversationID, before, limit+1)
	if err != nil {
		return resp, err
	}
	defer rows.Close()
	resp.Messages = make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return resp, err
		}
		resp.Messages = append(resp.Messages, m)
	}
	if rows.Err() != nil {
		return resp, rows.Err()
	}
	if len(resp.Messages) > limit {
		resp.Messages = resp.Messages[:limit]
		next := resp.Messages[len(resp.Messages)-1].CreatedAt
		resp.NextBefore = &next
	}
	return resp, nil
}

func (r *PGRepository) Send(ctx context.Context, conversationID, senderID, body string) (SendResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SendResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var res SendResult
	var peerLastRead time.Time
	err = tx.QueryRow(ctx, `
		SELECT o.user_id, (SELECT first_name FROM profiles WHERE user_id = me.user_id), o.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		FOR UPDATE OF o
	`, conversationID, senderID).Scan(&res.RecipientID, &res.SenderName, &peerLastRead)
	if err != nil {
		return SendResult{}, mapErr(err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM messages WHERE conversation_id = $1 AND sender_id = $2 AND created_at > $3
	`, conversationID, senderID, peerLastRead).Scan(&res.RecipientUnread); err != nil {
		return SendResult{}, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at
	`, conversationID, senderID, body).Scan(&res.Message.ID, &res.Message.ConversationID, &res.Message.SenderID, &res.Message.Body, &res.Message.CreatedAt); err != nil {
		return SendResult{}, err
	}
	// Sending counts as reading; a new message resurfaces a locally deleted conversation.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NULL,
		       last_read_at = CASE WHEN user_id = $2 THEN $3 ELSE last_read_at END
		WHERE conversation_id = $1
	`, conversationID, senderID, res.Message.CreatedAt); err != nil {
		return SendResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = $2 WHERE id = $1`, conversationID, res.Message.CreatedAt); err != nil {
		return SendResult{}, err
	}
	return res, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (string, error) {
	var peer string
	err := r.db.QueryRow(ctx, `
		WITH upd AS (
		  UPDATE conversation_participants SET last_read_at = clock_timestamp()
		  WHERE conversation_id = $1 AND user_id = $2 RETURNING conversation_id
		)
		SELECT o.user_id FROM upd JOIN conversation_participants o ON o.conversation_id = upd.conversation_id AND o.user_id <> $2
	`, conversationID, userID).Scan(&peer)
	return peer, mapErr(err)
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
