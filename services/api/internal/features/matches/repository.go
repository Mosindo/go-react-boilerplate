package matches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotEligible = errors.New("target cannot be swiped")
	ErrRepoNotFound    = errors.New("match not found")
)

type Cursor struct {
	SortAt time.Time
	ID     string
}

type Repository interface {
	Swipe(ctx context.Context, me, target, action string) (SwipeOutcome, error)
	List(ctx context.Context, me string, after *Cursor, limit int) ([]MatchRow, error)
	Get(ctx context.Context, me, matchID string) (otherID string, err error)
	Delete(ctx context.Context, me, matchID string) (otherID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func ordered(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Swipe records one decision and, when both sides like each other, creates the match, in one transaction.
// A per-pair advisory lock guarantees that two simultaneous opposite likes always produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, me, target, action string) (SwipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lo, hi := ordered(me, target)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "pair:"+lo+":"+hi); err != nil {
		return SwipeOutcome{}, err
	}

	var eligible bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2
		    AND (p.discoverable OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = $2))
		AND EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
		AND NOT EXISTS (SELECT 1 FROM blocks b
		  WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))`, me, target).Scan(&eligible)
	if err != nil {
		return SwipeOutcome{}, err
	}
	if !eligible {
		return SwipeOutcome{}, ErrRepoNotEligible
	}

	var inserted string
	err = tx.QueryRow(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING RETURNING action`, me, target, action).Scan(&inserted)
	out := SwipeOutcome{Action: action}
	if errors.Is(err, pgx.ErrNoRows) {
		// Repeated swipe: idempotent, never creates anything new.
		out.AlreadyDid = true
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, me, target).Scan(&out.Action); err != nil {
			return SwipeOutcome{}, err
		}
		_ = tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_a = $1 AND user_b = $2`, lo, hi).Scan(&out.MatchID)
		out.Matched = out.MatchID != ""
		return out, tx.Commit(ctx)
	}
	if err != nil {
		return SwipeOutcome{}, err
	}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')`, me, target).Scan(&reciprocal); err != nil {
			return SwipeOutcome{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1, $2) ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id`, lo, hi).Scan(&out.MatchID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return SwipeOutcome{}, err
			}
			if out.MatchID != "" {
				out.Matched, out.NewMatch = true, true
				if _, err := tx.Exec(ctx, `
					INSERT INTO notifications (user_id, type, actor_id, match_id)
					VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)`, me, target, out.MatchID); err != nil {
					return SwipeOutcome{}, err
				}
			}
		}
	}
	return out, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, me string, after *Cursor, limit int) ([]MatchRow, error) {
	var afterTS any
	var afterID any
	if after != nil {
		afterTS, afterID = after.SortAt, after.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text,
		       CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END::text,
		       m.created_at,
		       COALESCE(m.last_message_at, m.created_at) AS sort_at,
		       lm.body, lm.sender_id::text, lm.created_at,
		       (SELECT COUNT(*) FROM messages u WHERE u.match_id = m.id AND u.sender_id <> $1 AND u.read_at IS NULL)::int
		FROM matches m
		LEFT JOIN LATERAL (
		  SELECT body, sender_id, created_at FROM messages x
		  WHERE x.match_id = m.id
		    AND x.created_at > COALESCE(CASE WHEN m.user_a = $1 THEN m.a_cleared_at ELSE m.b_cleared_at END, '-infinity')
		  ORDER BY x.created_at DESC, x.id DESC LIMIT 1) lm ON TRUE
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($2::timestamptz IS NULL OR (COALESCE(m.last_message_at, m.created_at), m.id) < ($2::timestamptz, $3::uuid))
		ORDER BY sort_at DESC, m.id DESC
		LIMIT $4`, me, afterTS, afterID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (MatchRow, error) {
		var m MatchRow
		err := row.Scan(&m.ID, &m.OtherID, &m.CreatedAt, &m.SortAt, &m.LastBody, &m.LastSender, &m.LastAt, &m.UnreadCount)
		return m, err
	})
}

func (r *PGRepository) Get(ctx context.Context, me, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN user_a = $1 THEN user_b ELSE user_a END::text FROM matches
		WHERE id = $2 AND (user_a = $1 OR user_b = $1)`, me, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return other, err
}

func (r *PGRepository) Delete(ctx context.Context, me, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END::text`, me, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return other, err
}
