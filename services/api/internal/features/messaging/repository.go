package messaging

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotMember = errors.New("not a member of this conversation")

// Membership describes the viewer's membership of a conversation.
type Membership struct {
	OtherUserID     string
	ClearedAt       *time.Time
	OtherLastReadAt *time.Time
}

type Repository interface {
	Membership(ctx context.Context, matchID, userID string) (Membership, error)
	List(ctx context.Context, matchID string, clearedAt *time.Time, beforeID string, limit int) ([]Message, error)
	Insert(ctx context.Context, matchID, senderID, recipientID, body string) (Message, error)
	MarkRead(ctx context.Context, matchID, userID string) (time.Time, error)
	Clear(ctx context.Context, matchID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Membership returns ErrRepositoryNotMember for strangers and for non-existent
// conversations alike, so existence is never revealed to non-members.
func (r *PGRepository) Membership(ctx context.Context, matchID, userID string) (Membership, error) {
	var m Membership
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN ma.user_a = $2::uuid THEN ma.user_b ELSE ma.user_a END,
		       me.cleared_at, ot.last_read_at
		FROM matches ma
		JOIN match_members me ON me.match_id = ma.id AND me.user_id = $2::uuid
		JOIN match_members ot ON ot.match_id = ma.id AND ot.user_id <> $2::uuid
		JOIN users ou ON ou.id = ot.user_id AND ou.disabled_at IS NULL
		WHERE ma.id = $1::uuid
	`, matchID, userID).Scan(&m.OtherUserID, &m.ClearedAt, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrRepositoryNotMember
	}
	return m, err
}

func (r *PGRepository) List(ctx context.Context, matchID string, clearedAt *time.Time, beforeID string, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, match_id, sender_id, body, created_at
		FROM chat_messages
		WHERE match_id = $1::uuid
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3 = '' OR (created_at, id) < (SELECT created_at, id FROM chat_messages WHERE id = NULLIF($3, '')::uuid AND match_id = $1::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, matchID, clearedAt, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Insert(ctx context.Context, matchID, senderID, recipientID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO chat_messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, match_id, sender_id, body, created_at
	`, matchID, senderID, body).Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE match_members SET last_read_at = $3 WHERE match_id = $1 AND user_id = $2
	`, matchID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// One unread "message" notification per conversation (partial unique index).
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_notifications (user_id, kind, actor_id, match_id)
		VALUES ($1, 'message', $2, $3)
		ON CONFLICT (user_id, match_id) WHERE kind = 'message' AND read_at IS NULL DO NOTHING
	`, recipientID, senderID, matchID); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, userID string) (time.Time, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var readAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE match_members SET last_read_at = NOW() WHERE match_id = $1::uuid AND user_id = $2::uuid
		RETURNING last_read_at
	`, matchID, userID).Scan(&readAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, ErrRepositoryNotMember
		}
		return time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE user_notifications SET read_at = NOW()
		WHERE user_id = $2::uuid AND match_id = $1::uuid AND kind = 'message' AND read_at IS NULL
	`, matchID, userID); err != nil {
		return time.Time{}, err
	}
	return readAt, tx.Commit(ctx)
}

// Clear hides the history for this user only; the other participant keeps theirs.
func (r *PGRepository) Clear(ctx context.Context, matchID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE match_members SET cleared_at = NOW(), last_read_at = NOW()
		WHERE match_id = $1::uuid AND user_id = $2::uuid
	`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotMember
	}
	return nil
}
