package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Membership(ctx context.Context, conversationID, userID string) (Membership, error)
	ListConversations(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]StoredSummary, error)
	ConversationCursor(ctx context.Context, userID, conversationID string) (time.Time, error)
	ListMessages(ctx context.Context, conversationID string, after *time.Time, before *time.Time, beforeID string, limit int) ([]StoredMessage, error)
	MessageCursor(ctx context.Context, conversationID, messageID string) (time.Time, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (StoredMessage, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Membership(ctx context.Context, conversationID, userID string) (Membership, error) {
	m := Membership{ConversationID: conversationID}
	err := r.db.QueryRow(ctx, `
		SELECT c.match_id, partner.user_id, me.cleared_at, partner.last_read_at, mt.created_at,
		       mt.unmatched_at IS NULL AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = me.user_id AND b.blocked_id = partner.user_id)
				   OR (b.blocker_id = partner.user_id AND b.blocked_id = me.user_id)
		       )
		FROM conversations c
		JOIN matches mt ON mt.id = c.match_id
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $2
		JOIN conversation_participants partner ON partner.conversation_id = c.id AND partner.user_id <> $2
		WHERE c.id = $1
	`, conversationID, userID).Scan(&m.MatchID, &m.PartnerID, &m.ClearedAt, &m.PartnerLastReadAt, &m.MatchedAt, &m.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrNotFound
	}
	return m, err
}

// ListConversations returns active conversations ordered by latest activity.
// Locally deleted conversations stay hidden until a new message arrives.
func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]StoredSummary, error) {
	rows, err := r.db.Query(ctx, `
		WITH mine AS (
			SELECT c.id, c.match_id, mt.created_at AS matched_at, me.cleared_at, me.last_read_at,
			       partner.user_id AS partner_id, partner.last_read_at AS partner_last_read_at,
			       COALESCE(c.last_message_at, mt.created_at) AS activity_at, c.last_message_at
			FROM conversation_participants me
			JOIN conversations c ON c.id = me.conversation_id
			JOIN matches mt ON mt.id = c.match_id AND mt.unmatched_at IS NULL
			JOIN conversation_participants partner ON partner.conversation_id = c.id AND partner.user_id <> me.user_id
			WHERE me.user_id = $1
			  AND (me.cleared_at IS NULL OR c.last_message_at > me.cleared_at)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = partner.user_id)
				   OR (b.blocker_id = partner.user_id AND b.blocked_id = $1)
			  )
		)
		SELECT m.id, m.match_id, m.partner_id, m.matched_at, m.activity_at, m.partner_last_read_at,
		       last.id, last.sender_id, last.body, last.created_at,
		       (SELECT COUNT(*) FROM messages um
		         WHERE um.conversation_id = m.id AND um.sender_id <> $1
		           AND (m.last_read_at IS NULL OR um.created_at > m.last_read_at)
		           AND (m.cleared_at IS NULL OR um.created_at > m.cleared_at)) AS unread
		FROM mine m
		LEFT JOIN LATERAL (
			SELECT msg.id, msg.sender_id, msg.body, msg.created_at
			FROM messages msg
			WHERE msg.conversation_id = m.id
			  AND (m.cleared_at IS NULL OR msg.created_at > m.cleared_at)
			ORDER BY msg.created_at DESC, msg.id DESC
			LIMIT 1
		) last ON TRUE
		WHERE $2::timestamptz IS NULL OR (m.activity_at, m.id) < ($2, $3::uuid)
		ORDER BY m.activity_at DESC, m.id DESC
		LIMIT $4
	`, userID, before, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredSummary
	for rows.Next() {
		var s StoredSummary
		var lastID, lastSender, lastBody *string
		var lastAt *time.Time
		if err := rows.Scan(&s.ConversationID, &s.MatchID, &s.PartnerID, &s.MatchedAt, &s.ActivityAt, &s.PartnerLastReadAt,
			&lastID, &lastSender, &lastBody, &lastAt, &s.UnreadCount); err != nil {
			return nil, err
		}
		if lastID != nil {
			s.LastMessage = &StoredMessage{ID: *lastID, ConversationID: s.ConversationID, SenderID: *lastSender, Body: *lastBody, CreatedAt: *lastAt}
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) ConversationCursor(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(c.last_message_at, mt.created_at)
		FROM conversations c
		JOIN matches mt ON mt.id = c.match_id
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $2
		WHERE c.id = $1
	`, conversationID, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrInvalidCursor
	}
	return t, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, after *time.Time, before *time.Time, beforeID string, limit int) ([]StoredMessage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5
	`, conversationID, after, before, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredMessage, error) {
		var m StoredMessage
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
		return m, err
	})
}

func (r *PGRepository) MessageCursor(ctx context.Context, conversationID, messageID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1 AND conversation_id = $2`, messageID, conversationID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrInvalidCursor
	}
	return t, err
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (StoredMessage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredMessage{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	m := StoredMessage{ConversationID: conversationID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, senderID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return StoredMessage{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations SET last_message_at = GREATEST(COALESCE(last_message_at, $2), $2) WHERE id = $1
	`, conversationID, m.CreatedAt); err != nil {
		return StoredMessage{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = GREATEST(COALESCE(last_read_at, $3), $3)
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return StoredMessage{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants
		SET last_read_at = GREATEST(COALESCE(last_read_at, clock_timestamp()), clock_timestamp())
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&t)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotFound
	}
	return t, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func nullUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
