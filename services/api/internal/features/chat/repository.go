package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Conversation, error)
	ListMessages(ctx context.Context, userID, convID string, limit int, cur *httpx.Cursor) (MessagePage, error)
	// Send insere le message; renvoie aussi l'identifiant du destinataire.
	Send(ctx context.Context, userID, convID, body string) (Message, string, error)
	MarkRead(ctx context.Context, userID, convID string) (otherID string, readAt time.Time, err error)
	Clear(ctx context.Context, userID, convID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Conversation, error) {
	var curAt *time.Time
	var curID *string
	if cur != nil {
		curAt, curID = &cur.At, &cur.ID
	}
	// Une seule requete (pas de N+1): dernier message et non-lus par LATERAL.
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, o.user_id, p.first_name,
		       (SELECT id::text FROM photos WHERE user_id = o.user_id ORDER BY position LIMIT 1),
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages m
		         WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id
		           AND m.created_at > me.last_read_at
		           AND m.created_at > COALESCE(me.cleared_at, '-infinity')),
		       o.last_read_at
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
		JOIN profiles p ON p.user_id = o.user_id
		JOIN LATERAL (
		  SELECT id, sender_id, body, created_at FROM messages
		  WHERE conversation_id = c.id AND created_at > COALESCE(me.cleared_at, '-infinity')
		  ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = o.user_id) OR (b.blocker_id = o.user_id AND b.blocked_id = $1))
		  AND ($3::timestamptz IS NULL OR (lm.created_at, c.id) < ($3::timestamptz, $4::uuid))
		ORDER BY lm.created_at DESC, c.id DESC
		LIMIT $2`, userID, limit, curAt, curID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var cv Conversation
		var lm Message
		var unread int64
		if err := rows.Scan(&cv.ID, &cv.MatchID, &cv.UserID, &cv.FirstName, &cv.PhotoID,
			&lm.ID, &lm.SenderID, &lm.Body, &lm.CreatedAt, &unread, &cv.OtherReadAt); err != nil {
			return nil, err
		}
		lm.ConversationID = cv.ID
		cv.LastMessage = &lm
		cv.UnreadCount = int(unread)
		cv.UpdatedAt = lm.CreatedAt
		out = append(out, cv)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, convID string, limit int, cur *httpx.Cursor) (MessagePage, error) {
	page := MessagePage{Items: []Message{}}
	var clearedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT me.cleared_at, o.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, userID).Scan(&clearedAt, &page.OtherReadAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return page, ErrNotFound
		}
		return page, err
	}
	var curAt *time.Time
	var curID *string
	if cur != nil {
		curAt, curID = &cur.At, &cur.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1
		  AND ($3::timestamptz IS NULL OR created_at > $3)
		  AND ($4::timestamptz IS NULL OR (created_at, id) < ($4::timestamptz, $5::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $2`, convID, limit, clearedAt, curAt, curID)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return page, err
		}
		page.Items = append(page.Items, m)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) == limit {
		last := page.Items[len(page.Items)-1]
		page.NextCursor = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (r *PGRepository) Send(ctx context.Context, userID, convID, body string) (Message, string, error) {
	var m Message
	var otherID string
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return m, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var blocked bool
	err = tx.QueryRow(ctx, `
		SELECT o.user_id,
		       EXISTS (SELECT 1 FROM blocks b
		         WHERE (b.blocker_id = me.user_id AND b.blocked_id = o.user_id) OR (b.blocker_id = o.user_id AND b.blocked_id = me.user_id))
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, userID).Scan(&otherID, &blocked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m, "", ErrNotFound
		}
		return m, "", err
	}
	if blocked {
		return m, "", ErrBlocked
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1,$2,$3)
		RETURNING id, conversation_id, sender_id, body, created_at`, convID, userID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return m, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, m.CreatedAt); err != nil {
		return m, "", err
	}
	// Envoyer implique avoir lu; la conversation reapparait chez l'autre si elle avait ete supprimee localement (cleared_at < message).
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`,
		convID, userID, m.CreatedAt); err != nil {
		return m, "", err
	}
	// Une seule notification non lue par conversation (pas de flood).
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id)
		SELECT $1, 'message', $2, $3
		WHERE NOT EXISTS (SELECT 1 FROM notifications
		                  WHERE user_id = $1 AND type = 'message' AND ref_id = $3 AND read_at IS NULL)`,
		otherID, userID, convID); err != nil {
		return m, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return m, "", err
	}
	return m, otherID, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, convID string) (string, time.Time, error) {
	var otherID string
	var readAt time.Time
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", readAt, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx, `
		UPDATE conversation_participants me SET last_read_at = NOW()
		WHERE me.conversation_id = $1 AND me.user_id = $2
		RETURNING me.last_read_at,
		  (SELECT user_id::text FROM conversation_participants WHERE conversation_id = $1 AND user_id <> $2)`,
		convID, userID).Scan(&readAt, &otherID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", readAt, ErrNotFound
		}
		return "", readAt, err
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND ref_id = $2 AND read_at IS NULL`, userID, convID); err != nil {
		return "", readAt, err
	}
	return otherID, readAt, tx.Commit(ctx)
}

func (r *PGRepository) Clear(ctx context.Context, userID, convID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
