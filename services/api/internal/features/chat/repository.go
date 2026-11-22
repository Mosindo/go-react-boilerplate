package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListChats(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error)
	TotalUnread(ctx context.Context, userID string) (int, error)
	Thread(ctx context.Context, userID, otherID string) (thread, error)
	ListMessages(ctx context.Context, t thread, beforeID string, limit int) ([]Message, bool, error)
	CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error)
	MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error)
	Clear(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ListChats returns one row per match (conversations exist from match time),
// ordered by latest activity. Each row costs O(1) indexed lookups.
func (r *PGRepository) ListChats(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, m.created_at,
		       other.id, COALESCE(p.first_name, ''), other.is_demo,
		       ph.id, ph.width, ph.height,
		       lm.id, lm.sender_user_id, lm.content, lm.created_at,
		       unread.n
		FROM matches m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
		JOIN users other ON other.id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		LEFT JOIN profiles p ON p.user_id = other.id
		LEFT JOIN LATERAL (
			SELECT id, width, height FROM photos WHERE user_id = other.id ORDER BY position LIMIT 1
		) ph ON TRUE
		LEFT JOIN LATERAL (
			SELECT id, sender_user_id, content, created_at
			FROM messages
			WHERE conversation_id = c.id AND created_at > COALESCE(me.cleared_at, '-infinity')
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) lm ON TRUE
		CROSS JOIN LATERAL (
			SELECT COUNT(*)::int AS n FROM (
				SELECT 1 FROM messages
				WHERE conversation_id = c.id
				  AND sender_user_id <> $1
				  AND created_at > GREATEST(COALESCE(me.last_read_at, '-infinity'), COALESCE(me.cleared_at, '-infinity'))
				LIMIT 100
			) capped
		) unread
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1)
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedSummary, error) {
		var s storedSummary
		var lmID, lmSender, lmContent *string
		var lmAt *time.Time
		err := row.Scan(&s.MatchID, &s.ConversationID, &s.MatchedAt,
			&s.User.ID, &s.User.FirstName, &s.User.IsDemo,
			&s.PhotoID, &s.PhotoW, &s.PhotoH,
			&lmID, &lmSender, &lmContent, &lmAt,
			&s.UnreadCount)
		if lmID != nil && lmSender != nil && lmContent != nil && lmAt != nil {
			s.LastMessage = &MessagePreview{ID: *lmID, SenderUserID: *lmSender, Content: *lmContent, CreatedAt: *lmAt}
		}
		return s, err
	})
}

func (r *PGRepository) TotalUnread(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM (
			SELECT 1
			FROM conversation_participants me
			JOIN matches m ON m.conversation_id = me.conversation_id
			JOIN messages msg ON msg.conversation_id = me.conversation_id
			WHERE me.user_id = $1
			  AND msg.sender_user_id <> $1
			  AND msg.created_at > GREATEST(COALESCE(me.last_read_at, '-infinity'), COALESCE(me.cleared_at, '-infinity'))
			LIMIT 1000
		) capped
	`, userID).Scan(&n)
	return n, err
}

// Thread authorises access: a conversation is reachable only through an
// existing match between the two members. Blocking or unmatching deletes the
// match, which closes access for both.
func (r *PGRepository) Thread(ctx context.Context, userID, otherID string) (thread, error) {
	var t thread
	err := r.db.QueryRow(ctx, `
		SELECT c.id, me.cleared_at, them.last_read_at
		FROM matches m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN conversation_participants me ON me.conversation_id = c.id AND me.user_id = $1
		JOIN conversation_participants them ON them.conversation_id = c.id AND them.user_id = $2
		WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
	`, userID, otherID).Scan(&t.ConversationID, &t.ClearedAt, &t.OtherLastReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return thread{}, ErrNotMatched
	}
	return t, err
}

// ListMessages returns up to limit messages older than beforeID (newest page
// when empty), in chronological order, plus whether older ones exist.
func (r *PGRepository) ListMessages(ctx context.Context, t thread, beforeID string, limit int) ([]Message, bool, error) {
	var cursorAt *time.Time
	if beforeID != "" {
		var at time.Time
		err := r.db.QueryRow(ctx, `SELECT created_at FROM messages WHERE id = $1 AND conversation_id = $2`, beforeID, t.ConversationID).Scan(&at)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, ErrInvalidCursor
		}
		if err != nil {
			return nil, false, err
		}
		cursorAt = &at
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, sender_user_id, recipient_user_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND created_at > COALESCE($2::timestamptz, '-infinity')
		  AND ($3::timestamptz IS NULL OR (created_at, id) < ($3::timestamptz, $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5
	`, t.ConversationID, t.ClearedAt, cursorAt, nullableUUID(beforeID), limit+1)
	if err != nil {
		return nil, false, err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		var recipient *string
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderUserID, &recipient, &m.Content, &m.CreatedAt)
		if recipient != nil {
			m.RecipientUserID = *recipient
		}
		return m, err
	})
	if err != nil {
		return nil, false, err
	}
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, hasMore, nil
}

func nullableUUID(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (r *PGRepository) CreateMessage(ctx context.Context, conversationID, senderID, recipientID, content string) (Message, error) {
	var m Message
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
			VALUES ($1, $2, $3, $4)
			RETURNING id, conversation_id, sender_user_id, recipient_user_id, content, created_at
		`, conversationID, senderID, recipientID, content).Scan(
			&m.ID, &m.ConversationID, &m.SenderUserID, &m.RecipientUserID, &m.Content, &m.CreatedAt,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE conversation_participants SET last_read_at = $3
			WHERE conversation_id = $1 AND user_id = $2
		`, conversationID, senderID, m.CreatedAt)
		return err
	})
	return m, err
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
		RETURNING last_read_at
	`, conversationID, userID).Scan(&at)
	return at, err
}

func (r *PGRepository) Clear(ctx context.Context, conversationID, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE conversation_participants SET cleared_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	return err
}
