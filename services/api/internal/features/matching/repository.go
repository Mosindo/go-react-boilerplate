package matching

import (
	"context"
	"errors"

	"example.com/api/internal/platform/access"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, *MatchEvent, error)
	ResetPasses(ctx context.Context, userID string) (int64, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// candidatesSQL is the single place where recommendation ranking lives. Hard filters
// (mutual gender/age/distance preferences, no blocks, no previous decision, not matched)
// are applied first; ordering is then: most shared interests, closest, most recently active.
// Swap the ORDER BY (or post-process ids in the service) to evolve the algorithm.
const candidatesSQL = `
WITH me AS (
  SELECT p.user_id, p.gender, p.latitude, p.longitude,
         date_part('year', age(p.birth_date))::int AS age,
         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
)
SELECT c.user_id::text
FROM me
JOIN profiles c ON c.user_id <> me.user_id AND c.is_visible
JOIN preferences cp ON cp.user_id = c.user_id
JOIN users u ON u.id = c.user_id
CROSS JOIN LATERAL (
  SELECT date_part('year', age(c.birth_date))::int AS age,
    CASE WHEN me.latitude IS NULL OR c.latitude IS NULL THEN NULL ELSE
      6371 * acos(LEAST(1, GREATEST(-1,
        cos(radians(me.latitude)) * cos(radians(c.latitude)) * cos(radians(c.longitude) - radians(me.longitude))
        + sin(radians(me.latitude)) * sin(radians(c.latitude))))) END AS dist
) x
WHERE c.gender = ANY(me.interested_in)
  AND me.gender = ANY(cp.interested_in)
  AND x.age BETWEEN me.min_age AND me.max_age
  AND me.age BETWEEN cp.min_age AND cp.max_age
  AND (x.dist IS NULL OR x.dist <= LEAST(me.max_distance_km, cp.max_distance_km))
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
                                            OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
  AND NOT EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST(me.user_id, c.user_id) AND m.user_b_id = GREATEST(me.user_id, c.user_id))
ORDER BY (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
          WHERE a.user_id = me.user_id AND b.user_id = c.user_id) DESC,
         x.dist ASC NULLS LAST, u.last_active_at DESC, c.user_id
LIMIT $2`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	// Opening the discovery feed counts as activity (used to rank recently active users first).
	if _, err := r.pool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, candidatesSQL, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, *MatchEvent, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise both directions of a pair: two simultaneous likes must produce exactly one match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))`, userID, targetID); err != nil {
		return SwipeResult{}, nil, err
	}

	var complete bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&complete); err != nil {
		return SwipeResult{}, nil, err
	}
	if !complete {
		return SwipeResult{}, nil, ErrIncompleteProfile
	}

	var allowed bool
	if err := tx.QueryRow(ctx, `SELECT `+access.CanView("$1", "$2"), userID, targetID).Scan(&allowed); err != nil {
		return SwipeResult{}, nil, err
	}
	if !allowed {
		return SwipeResult{}, nil, ErrTargetUnavailable
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&existing)
	if err == nil {
		// Idempotent: a repeated decision never creates anything new. The first decision stands.
		res := SwipeResult{Action: existing, AlreadySwiped: true}
		var matchID, convID string
		qerr := tx.QueryRow(ctx, `
			SELECT m.id::text, c.id::text FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid) AND m.unmatched_at IS NULL`,
			userID, targetID).Scan(&matchID, &convID)
		if qerr == nil {
			res.Matched, res.MatchID, res.ConversationID = true, matchID, convID
		}
		return res, nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeResult{}, nil, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return SwipeResult{}, nil, err
	}
	res := SwipeResult{Action: action}
	if action != ActionLike {
		return res, nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, targetID, userID).Scan(&reciprocal); err != nil {
		return SwipeResult{}, nil, err
	}
	if !reciprocal {
		return res, nil, tx.Commit(ctx)
	}

	ev := &MatchEvent{}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET unmatched_at = NULL WHERE matches.unmatched_at IS NOT NULL
		RETURNING id::text, user_a_id::text, user_b_id::text`, userID, targetID).Scan(&ev.MatchID, &ev.UserA, &ev.UserB)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // already an active match
			return res, nil, tx.Commit(ctx)
		}
		return SwipeResult{}, nil, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id::text`, ev.MatchID).Scan(&ev.ConversationID); err != nil {
		return SwipeResult{}, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id) VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)`,
		userID, targetID, ev.MatchID); err != nil {
		return SwipeResult{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, nil, err
	}
	res.Matched, res.MatchID, res.ConversationID = true, ev.MatchID, ev.ConversationID
	return res, ev, nil
}

func (r *PGRepository) ResetPasses(ctx context.Context, userID string) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM swipes WHERE from_user_id = $1 AND action = 'pass'`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.pool.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW()
		WHERE id = $1 AND unmatched_at IS NULL AND (user_a_id = $2 OR user_b_id = $2)
		RETURNING user_a_id::text, user_b_id::text`, matchID, userID).Scan(&a, &b)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrMatchNotFound
		}
		return "", err
	}
	if a == userID {
		return b, nil
	}
	return a, nil
}
