package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// participant resolves the caller's membership. hiddenAt is the "cleared up to" marker of the caller.
func participant(ctx context.Context, q querier, userID, conversationID string) (other string, hiddenAt *time.Time, err error) {
	err = q.QueryRow(ctx, `
		SELECT o.user_id, me.hidden_at
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1::uuid AND me.user_id = $2::uuid
	`, conversationID, userID).Scan(&other, &hiddenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	return other, hiddenAt, err
}

func (r *Repository) listConversations(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]conversationRow, error) {
	args := []any{userID, limit + 1}
	cursorSQL := ""
	if cur != nil {
		cursorSQL = ` AND (c.updated_at, c.id) < ($3, $4::uuid)`
		args = append(args, cur.At, cur.ID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.match_id, o.user_id, c.updated_at,
		       lm.id, lm.sender_id, lm.body, lm.created_at, lm.read_at,
		       (SELECT COUNT(*) FROM messages um
		         WHERE um.conversation_id = c.id AND um.sender_id <> $1::uuid AND um.read_at IS NULL
		           AND um.created_at > COALESCE(me.hidden_at, '-infinity'))
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN conversation_participants o ON o.conversation_id = c.id AND o.user_id <> me.user_id
		JOIN LATERAL (
			SELECT m.id, m.sender_id, m.body, m.created_at, m.read_at FROM messages m
			WHERE m.conversation_id = c.id AND m.created_at > COALESCE(me.hidden_at, '-infinity')
			ORDER BY m.created_at DESC, m.id DESC LIMIT 1
		) lm ON TRUE
		WHERE me.user_id = $1::uuid`+cursorSQL+`
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conversationRow
	for rows.Next() {
		var cr conversationRow
		cr.Last.ConversationID = ""
		if err := rows.Scan(&cr.ID, &cr.MatchID, &cr.OtherUserID, &cr.UpdatedAt,
			&cr.Last.ID, &cr.Last.SenderID, &cr.Last.Body, &cr.Last.CreatedAt, &cr.Last.ReadAt, &cr.UnreadCount); err != nil {
			return nil, err
		}
		cr.Last.ConversationID = cr.ID
		out = append(out, cr)
	}
	return out, rows.Err()
}

func (r *Repository) listMessages(ctx context.Context, userID, conversationID string, limit int, cur *httpx.Cursor) ([]Message, error) {
	_, hiddenAt, err := participant(ctx, r.pool, userID, conversationID)
	if err != nil {
		return nil, err
	}
	args := []any{conversationID, hiddenAt, limit + 1}
	cursorSQL := ""
	if cur != nil {
		cursorSQL = ` AND (created_at, id) < ($4, $5::uuid)`
		args = append(args, cur.At, cur.ID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, sender_id, body, created_at, read_at
		FROM messages
		WHERE conversation_id = $1::uuid AND created_at > COALESCE($2::timestamptz, '-infinity')`+cursorSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) send(ctx context.Context, userID, conversationID, body string) (sendResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sendResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	other, _, err := participant(ctx, tx, userID, conversationID)
	if err != nil {
		return sendResult{}, err
	}
	var allowed bool
	if err := tx.QueryRow(ctx, `SELECT `+db.NotBlocked("$1::uuid", "$2::uuid"), userID, other).Scan(&allowed); err != nil {
		return sendResult{}, err
	}
	if !allowed {
		return sendResult{}, ErrBlocked
	}

	res := sendResult{OtherUserID: other}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at, read_at
	`, conversationID, userID, body).Scan(&res.Message.ID, &res.Message.ConversationID, &res.Message.SenderID,
		&res.Message.Body, &res.Message.CreatedAt, &res.Message.ReadAt); err != nil {
		return sendResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = $2 WHERE id = $1`, conversationID, res.Message.CreatedAt); err != nil {
		return sendResult{}, err
	}
	// One unread message notification per conversation: later messages are coalesced by the unique index.
	err = tx.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, conversation_id) VALUES ($1, 'message', $2, $3)
		ON CONFLICT (user_id, conversation_id) WHERE type = 'message' AND read_at IS NULL DO NOTHING
		RETURNING id
	`, other, userID, conversationID).Scan(&res.NotificationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return sendResult{}, err
	}
	return res, tx.Commit(ctx)
}

func (r *Repository) markRead(ctx context.Context, userID, conversationID string) (readResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return readResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	other, _, err := participant(ctx, tx, userID, conversationID)
	if err != nil {
		return readResult{}, err
	}
	res := readResult{OtherUserID: other, ReadAt: time.Now().UTC()}
	tag, err := tx.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE conversation_id = $1::uuid AND sender_id <> $2::uuid AND read_at IS NULL
	`, conversationID, userID, res.ReadAt)
	if err != nil {
		return readResult{}, err
	}
	res.Count = tag.RowsAffected()
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1::uuid AND conversation_id = $2::uuid AND type = 'message' AND read_at IS NULL
	`, userID, conversationID); err != nil {
		return readResult{}, err
	}
	return res, tx.Commit(ctx)
}

func (r *Repository) hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = clock_timestamp()
		WHERE conversation_id = $1::uuid AND user_id = $2::uuid
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) unreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_participants me
		JOIN messages m ON m.conversation_id = me.conversation_id
		WHERE me.user_id = $1::uuid AND m.sender_id <> me.user_id AND m.read_at IS NULL
		  AND m.created_at > COALESCE(me.hidden_at, '-infinity')
	`, userID).Scan(&n)
	return n, err
}

// partnersOf lists (conversationID, otherUserID) pairs, used before an account is deleted.
func (r *Repository) partnersOf(ctx context.Context, userID string) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT me.conversation_id, o.user_id
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.user_id = $1::uuid
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var conv, other string
		if err := rows.Scan(&conv, &other); err != nil {
			return nil, err
		}
		out[conv] = other
	}
	return out, rows.Err()
}
