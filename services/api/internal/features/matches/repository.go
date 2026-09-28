package matches

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("matches: not found")

// Querier is satisfied by *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// CreateInTx inserts the (ordered) match row and its conversation. Pair order is
// enforced with LEAST/GREATEST and the insert is ON CONFLICT DO NOTHING, so
// callers racing on the same pair end up with exactly one match; created reports
// whether this call inserted it.
func CreateInTx(ctx context.Context, tx pgx.Tx, userX, userY string) (matchID, conversationID string, created bool, err error) {
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id`, userX, userY).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT m.id, c.id FROM matches m JOIN match_conversations c ON c.match_id = m.id
			WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)`,
			userX, userY).Scan(&matchID, &conversationID)
		return matchID, conversationID, false, err
	}
	if err != nil {
		return "", "", false, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO match_conversations (match_id) VALUES ($1) RETURNING id`, matchID).Scan(&conversationID)
	return matchID, conversationID, err == nil, err
}

const summarySelect = `
	SELECT m.id, c.id, m.created_at, COALESCE(c.last_message_at, m.created_at),
	       o.user_id, o.first_name, o.birth_date,
	       ph.id, ph.position,
	       lm.body, lm.sender_id, lm.created_at,
	       (SELECT count(*) FROM match_messages mm
	        WHERE mm.conversation_id = c.id AND mm.sender_id <> $1 AND mm.read_at IS NULL)
	FROM matches m
	JOIN match_conversations c ON c.match_id = m.id
	JOIN profiles o ON o.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
	LEFT JOIN LATERAL (
	  SELECT id, position FROM photos WHERE user_id = o.user_id ORDER BY position LIMIT 1
	) ph ON TRUE
	LEFT JOIN LATERAL (
	  SELECT body, sender_id, created_at FROM match_messages
	  WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE (m.user_a = $1 OR m.user_b = $1)`

func scanRows(rows pgx.Rows) ([]Row, error) {
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		var photoID *string
		var photoPos *int16
		var lmBody, lmSender *string
		var lmAt *time.Time
		var unread int64
		if err := rows.Scan(&r.MatchID, &r.ConversationID, &r.CreatedAt, &r.SortAt,
			&r.User.UserID, &r.User.FirstName, &r.BirthDate,
			&photoID, &photoPos, &lmBody, &lmSender, &lmAt, &unread); err != nil {
			return nil, err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		r.SortAt = r.SortAt.UTC()
		if photoID != nil && photoPos != nil {
			r.User.Photo = &profiles.Photo{ID: *photoID, Position: int(*photoPos), URL: profiles.PhotoURL(*photoID)}
		}
		if lmBody != nil && lmSender != nil && lmAt != nil {
			r.LastMessage = &LastMessage{Body: *lmBody, SenderID: *lmSender, CreatedAt: lmAt.UTC()}
		}
		r.UnreadCount = int(unread)
		out = append(out, r)
	}
	return out, rows.Err()
}

// RowByID loads one match as seen by viewerID (used for events and swipe results).
func RowByID(ctx context.Context, q Querier, viewerID, matchID string) (Row, error) {
	rows, err := q.Query(ctx, summarySelect+` AND m.id = $2`, viewerID, matchID)
	if err != nil {
		return Row{}, err
	}
	list, err := scanRows(rows)
	if err != nil {
		return Row{}, err
	}
	if len(list) == 0 {
		return Row{}, ErrNotFound
	}
	return list[0], nil
}

type Repository interface {
	List(ctx context.Context, viewerID string, before *Cursor, limit int) ([]Row, error)
	Unmatch(ctx context.Context, viewerID, matchID string) (otherUserID string, ev RemovedEvent, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const notBlocked = `
	AND NOT EXISTS (
	  SELECT 1 FROM blocks b
	  WHERE (b.blocker_id = $1 AND b.blocked_id = o.user_id)
	     OR (b.blocker_id = o.user_id AND b.blocked_id = $1))`

func (r *PGRepository) List(ctx context.Context, viewerID string, before *Cursor, limit int) ([]Row, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if before == nil {
		rows, err = r.pool.Query(ctx, summarySelect+notBlocked+`
			ORDER BY COALESCE(c.last_message_at, m.created_at) DESC, m.id DESC LIMIT $2`, viewerID, limit)
	} else {
		rows, err = r.pool.Query(ctx, summarySelect+notBlocked+`
			AND (COALESCE(c.last_message_at, m.created_at), m.id) < ($3, $4::uuid)
			ORDER BY COALESCE(c.last_message_at, m.created_at) DESC, m.id DESC LIMIT $2`,
			viewerID, limit, before.SortAt, before.ID)
	}
	if err != nil {
		return nil, err
	}
	return scanRows(rows)
}

func (r *PGRepository) Unmatch(ctx context.Context, viewerID, matchID string) (string, RemovedEvent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", RemovedEvent{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ev RemovedEvent
	var a, b string
	err = tx.QueryRow(ctx, `
		SELECT m.id, c.id, m.user_a, m.user_b
		FROM matches m JOIN match_conversations c ON c.match_id = m.id
		WHERE m.id = $1 AND (m.user_a = $2 OR m.user_b = $2)
		FOR UPDATE OF m`, matchID, viewerID).Scan(&ev.MatchID, &ev.ConversationID, &a, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", RemovedEvent{}, ErrNotFound
	}
	if err != nil {
		return "", RemovedEvent{}, err
	}
	if err := notifications.DeleteForMatch(ctx, tx, matchID); err != nil {
		return "", RemovedEvent{}, err
	}
	// Cascades to match_conversations and match_messages.
	if _, err := tx.Exec(ctx, `DELETE FROM matches WHERE id = $1`, matchID); err != nil {
		return "", RemovedEvent{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", RemovedEvent{}, err
	}
	other := a
	if a == viewerID {
		other = b
	}
	return other, ev, nil
}
