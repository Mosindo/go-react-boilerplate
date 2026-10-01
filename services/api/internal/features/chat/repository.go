package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrConversationNotFound hides both "does not exist" and "not yours / closed".
var ErrConversationNotFound = errors.New("conversation not found")

type Repository interface {
	// Participant authorizes userID on the conversation and returns the other
	// member. It fails unless the match is active and neither side blocked the other.
	Participant(ctx context.Context, conversationID, userID string) (otherUserID string, err error)
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]convoRow, error)
	TotalUnread(ctx context.Context, userID string) (int, error)
	ListMessages(ctx context.Context, conversationID, userID, beforeID string, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, body string) (Message, error)
	MarkRead(ctx context.Context, conversationID, readerID string) (int, time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Participant(ctx context.Context, conversationID, userID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		SELECT CASE WHEN m.user_a = $2 THEN m.user_b ELSE m.user_a END
		FROM conversations c
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $2
		WHERE c.id = $1
		  AND (m.user_a = $2 OR m.user_b = $2)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
		                     OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))`,
		conversationID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrConversationNotFound
	}
	return other, err
}

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]convoRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, m.id, other.id,
		       lm.id, lm.sender_user_id, lm.content, lm.created_at, lm.read_at,
		       (SELECT COUNT(*) FROM messages x
		        WHERE x.conversation_id = c.id AND x.sender_user_id <> $1 AND x.read_at IS NULL
		          AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at))::int
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id AND c.kind = 'match'
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		CROSS JOIN LATERAL (SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS id) other
		JOIN LATERAL (
			SELECT msg.id, msg.sender_user_id, msg.content, msg.created_at, msg.read_at
			FROM messages msg
			WHERE msg.conversation_id = c.id AND (cp.cleared_at IS NULL OR msg.created_at > cp.cleared_at)
			ORDER BY msg.created_at DESC, msg.id DESC
			LIMIT 1
		) lm ON TRUE
		WHERE cp.user_id = $1
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
		                     OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))
		ORDER BY lm.created_at DESC, c.id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]convoRow, 0, limit)
	for rows.Next() {
		var row convoRow
		row.Last.ConversationID = ""
		if err := rows.Scan(&row.ID, &row.MatchID, &row.OtherUserID,
			&row.Last.ID, &row.Last.SenderID, &row.Last.Body, &row.Last.CreatedAt, &row.Last.ReadAt, &row.Unread); err != nil {
			return nil, err
		}
		row.Last.ConversationID = row.ID
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) TotalUnread(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id AND c.kind = 'match'
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN messages x ON x.conversation_id = c.id
		WHERE cp.user_id = $1 AND x.sender_user_id <> $1 AND x.read_at IS NULL
		  AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
		                     OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))`, userID).Scan(&n)
	return n, err
}

// ListMessages returns up to limit+1 messages older than beforeID (or the
// newest ones), oldest first within the page. The extra row signals hasMore.
func (r *PGRepository) ListMessages(ctx context.Context, conversationID, userID, beforeID string, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT x.id, x.conversation_id, x.sender_user_id, x.content, x.created_at, x.read_at
		FROM messages x
		JOIN conversation_participants cp ON cp.conversation_id = x.conversation_id AND cp.user_id = $2
		WHERE x.conversation_id = $1
		  AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at)
		  AND ($3::uuid IS NULL OR (x.created_at, x.id) < (SELECT b.created_at, b.id FROM messages b WHERE b.id = $3 AND b.conversation_id = $1))
		ORDER BY x.created_at DESC, x.id DESC
		LIMIT $4`, conversationID, userID, nullable(beforeID), limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Message, 0, limit+1)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, body string) (Message, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, conversation_id, sender_user_id, content, created_at, read_at`,
		conversationID, senderID, recipientID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, readerID string) (int, time.Time, error) {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE conversation_id = $1 AND sender_user_id <> $2 AND read_at IS NULL`, conversationID, readerID, now)
	if err != nil {
		return 0, now, err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE conversation_participants SET last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, readerID, now)
	return int(tag.RowsAffected()), now, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2`, conversationID, userID)
	return err
}
