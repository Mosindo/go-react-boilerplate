package conversations

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("conversation not found")
)

// Access is the caller's view of a match they belong to.
type Access struct {
	MatchID string
	OtherID string
	// HiddenAt is when the caller removed the conversation from their list, if ever.
	HiddenAt *time.Time
}

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error)
	Access(ctx context.Context, userID, matchID string) (Access, error)
	Messages(ctx context.Context, matchID string, after *time.Time, before *time.Time, limit int) ([]Message, error)
	Insert(ctx context.Context, userID, matchID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, matchID string) (int64, error)
	Hide(ctx context.Context, userID, matchID string) error
	Unmatch(ctx context.Context, userID, matchID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const notBlocked = `NOT EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_id = $1 AND b.blocked_id = o.user_id)
	   OR (b.blocker_id = o.user_id AND b.blocked_id = $1))`

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, o.user_id, pr.first_name,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = o.user_id ORDER BY ph.position LIMIT 1),
		       m.created_at, COALESCE(m.last_message_at, m.created_at),
		       lm.body, lm.sender_id, lm.created_at,
		       COALESCE(u.n, 0)
		FROM matches m
		CROSS JOIN LATERAL (SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS user_id,
		                           CASE WHEN m.user_a = $1 THEN m.hidden_a_at ELSE m.hidden_b_at END AS hidden_at) o
		JOIN profiles pr ON pr.user_id = o.user_id
		LEFT JOIN LATERAL (
		  SELECT body, sender_id, created_at FROM chat_messages cm
		  WHERE cm.match_id = m.id AND (o.hidden_at IS NULL OR cm.created_at > o.hidden_at)
		  ORDER BY cm.created_at DESC, cm.id DESC LIMIT 1) lm ON true
		LEFT JOIN LATERAL (
		  SELECT count(*)::int AS n FROM chat_messages cm
		  WHERE cm.match_id = m.id AND cm.sender_id <> $1 AND cm.read_at IS NULL
		    AND (o.hidden_at IS NULL OR cm.created_at > o.hidden_at)) u ON true
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND `+notBlocked+`
		  AND (o.hidden_at IS NULL OR m.last_message_at > o.hidden_at)
		  AND ($2::timestamptz IS NULL OR COALESCE(m.last_message_at, m.created_at) < $2)
		ORDER BY COALESCE(m.last_message_at, m.created_at) DESC, m.id DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Conversation, 0, limit)
	for rows.Next() {
		var c Conversation
		var photoID *string
		var body, sender *string
		var sentAt *time.Time
		if err := rows.Scan(&c.MatchID, &c.User.ID, &c.User.FirstName, &photoID, &c.MatchedAt, &c.SortKey,
			&body, &sender, &sentAt, &c.UnreadCount); err != nil {
			return nil, err
		}
		if photoID != nil {
			c.User.PhotoURL = "/photos/" + *photoID
		}
		if body != nil && sender != nil && sentAt != nil {
			c.LastMessage = &LastMessage{Body: *body, SenderID: *sender, CreatedAt: *sentAt}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Access resolves a match for a participant. It answers ErrNotFound both when the match does
// not exist and when the caller is not part of it, so ids cannot be probed.
func (r *PGRepository) Access(ctx context.Context, userID, matchID string) (Access, error) {
	a := Access{MatchID: matchID}
	err := r.pool.QueryRow(ctx, `
		SELECT o.user_id, o.hidden_at
		FROM matches m
		CROSS JOIN LATERAL (SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS user_id,
		                           CASE WHEN m.user_a = $1 THEN m.hidden_a_at ELSE m.hidden_b_at END AS hidden_at) o
		WHERE m.id = $2 AND (m.user_a = $1 OR m.user_b = $1)
		  AND `+notBlocked, userID, matchID).Scan(&a.OtherID, &a.HiddenAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Access{}, ErrNotFound
		}
		return Access{}, err
	}
	return a, nil
}

func (r *PGRepository) Messages(ctx context.Context, matchID string, after, before *time.Time, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, match_id, sender_id, body, created_at, read_at
		FROM chat_messages
		WHERE match_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, matchID, after, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Insert stores a message and bumps the match. A conversation a user removed from their list
// resurfaces on its own because last_message_at moves past their hidden_at cutoff; the cutoff
// itself is kept so earlier history stays hidden.
func (r *PGRepository) Insert(ctx context.Context, userID, matchID, body string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	m := Message{MatchID: matchID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO chat_messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, matchID, userID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, err
	}
	return m, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, matchID string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE chat_messages SET read_at = NOW()
		WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, matchID, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *PGRepository) Hide(ctx context.Context, userID, matchID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE matches
		SET hidden_a_at = CASE WHEN user_a = $1 THEN NOW() ELSE hidden_a_at END,
		    hidden_b_at = CASE WHEN user_b = $1 THEN NOW() ELSE hidden_b_at END
		WHERE id = $2 AND (user_a = $1 OR user_b = $1)
	`, userID, matchID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
	`, userID, matchID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
