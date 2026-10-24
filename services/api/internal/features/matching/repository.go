package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Discover(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, userID, targetID, action string) (swipeOutcome, error)
	Unmatch(ctx context.Context, userID, matchID string) (unmatched, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// eligibilityFrom is the single source of truth for "who may see / swipe whom".
// It is used both to build the discovery feed and to authorize a swipe, so a client
// can never act on a profile the feed would not have shown. $1 is the viewer.
//
// Rules: both profiles complete and active, mutual gender interest, mutual age range,
// mutual max distance (only when both positions are known), no block in either direction.
// Distance is computed with the haversine formula (no PostGIS dependency); add a
// bounding-box prefilter or PostGIS if profile volume grows.
const eligibilityFrom = `
	FROM profiles me
	JOIN users mu ON mu.id = me.user_id AND mu.status = 'active'
	JOIN preferences mp ON mp.user_id = me.user_id
	JOIN profiles c ON c.user_id <> me.user_id AND c.discoverable
	JOIN users cu ON cu.id = c.user_id AND cu.status = 'active'
	JOIN preferences cp ON cp.user_id = c.user_id
	CROSS JOIN LATERAL (
		SELECT CASE WHEN me.latitude IS NULL OR c.latitude IS NULL THEN NULL
		  ELSE 6371 * 2 * asin(sqrt(least(1,
			power(sin(radians(c.latitude - me.latitude) / 2), 2) +
			cos(radians(me.latitude)) * cos(radians(c.latitude)) *
			power(sin(radians(c.longitude - me.longitude) / 2), 2))))
		  END AS km
	) d
	WHERE me.user_id = $1
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = me.user_id)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND c.gender = ANY (mp.interested_in)
	  AND me.gender = ANY (cp.interested_in)
	  AND date_part('year', age(CURRENT_DATE, c.birth_date)) BETWEEN mp.age_min AND mp.age_max
	  AND date_part('year', age(CURRENT_DATE, me.birth_date)) BETWEEN cp.age_min AND cp.age_max
	  AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
		   OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND (d.km IS NULL
	       OR (d.km <= COALESCE(mp.max_distance_km, 1e9) AND d.km <= COALESCE(cp.max_distance_km, 1e9)))`

// discoverQuery ranks candidates; replace the ORDER BY to change the recommendation strategy:
// people who already liked the viewer first, then shared interests, then recent activity.
const discoverQuery = `SELECT c.user_id ` + eligibilityFrom + `
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = me.user_id AND s.target_id = c.user_id)
	ORDER BY
	  EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') DESC,
	  (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
	    WHERE a.user_id = me.user_id AND b.user_id = c.user_id) DESC,
	  cu.last_active_at DESC,
	  c.user_id
	LIMIT $2`

const eligibleQuery = `SELECT TRUE ` + eligibilityFrom + ` AND c.user_id = $2`

func (r *PGRepository) Discover(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, discoverQuery, userID, limit)
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

// Swipe records a decision atomically. A per-pair advisory lock serializes two users
// liking each other at the same instant, so exactly one match (and one conversation)
// is ever created and neither side can miss the other's like.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	low, high := orderPair(userID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, low+":"+high); err != nil {
		return swipeOutcome{}, err
	}

	var eligible bool
	err = tx.QueryRow(ctx, eligibleQuery, userID, targetID).Scan(&eligible)
	if errors.Is(err, pgx.ErrNoRows) {
		return swipeOutcome{}, ErrNotEligible
	}
	if err != nil {
		return swipeOutcome{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE actor_id = $1 AND target_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		if existing != action {
			return swipeOutcome{}, ErrAlreadySwiped
		}
		return currentOutcome(ctx, tx, low, high)
	case !errors.Is(err, pgx.ErrNoRows):
		return swipeOutcome{}, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return swipeOutcome{}, err
	}

	out := swipeOutcome{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, userID).Scan(&reciprocal); err != nil {
			return swipeOutcome{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_low, user_high) VALUES ($1, $2)
				ON CONFLICT (user_low, user_high) DO NOTHING
				RETURNING id
			`, low, high).Scan(&out.MatchID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return swipeOutcome{}, err
			}
			if out.MatchID != "" {
				out.NewMatch, out.Matched = true, true
				if err := tx.QueryRow(ctx, `
					INSERT INTO conversations (kind, created_by_user_id, match_id) VALUES ('direct', $1, $2) RETURNING id
				`, userID, out.MatchID).Scan(&out.ConversationID); err != nil {
					return swipeOutcome{}, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
				`, out.ConversationID, low, high); err != nil {
					return swipeOutcome{}, err
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return swipeOutcome{}, err
	}
	return out, nil
}

// currentOutcome reports the active match of a pair (idempotent replays of a like).
func currentOutcome(ctx context.Context, tx pgx.Tx, low, high string) (swipeOutcome, error) {
	var out swipeOutcome
	err := tx.QueryRow(ctx, `
		SELECT m.id, c.id FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_low = $1 AND m.user_high = $2 AND m.unmatched_at IS NULL
	`, low, high).Scan(&out.MatchID, &out.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return swipeOutcome{}, nil
	}
	if err != nil {
		return swipeOutcome{}, err
	}
	out.Matched = true
	return out, nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (unmatched, error) {
	var u unmatched
	err := r.db.QueryRow(ctx, `
		WITH m AS (
			UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
			WHERE id = $2 AND unmatched_at IS NULL AND (user_low = $1 OR user_high = $1)
			RETURNING id, CASE WHEN user_low = $1 THEN user_high ELSE user_low END AS other_id
		)
		SELECT m.other_id, c.id FROM m JOIN conversations c ON c.match_id = m.id
	`, userID, matchID).Scan(&u.OtherUserID, &u.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return unmatched{}, ErrMatchNotFound
	}
	return u, err
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
