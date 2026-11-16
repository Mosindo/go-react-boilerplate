package matching

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTargetUnavailable = errors.New("this profile is no longer available")
	ErrAlreadySwiped     = errors.New("you already answered this profile")
	ErrMatchNotFound     = errors.New("match not found")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
)

type Repository interface {
	TargetSwipeable(ctx context.Context, swiperID, targetID string) (bool, error)
	RecordSwipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error)
	GetMatch(ctx context.Context, userID, matchID string) (storedMatch, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// TargetSwipeable mirrors the visibility rules of discovery: the target has a
// discoverable profile with a photo and no block exists between the users.
func (r *PGRepository) TargetSwipeable(ctx context.Context, swiperID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2 AND p.is_discoverable
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
		)
	`, swiperID, targetID).Scan(&ok)
	return ok, err
}

// RecordSwipe stores a swipe and creates the match when the like is mutual.
//
// A transaction-scoped advisory lock on the (unordered) pair serializes the
// two users' swipes, so two simultaneous mutual likes always produce exactly
// one match. The primary key on swipes makes repeated likes impossible and
// the canonical (a < b) unique pair makes duplicate matches impossible.
func (r *PGRepository) RecordSwipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error) {
	var out swipeOutcome
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
		`, swiperID, targetID); err != nil {
			return err
		}

		var inserted bool
		err := tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
				ON CONFLICT (swiper_id, target_id) DO NOTHING
				RETURNING action
			)
			SELECT action, TRUE FROM ins
			UNION ALL
			SELECT action, FALSE FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND NOT EXISTS (SELECT 1 FROM ins)
		`, swiperID, targetID, action).Scan(&out.Action, &inserted)
		if err != nil {
			return err
		}
		if !inserted && out.Action != action {
			return ErrAlreadySwiped
		}
		if out.Action != ActionLike {
			return nil
		}

		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
		`, swiperID, targetID).Scan(&reciprocal); err != nil {
			return err
		}
		if !reciprocal {
			return nil
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO matches (user_a_id, user_b_id)
			VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			ON CONFLICT (user_a_id, user_b_id) DO NOTHING
			RETURNING id
		`, swiperID, targetID).Scan(&out.MatchID)
		if err == nil {
			out.MatchCreated = true
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Match already existed (idempotent repeat of the same like).
		var unmatchedAt *time.Time
		if err := tx.QueryRow(ctx, `
			SELECT id, unmatched_at FROM matches
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		`, swiperID, targetID).Scan(&out.MatchID, &unmatchedAt); err != nil {
			return err
		}
		if unmatchedAt != nil {
			out.MatchID = ""
		}
		return nil
	})
	return out, err
}

const matchSelect = `
	SELECT m.id,
	       CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END AS other_id,
	       m.created_at, m.last_message_at,
	       lm.body, lm.sender_id, lm.created_at,
	       (SELECT COUNT(*) FROM messages x WHERE x.match_id = m.id AND x.sender_id <> $1 AND x.read_at IS NULL)::int
	FROM matches m
	LEFT JOIN LATERAL (
		SELECT body, sender_id, created_at FROM messages
		WHERE match_id = m.id ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
`

func scanMatch(row pgx.Row) (storedMatch, error) {
	var m storedMatch
	var body, sender *string
	var sentAt *time.Time
	err := row.Scan(&m.ID, &m.OtherUserID, &m.CreatedAt, &m.LastMessageAt, &body, &sender, &sentAt, &m.UnreadCount)
	if err != nil {
		return m, err
	}
	if body != nil && sender != nil && sentAt != nil {
		m.LastMessage = &LastMessage{Body: *body, SenderID: *sender, CreatedAt: *sentAt}
	}
	return m, nil
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (storedMatch, error) {
	m, err := scanMatch(r.db.QueryRow(ctx, matchSelect+`
		WHERE m.id = $2 AND m.unmatched_at IS NULL AND (m.user_a_id = $1 OR m.user_b_id = $1)
	`, userID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return storedMatch{}, ErrMatchNotFound
	}
	return m, err
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, matchSelect+`
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1) AND m.unmatched_at IS NULL
		ORDER BY COALESCE(m.last_message_at, m.created_at) DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedMatch, error) { return scanMatch(row) })
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW()
		WHERE id = $2 AND unmatched_at IS NULL AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}
