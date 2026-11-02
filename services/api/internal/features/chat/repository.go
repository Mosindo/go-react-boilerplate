package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("conversation not found")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// membership returns the partner id and the user's own hidden_at when userID
// belongs to the match. Every chat operation starts here, so a user can never
// reach a conversation they are not part of.
func (r *Repository) membership(ctx context.Context, matchID, userID string) (partnerID string, hiddenAt *time.Time, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT CASE WHEN user_a_id = $2 THEN user_b_id ELSE user_a_id END,
		       CASE WHEN user_a_id = $2 THEN a_hidden_at ELSE b_hidden_at END
		FROM matches WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)`, matchID, userID).Scan(&partnerID, &hiddenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	return
}

func (r *Repository) List(ctx context.Context, userID string, limit int, before *time.Time) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, pr.user_id, pr.first_name,
		       (SELECT id FROM photos ph WHERE ph.user_id = pr.user_id ORDER BY position LIMIT 1),
		       m.created_at, lm.body, lm.sender_id, lm.created_at,
		       (SELECT COUNT(*) FROM messages x WHERE x.match_id = m.id AND x.sender_id <> $1 AND x.read_at IS NULL),
		       COALESCE(m.last_message_at, m.created_at) AS activity
		FROM matches m
		JOIN profiles pr ON pr.user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		LEFT JOIN LATERAL (SELECT body, sender_id, created_at FROM messages WHERE match_id = m.id ORDER BY created_at DESC, id DESC LIMIT 1) lm ON TRUE
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1)
		  AND (CASE WHEN m.user_a_id = $1 THEN m.a_hidden_at ELSE m.b_hidden_at END IS NULL
		       OR m.last_message_at > CASE WHEN m.user_a_id = $1 THEN m.a_hidden_at ELSE m.b_hidden_at END)
		  AND ($2::timestamptz IS NULL OR COALESCE(m.last_message_at, m.created_at) < $2)
		ORDER BY activity DESC, m.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var c Conversation
		var photoID *string
		var body, sender *string
		var at *time.Time
		if err := rows.Scan(&c.MatchID, &c.Partner.UserID, &c.Partner.FirstName, &photoID, &c.CreatedAt, &body, &sender, &at, &c.UnreadCount, &c.ActivityAt); err != nil {
			return nil, err
		}
		if photoID != nil {
			u := "/photos/" + *photoID
			c.Partner.PhotoURL = &u
		}
		if body != nil {
			c.LastMessage = &LastMessage{Body: *body, SenderID: *sender, CreatedAt: *at}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Messages(ctx context.Context, matchID, userID string, limit int, beforeID *int64) ([]Message, error) {
	_, hidden, err := r.membership(ctx, matchID, userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, match_id, sender_id, body, created_at, read_at FROM messages
		WHERE match_id = $1 AND ($2::bigint IS NULL OR id < $2) AND ($3::timestamptz IS NULL OR created_at > $3)
		ORDER BY id DESC LIMIT $4`, matchID, beforeID, hidden, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Send stores a message and returns it with the recipient id.
func (r *Repository) Send(ctx context.Context, matchID, userID, body string) (Message, string, error) {
	partnerID, _, err := r.membership(ctx, matchID, userID)
	if err != nil {
		return Message{}, "", err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, match_id, sender_id, body, created_at, read_at`, matchID, userID, body).
		Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
	if err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	return m, partnerID, tx.Commit(ctx)
}

// MarkRead marks the partner's messages as read and returns how many changed.
func (r *Repository) MarkRead(ctx context.Context, matchID, userID string) (partnerID string, n int64, err error) {
	partnerID, _, err = r.membership(ctx, matchID, userID)
	if err != nil {
		return "", 0, err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE messages SET read_at = NOW() WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL`, matchID, userID)
	if err != nil {
		return "", 0, err
	}
	_, err = r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'matchId' = $2`, userID, matchID)
	return partnerID, tag.RowsAffected(), err
}

// Hide removes the conversation from the user's inbox only. It comes back if
// the partner writes again; older messages stay hidden for this user.
func (r *Repository) Hide(ctx context.Context, matchID, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE matches SET
		  a_hidden_at = CASE WHEN user_a_id = $2 THEN NOW() ELSE a_hidden_at END,
		  b_hidden_at = CASE WHEN user_b_id = $2 THEN NOW() ELSE b_hidden_at END
		WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Unmatch deletes the match and its messages for both users.
func (r *Repository) Unmatch(ctx context.Context, matchID, userID string) (partnerID string, err error) {
	partnerID, _, err = r.membership(ctx, matchID, userID)
	if err != nil {
		return "", err
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM matches WHERE id = $1`, matchID)
	return partnerID, err
}
