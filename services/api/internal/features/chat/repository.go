package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotParticipant = errors.New("not a participant")

type Repository interface {
	ListConversations(ctx context.Context, userID string, before *cursor, limit int) ([]conversationRow, error)
	GetConversation(ctx context.Context, userID, conversationID string) (conversationRow, error)
	Participants(ctx context.Context, userID, conversationID string) (otherUserID string, hiddenAt *time.Time, err error)
	ListMessages(ctx context.Context, conversationID string, after *time.Time, before *cursor, limit int) ([]Message, error)
	OtherLastReadAt(ctx context.Context, userID, conversationID string) (*time.Time, error)
	CreateMessage(ctx context.Context, conversationID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, conversationID string) (*time.Time, error)
	Hide(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// conversationSelect lists the caller's conversations with the other
// participant, last visible message and unread count in one query.
const conversationSelect = `
	SELECT c.id, m.id, m.created_at, other.user_id,
	       lm.id, lm.sender_id, lm.body, lm.created_at,
	       (SELECT COUNT(*) FROM messages um
	         WHERE um.conversation_id = c.id
	           AND um.sender_id <> $1
	           AND um.created_at > COALESCE(me.last_read_at, '-infinity')
	           AND um.created_at > COALESCE(me.hidden_at, '-infinity')),
	       COALESCE(c.last_message_at, c.created_at)
	FROM conversation_participants me
	JOIN conversations c ON c.id = me.conversation_id
	JOIN matches m ON m.id = c.match_id
	JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> $1
	LEFT JOIN LATERAL (
		SELECT id, sender_id, body, created_at FROM messages
		WHERE conversation_id = c.id AND created_at > COALESCE(me.hidden_at, '-infinity')
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	) lm ON TRUE
	WHERE me.user_id = $1
	  AND (me.hidden_at IS NULL OR c.last_message_at > me.hidden_at)
	  AND ` + "NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = other.user_id) OR (b.blocker_id = other.user_id AND b.blocked_id = $1))"

func scanConversation(row pgx.Row) (conversationRow, error) {
	var r conversationRow
	var lmID, lmSender, lmBody *string
	var lmAt *time.Time
	err := row.Scan(&r.ID, &r.MatchID, &r.MatchedAt, &r.OtherUserID, &lmID, &lmSender, &lmBody, &lmAt, &r.UnreadCount, &r.ActivityAt)
	if err != nil {
		return r, err
	}
	if lmID != nil {
		r.LastMessage = &Message{ID: *lmID, ConversationID: r.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
	}
	return r, nil
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, before *cursor, limit int) ([]conversationRow, error) {
	query := conversationSelect
	args := []any{userID, limit}
	if before != nil {
		query += ` AND (COALESCE(c.last_message_at, c.created_at), c.id) < ($3, $4::uuid)`
		args = append(args, before.At, before.ID)
	}
	query += ` ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC LIMIT $2`

	rows, err := r.dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]conversationRow, 0, limit)
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PGRepository) GetConversation(ctx context.Context, userID, conversationID string) (conversationRow, error) {
	// $2 is the LIMIT slot in the shared select; reuse it for the id filter.
	item, err := scanConversation(r.dbPool.QueryRow(ctx, conversationSelect+` AND c.id = $2::uuid`, userID, conversationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return conversationRow{}, ErrRepositoryNotParticipant
	}
	return item, err
}

// Participants verifies membership (and absence of blocks) and returns the
// other participant and the caller's local-deletion watermark.
func (r *PGRepository) Participants(ctx context.Context, userID, conversationID string) (string, *time.Time, error) {
	var otherID string
	var hiddenAt *time.Time
	err := r.dbPool.QueryRow(ctx, `
		SELECT other.user_id, me.hidden_at
		FROM conversation_participants me
		JOIN conversation_participants other ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $2 AND me.user_id = $1
		  AND `+profiles.NotBlockedSQL("me.user_id", "other.user_id"),
		userID, conversationID).Scan(&otherID, &hiddenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrRepositoryNotParticipant
	}
	return otherID, hiddenAt, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID string, after *time.Time, before *cursor, limit int) ([]Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM messages
		WHERE conversation_id = $1 AND created_at > COALESCE($2, '-infinity'::timestamptz)`
	args := []any{conversationID, after, limit}
	if before != nil {
		query += ` AND (created_at, id) < ($4, $5::uuid)`
		args = append(args, before.At, before.ID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT $3`

	rows, err := r.dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *PGRepository) OtherLastReadAt(ctx context.Context, userID, conversationID string) (*time.Time, error) {
	var at *time.Time
	err := r.dbPool.QueryRow(ctx, `
		SELECT last_read_at FROM conversation_participants
		WHERE conversation_id = $2 AND user_id <> $1
	`, userID, conversationID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return at, err
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at
	`, conversationID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt)
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conversations SET last_message_at = GREATEST(COALESCE(last_message_at, $2), $2) WHERE id = $1
	`, conversationID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	// Sending implies having read everything before.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = GREATEST(COALESCE(last_read_at, $3), $3)
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

// MarkRead advances the read watermark to the newest message. It returns nil
// when there is nothing new to mark.
func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (*time.Time, error) {
	var at *time.Time
	err := r.dbPool.QueryRow(ctx, `
		UPDATE conversation_participants cp
		SET last_read_at = latest.created_at
		FROM (SELECT MAX(created_at) AS created_at FROM messages WHERE conversation_id = $2) latest
		WHERE cp.conversation_id = $2 AND cp.user_id = $1
		  AND latest.created_at IS NOT NULL
		  AND (cp.last_read_at IS NULL OR cp.last_read_at < latest.created_at)
		RETURNING cp.last_read_at
	`, userID, conversationID).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return at, err
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp()
		WHERE conversation_id = $2 AND user_id = $1
	`, userID, conversationID)
	return err
}
