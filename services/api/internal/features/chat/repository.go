package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type Repository interface {
	// Participant returns the other person of a match, or ErrNotFound when userID
	// is not one of the two members. Every chat operation starts here.
	Participant(ctx context.Context, userID, matchID string) (otherID string, err error)
	Conversations(ctx context.Context, userID string, limit, offset int) ([]StoredConversation, error)
	Messages(ctx context.Context, userID, matchID, beforeID string, limit int) ([]Message, error)
	Insert(ctx context.Context, matchID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, matchID string) (int, error)
	Clear(ctx context.Context, userID, matchID string) error
	UnreadTotal(ctx context.Context, userID string) (int, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) Participant(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		SELECT (CASE WHEN user_a = $1 THEN user_b ELSE user_a END)::text
		FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
	`, userID, matchID).Scan(&other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return other, nil
}

func (r *PGRepository) Conversations(ctx context.Context, userID string, limit, offset int) ([]StoredConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text, (CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END)::text, m.created_at, m.activity_at,
		       lm.id::text, lm.sender_id::text, lm.body, lm.created_at, lm.read_at,
		       (SELECT COUNT(*) FROM messages x
		        WHERE x.match_id = m.id AND x.sender_id <> $1 AND x.read_at IS NULL
		          AND x.created_at > COALESCE(cs.cleared_at, '-infinity'))
		FROM matches m
		LEFT JOIN conversation_states cs ON cs.match_id = m.id AND cs.user_id = $1
		LEFT JOIN LATERAL (
		  SELECT id, sender_id, body, created_at, read_at FROM messages
		  WHERE match_id = m.id AND created_at > COALESCE(cs.cleared_at, '-infinity')
		  ORDER BY created_at DESC, id DESC LIMIT 1
		) lm ON TRUE
		WHERE m.user_a = $1 OR m.user_b = $1
		ORDER BY m.activity_at DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredConversation
	for rows.Next() {
		var c StoredConversation
		var msgID, sender, body *string
		var msgAt *time.Time
		var readAt *time.Time
		if err := rows.Scan(&c.MatchID, &c.OtherID, &c.CreatedAt, &c.ActivityAt, &msgID, &sender, &body, &msgAt, &readAt, &c.Unread); err != nil {
			return nil, err
		}
		if msgID != nil {
			c.LastMessage = &Message{ID: *msgID, MatchID: c.MatchID, SenderID: *sender, Body: *body, CreatedAt: *msgAt, ReadAt: readAt}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Messages returns up to limit messages older than beforeID (or the newest ones),
// newest first, hiding anything the user cleared. Callers reverse for display.
func (r *PGRepository) Messages(ctx context.Context, userID, matchID, beforeID string, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT msg.id::text, msg.match_id::text, msg.sender_id::text, msg.body, msg.created_at, msg.read_at
		FROM messages msg
		LEFT JOIN conversation_states cs ON cs.match_id = msg.match_id AND cs.user_id = $1
		WHERE msg.match_id = $2
		  AND msg.created_at > COALESCE(cs.cleared_at, '-infinity')
		  AND ($3::uuid IS NULL OR (msg.created_at, msg.id) < (SELECT created_at, id FROM messages WHERE id = $3 AND match_id = $2))
		ORDER BY msg.created_at DESC, msg.id DESC
		LIMIT $4
	`, userID, matchID, nullable(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (r *PGRepository) Insert(ctx context.Context, matchID, senderID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	m := Message{MatchID: matchID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id::text, created_at
	`, matchID, senderID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET activity_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, matchID string) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE messages SET read_at = NOW()
		WHERE match_id = $2 AND sender_id <> $1 AND read_at IS NULL
	`, userID, matchID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *PGRepository) Clear(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO conversation_states (match_id, user_id, cleared_at) VALUES ($2, $1, NOW())
		ON CONFLICT (match_id, user_id) DO UPDATE SET cleared_at = NOW()
	`, userID, matchID)
	return err
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM messages x
		JOIN matches m ON m.id = x.match_id AND (m.user_a = $1 OR m.user_b = $1)
		LEFT JOIN conversation_states cs ON cs.match_id = m.id AND cs.user_id = $1
		WHERE x.sender_id <> $1 AND x.read_at IS NULL AND x.created_at > COALESCE(cs.cleared_at, '-infinity')
	`, userID).Scan(&n)
	return n, err
}
