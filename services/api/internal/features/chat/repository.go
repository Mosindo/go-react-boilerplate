package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrCursorNotFound       = errors.New("invalid cursor")
)

type Repository interface {
	// Membership returns the other participant of an active match the user
	// belongs to, or ErrConversationNotFound.
	Membership(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	List(ctx context.Context, matchID, before string, limit int) ([]Message, error)
	Insert(ctx context.Context, matchID, senderID, body string, clientID *string) (Message, bool, error)
	MarkRead(ctx context.Context, matchID, readerID string, at time.Time) (int64, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const columns = `id, match_id, sender_id, body, client_message_id, created_at, read_at`

func scan(row pgx.Row) (Message, error) {
	var m Message
	err := row.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.ClientMessageID, &m.CreatedAt, &m.ReadAt)
	return m, err
}

// Membership also rejects conversations where either side blocked the other.
func (r *PGRepository) Membership(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		FROM matches m
		WHERE m.id = $2 AND m.unmatched_at IS NULL AND (m.user_a_id = $1 OR m.user_b_id = $1)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = m.user_a_id AND b.blocked_id = m.user_b_id)
			   OR (b.blocker_id = m.user_b_id AND b.blocked_id = m.user_a_id)
		  )
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrConversationNotFound
	}
	return other, err
}

// List returns messages newest first using keyset pagination.
func (r *PGRepository) List(ctx context.Context, matchID, before string, limit int) ([]Message, error) {
	var rows pgx.Rows
	var err error
	if before == "" {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM messages WHERE match_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2
		`, matchID, limit)
	} else {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE id = $1 AND match_id = $2)`, before, matchID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCursorNotFound
		}
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM messages
			WHERE match_id = $1
			  AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $2)
			ORDER BY created_at DESC, id DESC LIMIT $3
		`, matchID, before, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) { return scan(row) })
}

// Insert stores a message. A retried send with the same clientMessageId
// returns the original message (created=false) instead of duplicating it.
func (r *PGRepository) Insert(ctx context.Context, matchID, senderID, body string, clientID *string) (Message, bool, error) {
	var msg Message
	created := false
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Lock the match row so an unmatch/block racing this send either
		// commits first (and the send fails) or waits for the message.
		var locked string
		err := tx.QueryRow(ctx, `
			SELECT id FROM matches
			WHERE id = $1 AND unmatched_at IS NULL AND ($2 = user_a_id OR $2 = user_b_id)
			FOR SHARE
		`, matchID, senderID).Scan(&locked)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrConversationNotFound
		}
		if err != nil {
			return err
		}
		msg, err = scan(tx.QueryRow(ctx, `
			INSERT INTO messages (match_id, sender_id, body, client_message_id)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (sender_id, client_message_id) WHERE client_message_id IS NOT NULL DO NOTHING
			RETURNING `+columns, matchID, senderID, body, clientID))
		if errors.Is(err, pgx.ErrNoRows) {
			msg, err = scan(tx.QueryRow(ctx, `
				SELECT `+columns+` FROM messages WHERE sender_id = $1 AND client_message_id = $2 AND match_id = $3
			`, senderID, clientID, matchID))
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrConversationNotFound
			}
			return err
		}
		if err != nil {
			return err
		}
		created = true
		_, err = tx.Exec(ctx, `UPDATE matches SET last_message_at = $2 WHERE id = $1`, matchID, msg.CreatedAt)
		return err
	})
	return msg, created, err
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, readerID string, at time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, matchID, readerID, at)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
