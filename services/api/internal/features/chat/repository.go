package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNotParticipant = errors.New("chat: not a participant")

// CreateConversation opens the conversation of a new match inside the caller's transaction.
func CreateConversation(ctx context.Context, q db.Querier, matchID, userA, userB string) (string, error) {
	var id string
	if err := q.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id RETURNING id`, matchID).Scan(&id); err != nil {
		return "", err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING`, id, userA, userB)
	return id, err
}

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type convRow struct {
	Conversation
	photoID *string
}

type convFilter struct {
	convID *string
	cursor *cursor
	limit  int
}

type cursor struct {
	at time.Time
	id string
}

func (r *Repository) conversations(ctx context.Context, viewerID string, f convFilter) ([]convRow, error) {
	var curAt time.Time
	var curID *string
	if f.cursor != nil {
		curAt, curID = f.cursor.at, &f.cursor.id
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, c.created_at, c.last_message_at,
		       o.user_id, pr.first_name,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = o.user_id ORDER BY ph.position LIMIT 1),
		       lm.body, lm.sender_id, lm.created_at,
		       (SELECT COUNT(*) FROM messages m WHERE m.conversation_id = c.id AND m.sender_id <> me.user_id AND m.created_at > me.last_read_at)
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
		JOIN profiles pr ON pr.user_id = o.user_id
		LEFT JOIN LATERAL (
			SELECT body, sender_id, created_at FROM messages
			WHERE conversation_id = c.id AND created_at > COALESCE(me.hidden_at, '-infinity'::timestamptz)
			ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND ($2::uuid IS NULL OR c.id = $2::uuid)
		  AND ($2::uuid IS NOT NULL OR me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		  AND ($3::uuid IS NULL OR (c.last_message_at, c.id) < ($4::timestamptz, $3::uuid))
		ORDER BY c.last_message_at DESC, c.id DESC
		LIMIT $5`, viewerID, f.convID, curID, curAt, f.limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []convRow{}
	for rows.Next() {
		var cr convRow
		var body, sender *string
		var sentAt *time.Time
		if err := rows.Scan(&cr.ID, &cr.MatchID, &cr.CreatedAt, &cr.LastActivityAt, &cr.User.UserID, &cr.User.FirstName,
			&cr.photoID, &body, &sender, &sentAt, &cr.UnreadCount); err != nil {
			return nil, err
		}
		if body != nil {
			cr.LastMessage = &LastMessage{Body: *body, SenderID: *sender, CreatedAt: *sentAt}
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}

func (r *Repository) Hide(ctx context.Context, convID, userID string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errNotParticipant
	}
	return nil
}

func (r *Repository) IsParticipant(ctx context.Context, convID, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)`, convID, userID).Scan(&ok)
	return ok, err
}

func (r *Repository) Messages(ctx context.Context, convID, userID string, before *string, limit int) ([]Message, *time.Time, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at
		FROM messages m
		JOIN conversation_participants me ON me.conversation_id = m.conversation_id AND me.user_id = $2
		WHERE m.conversation_id = $1
		  AND m.created_at > COALESCE(me.hidden_at, '-infinity'::timestamptz)
		  AND ($3::uuid IS NULL OR (m.created_at, m.id) < (SELECT b.created_at, b.id FROM messages b WHERE b.id = $3::uuid AND b.conversation_id = $1))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4`, convID, userID, before, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var readAt *time.Time
	err = r.pool.QueryRow(ctx, `SELECT last_read_at FROM conversation_participants WHERE conversation_id = $1 AND user_id <> $2`, convID, userID).Scan(&readAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	return out, readAt, nil
}

// Insert stores a message if sender belongs to the conversation, returning it with the other participant's id.
func (r *Repository) Insert(ctx context.Context, convID, senderID, body string) (Message, string, error) {
	var m Message
	var other string
	err := r.pool.QueryRow(ctx, `
		WITH p AS (
			SELECT o.user_id AS other FROM conversation_participants me
			JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
			WHERE me.conversation_id = $1 AND me.user_id = $2
		), ins AS (
			INSERT INTO messages (conversation_id, sender_id, body) SELECT $1, $2, $3 FROM p RETURNING id, created_at
		), upd AS (
			UPDATE conversations SET last_message_at = (SELECT created_at FROM ins) WHERE id = $1 AND EXISTS (SELECT 1 FROM ins) RETURNING 1
		), rd AS (
			UPDATE conversation_participants SET last_read_at = (SELECT created_at FROM ins)
			WHERE conversation_id = $1 AND user_id = $2 AND EXISTS (SELECT 1 FROM ins) RETURNING 1
		)
		SELECT ins.id, ins.created_at, p.other FROM ins, p`, convID, senderID, body).Scan(&m.ID, &m.CreatedAt, &other)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, "", errNotParticipant
	}
	m.ConversationID, m.SenderID, m.Body = convID, senderID, body
	return m, other, err
}

func (r *Repository) MarkRead(ctx context.Context, convID, userID string) (string, time.Time, error) {
	var readAt time.Time
	var other string
	err := r.pool.QueryRow(ctx, `
		WITH u AS (
			UPDATE conversation_participants SET last_read_at = NOW() WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
		)
		SELECT u.last_read_at, o.user_id FROM u
		JOIN conversation_participants o ON o.conversation_id = $1 AND o.user_id <> $2`, convID, userID).Scan(&readAt, &other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, errNotParticipant
	}
	return other, readAt, err
}
