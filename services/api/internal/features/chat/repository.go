package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("conversation not found")

type Repository interface {
	ListConversations(ctx context.Context, userID, beforeID string, limit int) ([]Conversation, error)
	Membership(ctx context.Context, userID, conversationID string) (Membership, error)
	ListMessages(ctx context.Context, m Membership, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, userID, conversationID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ListConversations returns the caller's conversations ordered by activity,
// with last message and unread count, in a single keyset-paginated query.
func (r *PGRepository) ListConversations(ctx context.Context, userID, beforeID string, limit int) ([]Conversation, error) {
	rows, err := r.db.Query(ctx, `
		WITH mine AS (
			SELECT c.id, c.match_id, c.created_at, COALESCE(c.last_message_at, c.created_at) AS activity,
			       me.last_read_at, me.hidden_at
			FROM conversation_participants me
			JOIN conversations c ON c.id = me.conversation_id
			WHERE me.user_id = $1
			  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
		),
		cursor AS (
			SELECT activity, id FROM mine WHERE id = $2::uuid
		)
		SELECT mine.id, mine.match_id, mine.created_at,
		       other.user_id, p.first_name, p.birthdate,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM messages m
		        WHERE m.conversation_id = mine.id
		          AND m.sender_id <> $1
		          AND m.created_at > COALESCE(mine.last_read_at, '-infinity'::timestamptz)
		          AND m.created_at > COALESCE(mine.hidden_at, '-infinity'::timestamptz)) AS unread,
		       other.last_read_at
		FROM mine
		JOIN conversation_participants other ON other.conversation_id = mine.id AND other.user_id <> $1
		JOIN profiles p ON p.user_id = other.user_id
		LEFT JOIN LATERAL (
			SELECT m.id, m.sender_id, m.body, m.created_at
			FROM messages m
			WHERE m.conversation_id = mine.id
			  AND m.created_at > COALESCE(mine.hidden_at, '-infinity'::timestamptz)
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE $2::uuid IS NULL OR (mine.activity, mine.id) < (SELECT activity, id FROM cursor)
		ORDER BY mine.activity DESC, mine.id DESC
		LIMIT $3
	`, userID, nullableUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Conversation, 0, limit)
	for rows.Next() {
		var c Conversation
		var lmID, lmSender, lmBody *string
		var lmCreated, otherRead *time.Time
		if err := rows.Scan(&c.ID, &c.MatchID, &c.MatchedAt, &c.OtherUser.UserID, &c.OtherUser.FirstName, &c.otherBirthday,
			&lmID, &lmSender, &lmBody, &lmCreated, &c.UnreadCount, &otherRead); err != nil {
			return nil, err
		}
		if lmID != nil {
			msg := Message{ID: *lmID, ConversationID: c.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmCreated}
			if msg.SenderID == userID {
				msg.IsRead = otherRead != nil && !otherRead.Before(msg.CreatedAt)
			} else {
				msg.IsRead = c.UnreadCount == 0
			}
			c.LastMessage = &msg
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) Membership(ctx context.Context, userID, conversationID string) (Membership, error) {
	m := Membership{ConversationID: conversationID}
	err := r.db.QueryRow(ctx, `
		SELECT other.user_id, me.hidden_at, me.last_read_at, other.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants other
		  ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&m.OtherUserID, &m.HiddenAt, &m.MyLastReadAt, &m.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrRepositoryNotFound
	}
	return m, err
}

func (r *PGRepository) ListMessages(ctx context.Context, m Membership, beforeID string, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.sender_id, m.body, m.created_at
		FROM messages m
		WHERE m.conversation_id = $1
		  AND m.created_at > COALESCE($2::timestamptz, '-infinity'::timestamptz)
		  AND ($3::uuid IS NULL OR (m.created_at, m.id) < (
				SELECT c.created_at, c.id FROM messages c WHERE c.id = $3::uuid AND c.conversation_id = $1))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4
	`, m.ConversationID, m.HiddenAt, nullableUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit)
	for rows.Next() {
		msg := Message{ConversationID: m.ConversationID}
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, msg)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateMessage(ctx context.Context, userID, conversationID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	msg := Message{ConversationID: conversationID, SenderID: userID, Body: body, IsRead: false}
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body)
		SELECT $1, $2, $3
		WHERE EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
		RETURNING id, created_at
	`, conversationID, userID, body).Scan(&msg.ID, &msg.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrRepositoryNotFound
	}
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, msg.CreatedAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3 WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID, msg.CreatedAt); err != nil {
		return Message{}, err
	}
	return msg, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var readAt time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrRepositoryNotFound
	}
	return readAt, err
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp(), last_read_at = clock_timestamp()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func nullableUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}
