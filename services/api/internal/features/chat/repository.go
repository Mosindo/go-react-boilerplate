package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotMember = errors.New("not a member of this conversation")

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, after *Cursor) ([]conversationRow, error)
	OtherMember(ctx context.Context, matchID, userID string) (otherUserID string, otherLastReadAt *time.Time, err error)
	ListMessages(ctx context.Context, matchID string, limit int, before *Cursor) ([]Message, error)
	InsertMessage(ctx context.Context, matchID, senderID, body string) (Message, string, error)
	MarkRead(ctx context.Context, matchID, userID string) (time.Time, error)
	Hide(ctx context.Context, matchID, userID string) error
	FirstName(ctx context.Context, userID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ListConversations returns the user's matches ordered by latest activity.
// Conversations hidden by the user stay hidden until a newer message arrives.
// Every sub-query is bounded by an index on (match_id, created_at).
func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, after *Cursor) ([]conversationRow, error) {
	var afterAt *time.Time
	var afterID *string
	if after != nil {
		afterAt, afterID = &after.At, &after.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id,
		       other.user_id,
		       m.created_at,
		       COALESCE(m.last_message_at, m.created_at) AS activity_at,
		       last.id, last.sender_id, last.body, last.created_at,
		       (SELECT COUNT(*) FROM messages u
		         WHERE u.match_id = m.id AND u.sender_id <> $1
		           AND u.created_at > COALESCE(me.last_read_at, '-infinity'::timestamptz)) AS unread,
		       other.last_read_at
		FROM match_members me
		JOIN matches m ON m.id = me.match_id
		JOIN match_members other ON other.match_id = m.id AND other.user_id <> me.user_id
		LEFT JOIN LATERAL (
			SELECT id, sender_id, body, created_at FROM messages
			WHERE match_id = m.id ORDER BY created_at DESC, id DESC LIMIT 1
		) last ON TRUE
		WHERE me.user_id = $1
		  AND (me.hidden_at IS NULL OR me.hidden_at < COALESCE(m.last_message_at, m.created_at))
		  AND ($3::timestamptz IS NULL OR (COALESCE(m.last_message_at, m.created_at), m.id) < ($3::timestamptz, $4::uuid))
		ORDER BY activity_at DESC, m.id DESC
		LIMIT $2
	`, userID, limit, afterAt, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]conversationRow, 0, limit)
	for rows.Next() {
		var c conversationRow
		var lastID, lastSender, lastBody *string
		var lastAt *time.Time
		if err := rows.Scan(&c.MatchID, &c.OtherUserID, &c.CreatedAt, &c.ActivityAt,
			&lastID, &lastSender, &lastBody, &lastAt, &c.UnreadCount, &c.OtherLastReadAt); err != nil {
			return nil, err
		}
		if lastID != nil {
			c.LastMessage = &Message{ID: *lastID, MatchID: c.MatchID, SenderID: *lastSender, Body: *lastBody, CreatedAt: *lastAt}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// OtherMember enforces membership: it fails with ErrNotMember unless userID
// belongs to the match.
func (r *PGRepository) OtherMember(ctx context.Context, matchID, userID string) (string, *time.Time, error) {
	var other string
	var readAt *time.Time
	err := r.db.QueryRow(ctx, `
		SELECT o.user_id, o.last_read_at
		FROM match_members me
		JOIN match_members o ON o.match_id = me.match_id AND o.user_id <> me.user_id
		WHERE me.match_id = $1 AND me.user_id = $2
	`, matchID, userID).Scan(&other, &readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotMember
	}
	return other, readAt, err
}

func (r *PGRepository) ListMessages(ctx context.Context, matchID string, limit int, before *Cursor) ([]Message, error) {
	var beforeAt *time.Time
	var beforeID *string
	if before != nil {
		beforeAt, beforeID = &before.At, &before.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, match_id, sender_id, body, created_at FROM messages
		WHERE match_id = $1
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3::timestamptz, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, matchID, limit, beforeAt, beforeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// InsertMessage writes a message after locking the match row, so a concurrent
// unmatch/block cannot interleave. It returns the recipient id.
func (r *PGRepository) InsertMessage(ctx context.Context, matchID, senderID, body string) (Message, string, error) {
	var msg Message
	var recipient string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var low, high string
		err := tx.QueryRow(ctx, `
			SELECT user_low_id, user_high_id FROM matches
			WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
			FOR UPDATE
		`, matchID, senderID).Scan(&low, &high)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotMember
		}
		if err != nil {
			return err
		}
		recipient = low
		if low == senderID {
			recipient = high
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
			RETURNING id, match_id, sender_id, body, created_at
		`, matchID, senderID, body).Scan(&msg.ID, &msg.MatchID, &msg.SenderID, &msg.Body, &msg.CreatedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, msg.CreatedAt); err != nil {
			return err
		}
		// Sending implies having read the conversation.
		_, err = tx.Exec(ctx, `
			UPDATE match_members SET last_read_at = $3 WHERE match_id = $1 AND user_id = $2
		`, matchID, senderID, msg.CreatedAt)
		return err
	})
	return msg, recipient, err
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, userID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE match_members SET last_read_at = clock_timestamp()
		WHERE match_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, matchID, userID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotMember
	}
	return at, err
}

func (r *PGRepository) Hide(ctx context.Context, matchID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE match_members SET hidden_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE match_id = $1 AND user_id = $2
	`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}
	return nil
}

func (r *PGRepository) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	return name, err
}
