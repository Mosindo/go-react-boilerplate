package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("conversation not found")

type conversationRow struct {
	Conversation
	OtherID string
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]conversationRow, error)
	ListMessages(ctx context.Context, userID, matchID string, limit int, beforeID int64) ([]Message, error)
	// AddMessage stores a message if userID belongs to the active match; it
	// returns the recipient id and queues the recipient's notification.
	AddMessage(ctx context.Context, userID, matchID, body string) (Message, string, error)
	// MarkRead flags the other participant's unread messages as read.
	MarkRead(ctx context.Context, userID, matchID string) (ReadReceipt, string, error)
	Clear(ctx context.Context, userID, matchID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]conversationRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT * FROM (
		  SELECT my.id AS match_id, my.created_at AS matched_at, my.other_id,
		         lm.id AS msg_id, lm.sender_id, lm.body, lm.created_at AS msg_at, lm.read_at,
		         COALESCE(un.cnt, 0)::int,
		         COALESCE(lm.created_at, my.created_at) AS activity_at
		  FROM (
		    SELECT m.id, m.created_at, cs.cleared_at,
		           CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS other_id
		    FROM matches m
		    LEFT JOIN conversation_states cs ON cs.match_id = m.id AND cs.user_id = $1
		    WHERE m.ended_at IS NULL AND ($1 = m.user_a OR $1 = m.user_b)
		  ) my
		  LEFT JOIN LATERAL (
		    SELECT id, sender_id, body, created_at, read_at FROM messages
		    WHERE match_id = my.id AND (my.cleared_at IS NULL OR created_at > my.cleared_at)
		    ORDER BY created_at DESC, id DESC LIMIT 1
		  ) lm ON TRUE
		  LEFT JOIN LATERAL (
		    SELECT count(*) AS cnt FROM messages
		    WHERE match_id = my.id AND sender_id <> $1 AND read_at IS NULL
		      AND (my.cleared_at IS NULL OR created_at > my.cleared_at)
		  ) un ON TRUE
		  WHERE my.cleared_at IS NULL OR lm.id IS NOT NULL
		) t
		WHERE ($2::timestamptz IS NULL OR (t.activity_at, t.match_id) < ($2, $3::uuid))
		ORDER BY t.activity_at DESC, t.match_id DESC
		LIMIT $4
	`, userID, beforeAt, nullIfEmpty(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []conversationRow{}
	for rows.Next() {
		var cr conversationRow
		var mid *int64
		var sender, body *string
		var mat *time.Time
		var readAt *time.Time
		if err := rows.Scan(&cr.MatchID, &cr.MatchedAt, &cr.OtherID, &mid, &sender, &body, &mat, &readAt, &cr.UnreadCount, &cr.ActivityAt); err != nil {
			return nil, err
		}
		if mid != nil {
			cr.LastMessage = &Message{ID: *mid, MatchID: cr.MatchID, SenderID: *sender, Body: *body, CreatedAt: *mat, ReadAt: readAt}
		}
		out = append(out, cr)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, matchID string, limit int, beforeID int64) ([]Message, error) {
	var member bool
	var clearedAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT TRUE, cs.cleared_at
		FROM matches m
		LEFT JOIN conversation_states cs ON cs.match_id = m.id AND cs.user_id = $1
		WHERE m.id = $2 AND m.ended_at IS NULL AND ($1 = m.user_a OR $1 = m.user_b)
	`, userID, matchID).Scan(&member, &clearedAt)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_id, body, created_at, read_at FROM messages
		WHERE match_id = $1
		  AND ($2::timestamptz IS NULL OR created_at > $2)
		  AND ($3::bigint = 0 OR id < $3)
		ORDER BY id DESC
		LIMIT $4
	`, matchID, clearedAt, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	msgs := []Message{}
	for rows.Next() {
		m := Message{MatchID: matchID}
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *PGRepository) AddMessage(ctx context.Context, userID, matchID, body string) (Message, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var recipient string
	err = tx.QueryRow(ctx, `
		SELECT CASE WHEN user_a = $1 THEN user_b ELSE user_a END
		FROM matches WHERE id = $2 AND ended_at IS NULL AND ($1 = user_a OR $1 = user_b)
		FOR SHARE
	`, userID, matchID).Scan(&recipient)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return Message{}, "", ErrRepositoryNotFound
	}
	if err != nil {
		return Message{}, "", err
	}

	msg := Message{MatchID: matchID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3) RETURNING id, created_at
	`, matchID, userID, body).Scan(&msg.ID, &msg.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if err := notifications.Create(ctx, tx, notifications.New{
		UserID: recipient, Type: notifications.TypeMessage, ActorID: userID, MatchID: matchID,
	}); err != nil {
		return Message{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Message{}, "", err
	}
	return msg, recipient, nil
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, matchID string) (ReadReceipt, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReadReceipt{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var other string
	err = tx.QueryRow(ctx, `
		SELECT CASE WHEN user_a = $1 THEN user_b ELSE user_a END
		FROM matches WHERE id = $2 AND ended_at IS NULL AND ($1 = user_a OR $1 = user_b)
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return ReadReceipt{}, "", ErrRepositoryNotFound
	}
	if err != nil {
		return ReadReceipt{}, "", err
	}

	receipt := ReadReceipt{MatchID: matchID, ReaderID: userID}
	err = tx.QueryRow(ctx, `
		WITH upd AS (
		  UPDATE messages SET read_at = NOW()
		  WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
		  RETURNING 1
		)
		SELECT count(*)::int, NOW() FROM upd
	`, matchID, userID).Scan(&receipt.Count, &receipt.ReadAt)
	if err != nil {
		return ReadReceipt{}, "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND match_id = $2 AND type = 'message' AND read_at IS NULL
	`, userID, matchID); err != nil {
		return ReadReceipt{}, "", err
	}
	return receipt, other, tx.Commit(ctx)
}

func (r *PGRepository) Clear(ctx context.Context, userID, matchID string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO conversation_states (match_id, user_id, cleared_at)
		SELECT m.id, $1, NOW() FROM matches m
		WHERE m.id = $2 AND m.ended_at IS NULL AND ($1 = m.user_a OR $1 = m.user_b)
		ON CONFLICT (match_id, user_id) DO UPDATE SET cleared_at = EXCLUDED.cleared_at
	`, userID, matchID)
	if isInvalidUUID(err) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND match_id = $2 AND read_at IS NULL`, userID, matchID)
	return err
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
