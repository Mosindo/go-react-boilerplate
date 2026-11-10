package matches

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/cursor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("match not found")

type Repository interface {
	List(ctx context.Context, userID string, after *cursor.Position, limit int) ([]MatchSummary, error)
	Get(ctx context.Context, userID, matchID string) (MatchSummary, error)
	OtherParticipant(ctx context.Context, userID, matchID string) (string, error)
	ListMessages(ctx context.Context, matchID string, before *cursor.Position, limit int) ([]Message, error)
	CreateMessage(ctx context.Context, matchID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, matchID, readerID string) (int64, time.Time, error)
	Delete(ctx context.Context, userID, matchID string) (otherID string, err error)
	FirstName(ctx context.Context, userID string) (string, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Every query filters on the caller's id: a user can never reach a match (and
// therefore messages) they do not participate in.
const summarySQL = `
	WITH mine AS (
		SELECT id, user_b_id AS other_id, created_at, last_activity_at FROM matches WHERE user_a_id = $1
		UNION ALL
		SELECT id, user_a_id AS other_id, created_at, last_activity_at FROM matches WHERE user_b_id = $1
	)
	SELECT m.id, m.other_id, m.created_at, m.last_activity_at,
		lm.id, lm.sender_id, lm.body, lm.created_at, lm.read_at,
		(SELECT COUNT(*) FROM messages u WHERE u.match_id = m.id AND u.sender_id <> $1 AND u.read_at IS NULL)
	FROM mine m
	LEFT JOIN LATERAL (
		SELECT id, sender_id, body, created_at, read_at FROM messages
		WHERE match_id = m.id ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE`

func scanSummary(row pgx.Row) (MatchSummary, error) {
	var s MatchSummary
	var msgID, sender, body *string
	var msgAt *time.Time
	var readAt *time.Time
	if err := row.Scan(&s.ID, &s.OtherUserID, &s.CreatedAt, &s.LastActivityAt, &msgID, &sender, &body, &msgAt, &readAt, &s.UnreadCount); err != nil {
		return MatchSummary{}, err
	}
	if msgID != nil {
		s.LastMessage = &Message{ID: *msgID, MatchID: s.ID, SenderID: *sender, Body: *body, CreatedAt: *msgAt, ReadAt: readAt}
	}
	return s, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, after *cursor.Position, limit int) ([]MatchSummary, error) {
	var at *time.Time
	var id any
	if after != nil {
		at, id = &after.Time, after.ID
	}
	rows, err := r.pool.Query(ctx, summarySQL+`
		WHERE ($2::timestamptz IS NULL OR (m.last_activity_at, m.id) < ($2, $3::uuid))
		ORDER BY m.last_activity_at DESC, m.id DESC
		LIMIT $4
	`, userID, at, id, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MatchSummary, error) { return scanSummary(row) })
}

func (r *PGRepository) Get(ctx context.Context, userID, matchID string) (MatchSummary, error) {
	s, err := scanSummary(r.pool.QueryRow(ctx, summarySQL+` WHERE m.id = $2`, userID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return MatchSummary{}, ErrRepoNotFound
	}
	return s, err
}

func (r *PGRepository) OtherParticipant(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		SELECT CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
		FROM matches WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return other, err
}

func (r *PGRepository) ListMessages(ctx context.Context, matchID string, before *cursor.Position, limit int) ([]Message, error) {
	var at *time.Time
	var id any
	if before != nil {
		at, id = &before.Time, before.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, match_id, sender_id, body, created_at, read_at FROM messages
		WHERE match_id = $1 AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`, matchID, at, id, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
		return m, err
	})
}

func (r *PGRepository) CreateMessage(ctx context.Context, matchID, senderID, body string) (Message, error) {
	var m Message
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// The match row lock serializes concurrent sends and guarantees the
		// match still exists (an unmatch/block deletes it).
		tag, err := tx.Exec(ctx, `
			UPDATE matches SET last_activity_at = NOW()
			WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
		`, matchID, senderID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRepoNotFound
		}
		return tx.QueryRow(ctx, `
			INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
			RETURNING id, match_id, sender_id, body, created_at, read_at
		`, matchID, senderID, body).Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
	})
	return m, err
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, readerID string) (int64, time.Time, error) {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `
		UPDATE messages SET read_at = $3
		WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, matchID, readerID, now)
	if err != nil {
		return 0, time.Time{}, err
	}
	return tag.RowsAffected(), now, nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return other, err
}

func (r *PGRepository) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "Quelqu'un", nil
	}
	return name, err
}
