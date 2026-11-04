package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotEligible       = errors.New("target cannot be swiped")
	ErrProfileIncomplete = errors.New("profile incomplete")
)

type Repository interface {
	// Candidates returns the ids of the next profiles for the user, best first.
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, from, to, kind string) (swipeOutcome, error)
	LikesReceived(ctx context.Context, userID string, limit int) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// candidatesSQL selects people the user has not judged yet who fit both sides'
// preferences. The ORDER BY is the single place to evolve recommendations:
// today it prefers people who already liked the user, then proximity, then recency.
const candidatesSQL = `
WITH me AS (
  SELECT p.user_id, p.gender, p.latitude, p.longitude,
         EXTRACT(YEAR FROM age(p.birth_date))::int AS age,
         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
    AND EXISTS (SELECT 1 FROM photos WHERE user_id = p.user_id)
), cand AS (
  SELECT c.user_id, u.last_active_at,
         EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = c.user_id AND s.to_user_id = me.user_id AND s.kind = 'like') AS liked_me,
         CASE WHEN me.latitude IS NULL OR c.latitude IS NULL THEN NULL ELSE
           6371 * 2 * asin(sqrt(least(1, power(sin(radians(c.latitude - me.latitude) / 2), 2)
             + cos(radians(me.latitude)) * cos(radians(c.latitude)) * power(sin(radians(c.longitude - me.longitude) / 2), 2))))
         END AS distance_km
  FROM me
  JOIN profiles c ON c.user_id <> me.user_id AND c.is_visible
  JOIN users u ON u.id = c.user_id
  JOIN preferences cp ON cp.user_id = c.user_id
  WHERE c.gender = ANY (me.interested_in)
    AND EXTRACT(YEAR FROM age(c.birth_date))::int BETWEEN me.min_age AND me.max_age
    AND me.gender = ANY (cp.interested_in)
    AND me.age BETWEEN cp.min_age AND cp.max_age
    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM blocks b
                    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
                       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
    AND (me.latitude IS NULL OR (
          c.latitude IS NOT NULL
          AND c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0))
)
SELECT user_id FROM cand
WHERE distance_km IS NULL OR distance_km <= (SELECT max_distance_km FROM me)
ORDER BY liked_me DESC, distance_km ASC NULLS LAST, last_active_at DESC, user_id
LIMIT $2`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	return r.ids(ctx, candidatesSQL, userID, limit)
}

func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit int) ([]string, error) {
	return r.ids(ctx, `
		SELECT s.from_user_id FROM swipes s
		JOIN profiles p ON p.user_id = s.from_user_id AND p.is_visible
		WHERE s.to_user_id = $1 AND s.kind = 'like'
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.from_user_id = $1 AND mine.to_user_id = s.from_user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = s.from_user_id)
		                     OR (b.blocker_id = s.from_user_id AND b.blocked_id = $1))
		ORDER BY s.created_at DESC LIMIT $2`, userID, limit)
}

func (r *PGRepository) ids(ctx context.Context, sql string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Swipe records a decision atomically and creates the match + conversation when
// the interest is mutual. A per-pair advisory lock serialises concurrent opposite
// swipes so exactly one of them observes the other and creates the match.
func (r *PGRepository) Swipe(ctx context.Context, from, to, kind string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || GREATEST($1::text, $2::text), 11))`, from, to); err != nil {
		return swipeOutcome{}, err
	}

	var complete bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, from).Scan(&complete); err != nil {
		return swipeOutcome{}, err
	}
	if !complete {
		return swipeOutcome{}, ErrProfileIncomplete
	}

	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p WHERE p.user_id = $2 AND p.user_id <> $1
		    AND (p.is_visible OR EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = $2 AND s.to_user_id = $1 AND s.kind = 'like'))
		    AND NOT EXISTS (SELECT 1 FROM blocks b
		                    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)))`,
		from, to).Scan(&eligible); err != nil {
		return swipeOutcome{}, err
	}
	if !eligible {
		return swipeOutcome{}, ErrNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, kind) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING`, from, to, kind)
	if err != nil {
		return swipeOutcome{}, err
	}
	out := swipeOutcome{Inserted: tag.RowsAffected() == 1}

	// Report an existing match for idempotent replays.
	if !out.Inserted || kind == ActionLike {
		var reciprocal bool
		if out.Inserted {
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND kind = 'like')`, to, from).Scan(&reciprocal); err != nil {
				return swipeOutcome{}, err
			}
		}
		if out.Inserted && reciprocal {
			if err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = EXCLUDED.user_a_id
				RETURNING id`, from, to).Scan(&out.MatchID); err != nil {
				return swipeOutcome{}, err
			}
			if err := tx.QueryRow(ctx, `
				INSERT INTO conversations (match_id) VALUES ($1)
				ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id RETURNING id`, out.MatchID).Scan(&out.ConversationID); err != nil {
				return swipeOutcome{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
				ON CONFLICT DO NOTHING`, out.ConversationID, from, to); err != nil {
				return swipeOutcome{}, err
			}
			out.Matched = true
		} else if !out.Inserted {
			err := tx.QueryRow(ctx, `
				SELECT m.id, c.id FROM matches m JOIN conversations c ON c.match_id = m.id
				WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)`,
				from, to).Scan(&out.MatchID, &out.ConversationID)
			if err == nil {
				out.Matched = true
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return swipeOutcome{}, err
			}
		}
	}
	return out, tx.Commit(ctx)
}
