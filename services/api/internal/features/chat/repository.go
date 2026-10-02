package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotParticipant = errors.New("not a participant")

type Repository interface {
	// Insert stores a message and the recipient's notification; it fails with ErrRepoNotParticipant
	// unless sender belongs to the match. It returns the recipient id.
	Insert(ctx context.Context, matchID, senderID, body string) (Message, string, error)
	List(ctx context.Context, matchID, userID string, beforeID int64, limit int) ([]Message, error)
	MarkRead(ctx context.Context, matchID, userID string) (n int, otherID string, err error)
	Clear(ctx context.Context, matchID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const otherSQL = `CASE WHEN user_a = $2 THEN user_b ELSE user_a END::text`

func (r *PGRepository) Insert(ctx context.Context, matchID, senderID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var other string
	err = tx.QueryRow(ctx, `SELECT `+otherSQL+` FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2) FOR UPDATE`, matchID, senderID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, "", ErrRepoNotParticipant
	}
	if err != nil {
		return Message{}, "", err
	}
	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, match_id::text, sender_id::text, body, created_at, read_at`, matchID, senderID, body).
		Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
	if err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// One unread "message" notification per conversation: no spam when someone sends ten messages.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id) VALUES ($1, 'message', $2, $3)
		ON CONFLICT (user_id, match_id) WHERE type = 'message' AND read_at IS NULL DO NOTHING`, other, senderID, matchID); err != nil {
		return Message{}, "", err
	}
	return m, other, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, matchID, userID string, beforeID int64, limit int) ([]Message, error) {
	var ok bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2))`, matchID, userID).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrRepoNotParticipant
	}
	rows, err := r.db.Query(ctx, `
		SELECT x.id, x.match_id::text, x.sender_id::text, x.body, x.created_at, x.read_at
		FROM messages x JOIN matches m ON m.id = x.match_id
		WHERE x.match_id = $1 AND ($3::bigint = 0 OR x.id < $3)
		  AND x.created_at > COALESCE(CASE WHEN m.user_a = $2 THEN m.a_cleared_at ELSE m.b_cleared_at END, '-infinity')
		ORDER BY x.created_at DESC, x.id DESC LIMIT $4`, matchID, userID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
		return m, err
	})
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, userID string) (int, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var other string
	err = tx.QueryRow(ctx, `SELECT `+otherSQL+` FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrRepoNotParticipant
	}
	if err != nil {
		return 0, "", err
	}
	tag, err := tx.Exec(ctx, `UPDATE messages SET read_at = NOW() WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL`, matchID, userID)
	if err != nil {
		return 0, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $2 AND match_id = $1 AND type = 'message' AND read_at IS NULL`, matchID, userID); err != nil {
		return 0, "", err
	}
	return int(tag.RowsAffected()), other, tx.Commit(ctx)
}

func (r *PGRepository) Clear(ctx context.Context, matchID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE matches SET
		  a_cleared_at = CASE WHEN user_a = $2 THEN NOW() ELSE a_cleared_at END,
		  b_cleared_at = CASE WHEN user_b = $2 THEN NOW() ELSE b_cleared_at END
		WHERE id = $1 AND (user_a = $2 OR user_b = $2)`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotParticipant
	}
	return nil
}
