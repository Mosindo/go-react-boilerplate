package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("chat: conversation not found")

type ConvCursor struct {
	At time.Time
	ID string
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, before *ConvCursor) ([]ConversationSummary, *ConvCursor, error)
	ListMessages(ctx context.Context, convID, userID string, limit int, beforeID int64) (MessagePage, error)
	InsertMessage(ctx context.Context, convID, senderID, body string) (msg Message, recipientID string, err error)
	MarkRead(ctx context.Context, convID, userID string) (upTo int64, otherUserID string, err error)
	Clear(ctx context.Context, convID, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, before *ConvCursor) ([]ConversationSummary, *ConvCursor, error) {
	var bt *time.Time
	var bid *string
	if before != nil {
		bt, bid = &before.At, &before.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT cv.id, cv.match_id, other.user_id, COALESCE(p.first_name, ''), COALESCE(ph.id::text, ''),
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages m
		        WHERE m.conversation_id = cv.id AND m.id > me.last_read_message_id
		          AND m.sender_id <> $1 AND (me.cleared_at IS NULL OR m.created_at > me.cleared_at))
		FROM conversation_participants me
		JOIN conversations cv ON cv.id = me.conversation_id
		JOIN conversation_participants other ON other.conversation_id = cv.id AND other.user_id <> $1
		LEFT JOIN profiles p ON p.user_id = other.user_id
		LEFT JOIN photos ph ON ph.user_id = other.user_id AND ph.position = 0
		JOIN LATERAL (
		  SELECT m.id, m.sender_id, m.body, m.created_at FROM messages m
		  WHERE m.conversation_id = cv.id AND (me.cleared_at IS NULL OR m.created_at > me.cleared_at)
		  ORDER BY m.created_at DESC, m.id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1
		  AND ($3::timestamptz IS NULL OR (lm.created_at, cv.id) < ($3, $4::uuid))
		ORDER BY lm.created_at DESC, cv.id DESC
		LIMIT $2`, userID, limit+1, bt, bid)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []ConversationSummary{}
	for rows.Next() {
		var s ConversationSummary
		var m Message
		if err := rows.Scan(&s.ID, &s.MatchID, &s.Peer.UserID, &s.Peer.FirstName, &s.Peer.PhotoID,
			&m.ID, &m.SenderID, &m.Body, &m.CreatedAt, &s.UnreadCount); err != nil {
			return nil, nil, err
		}
		m.ConversationID = s.ID
		s.LastMessage = &m
		s.UpdatedAt = m.CreatedAt
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *ConvCursor
	if len(out) > limit {
		out = out[:limit]
		last := out[len(out)-1]
		next = &ConvCursor{At: last.UpdatedAt, ID: last.ID}
	}
	return out, next, nil
}

func (r *PGRepository) ListMessages(ctx context.Context, convID, userID string, limit int, beforeID int64) (MessagePage, error) {
	var clearedAt *time.Time
	var otherRead int64
	err := r.db.QueryRow(ctx, `
		SELECT me.cleared_at, other.last_read_message_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, userID).Scan(&clearedAt, &otherRead)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessagePage{}, ErrRepoNotFound
		}
		return MessagePage{}, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, sender_id, body, created_at FROM messages
		WHERE conversation_id = $1
		  AND ($3::bigint = 0 OR id < $3)
		  AND ($4::timestamptz IS NULL OR created_at > $4)
		ORDER BY id DESC
		LIMIT $2`, convID, limit+1, beforeID, clearedAt)
	if err != nil {
		return MessagePage{}, err
	}
	defer rows.Close()
	page := MessagePage{Messages: []Message{}, OtherLastReadID: otherRead}
	for rows.Next() {
		m := Message{ConversationID: convID}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return MessagePage{}, err
		}
		page.Messages = append(page.Messages, m)
	}
	if err := rows.Err(); err != nil {
		return MessagePage{}, err
	}
	if len(page.Messages) > limit {
		page.Messages = page.Messages[:limit]
		page.NextCursor = formatID(page.Messages[len(page.Messages)-1].ID)
	}
	return page, nil
}

func (r *PGRepository) InsertMessage(ctx context.Context, convID, senderID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var recipient string
	err = tx.QueryRow(ctx, `
		SELECT other.user_id
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, senderID).Scan(&recipient)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, "", ErrRepoNotFound
		}
		return Message{}, "", err
	}
	msg := Message{ConversationID: convID, SenderID: senderID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at`, convID, senderID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_message_id = $3
		WHERE conversation_id = $1 AND user_id = $2`, convID, senderID, msg.ID); err != nil {
		return Message{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, "", err
	}
	return msg, recipient, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, convID, userID string) (int64, string, error) {
	var upTo int64
	var other string
	err := r.db.QueryRow(ctx, `
		WITH latest AS (SELECT COALESCE(MAX(id), 0) AS id FROM messages WHERE conversation_id = $1),
		upd AS (
		  UPDATE conversation_participants cp
		  SET last_read_message_id = GREATEST(cp.last_read_message_id, (SELECT id FROM latest))
		  WHERE cp.conversation_id = $1 AND cp.user_id = $2
		  RETURNING cp.last_read_message_id
		)
		SELECT upd.last_read_message_id,
		       (SELECT user_id FROM conversation_participants WHERE conversation_id = $1 AND user_id <> $2)
		FROM upd`, convID, userID).Scan(&upTo, &other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrRepoNotFound
		}
		return 0, "", err
	}
	return upTo, other, nil
}

func (r *PGRepository) Clear(ctx context.Context, convID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}
