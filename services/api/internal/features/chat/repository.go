package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("chat: conversation not found")

type storedConversation struct {
	ID, MatchID, OtherID, OtherName string
	OtherPhotoID                    *string
	LastBody, LastSender            *string
	LastAt                          *time.Time
	UpdatedAt                       time.Time
	Unread                          int
}

type Repository interface {
	ListConversations(ctx context.Context, me string, before *time.Time, limit int) ([]storedConversation, error)
	Messages(ctx context.Context, me, convID string, beforeID int64, limit int) (msgs []Message, otherLastRead time.Time, err error)
	Send(ctx context.Context, me, convID, body string) (msg Message, recipientID string, err error)
	MarkRead(ctx context.Context, me, convID string) (otherID string, err error)
	Hide(ctx context.Context, me, convID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, me string, before *time.Time, limit int) ([]storedConversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.match_id, o.user_id, pr.first_name,
		       (SELECT id FROM photos WHERE user_id = o.user_id AND position = 0),
		       lm.body, lm.sender_id::text, lm.created_at, c.last_message_at,
		       (SELECT COUNT(*) FROM messages m
		         WHERE m.conversation_id = c.id AND m.sender_id <> $1 AND m.created_at > mine.last_read_at
		           AND (mine.hidden_at IS NULL OR m.created_at > mine.hidden_at))
		FROM conversation_participants mine
		JOIN conversations c ON c.id = mine.conversation_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> $1
		JOIN profiles pr ON pr.user_id = o.user_id
		LEFT JOIN LATERAL (
			SELECT m.body, m.sender_id, m.created_at FROM messages m
			WHERE m.conversation_id = c.id AND (mine.hidden_at IS NULL OR m.created_at > mine.hidden_at)
			ORDER BY m.created_at DESC, m.id DESC LIMIT 1
		) lm ON TRUE
		WHERE mine.user_id = $1
		  AND (mine.hidden_at IS NULL OR c.last_message_at > mine.hidden_at)
		  AND ($2::timestamptz IS NULL OR c.last_message_at < $2)
		ORDER BY c.last_message_at DESC
		LIMIT $3
	`, me, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedConversation
	for rows.Next() {
		var c storedConversation
		if err := rows.Scan(&c.ID, &c.MatchID, &c.OtherID, &c.OtherName, &c.OtherPhotoID,
			&c.LastBody, &c.LastSender, &c.LastAt, &c.UpdatedAt, &c.Unread); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// membership returns the caller's hidden_at and the other member's id/last_read_at, or ErrNotFound
// when the caller is not a participant — the only authorisation check for every chat operation.
func membership(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, me, convID string) (hiddenAt *time.Time, otherID string, otherRead time.Time, err error) {
	err = q.QueryRow(ctx, `
		SELECT mine.hidden_at, o.user_id::text, o.last_read_at
		FROM conversation_participants mine
		JOIN conversation_participants o ON o.conversation_id = mine.conversation_id AND o.user_id <> mine.user_id
		WHERE mine.conversation_id = $1 AND mine.user_id = $2
	`, convID, me).Scan(&hiddenAt, &otherID, &otherRead)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

func (r *PGRepository) Messages(ctx context.Context, me, convID string, beforeID int64, limit int) ([]Message, time.Time, error) {
	hidden, _, otherRead, err := membership(ctx, r.db, me, convID)
	if err != nil {
		return nil, time.Time{}, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id::text, sender_id::text, body, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::bigint = 0 OR id < $2)
		  AND ($3::timestamptz IS NULL OR created_at > $3)
		ORDER BY id DESC
		LIMIT $4
	`, convID, beforeID, hidden, limit)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, time.Time{}, err
		}
		out = append(out, m)
	}
	return out, otherRead, rows.Err()
}

func (r *PGRepository) Send(ctx context.Context, me, convID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	_, otherID, _, err := membership(ctx, tx, me, convID)
	if err != nil {
		return Message{}, "", err
	}
	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id::text, sender_id::text, body, created_at
	`, convID, me, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// Writing implies having read everything so far.
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2`, convID, me, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, 'message', $2, $3)
		ON CONFLICT (user_id, type, ref_id) WHERE read_at IS NULL DO NOTHING
	`, otherID, me, convID); err != nil {
		return Message{}, "", err
	}
	return m, otherID, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, me, convID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	_, otherID, _, err := membership(ctx, tx, me, convID)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET last_read_at = NOW() WHERE conversation_id = $1 AND user_id = $2`, convID, me); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND type = 'message' AND ref_id = $2 AND read_at IS NULL`, me, convID); err != nil {
		return "", err
	}
	return otherID, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, me, convID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, convID, me)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
