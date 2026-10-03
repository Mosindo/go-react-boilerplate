package matching

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile and add a photo first")
	ErrNotEligible       = errors.New("profile is not available")
)

// SwipeOutcome describes what a swipe changed.
type SwipeOutcome struct {
	AlreadySwiped bool
	Matched       bool
	NewMatch      bool
	MatchID       string
}

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// candidatePredicate is the single definition of "user c may be shown to / swiped by me".
// Both discovery and swipe validation use it, so the rules cannot drift apart. It expects
// the CTE `me` and the aliases c (profiles) and cp (preferences) plus the lateral `d.km`.
const candidateFrom = `
	FROM profiles c
	JOIN preferences cp ON cp.user_id = c.user_id
	CROSS JOIN me
	CROSS JOIN LATERAL (
	  SELECT CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL
	         THEN ` + "%DIST%" + ` END AS km
	) d`

const candidateWhere = `
	WHERE c.user_id <> me.user_id
	  AND c.is_visible
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND c.gender = ANY (me.interested_in)
	  AND me.gender = ANY (cp.interested_in)
	  AND ` + "%AGE_C%" + ` BETWEEN me.min_age AND me.max_age
	  AND me.age BETWEEN cp.min_age AND cp.max_age
	  AND NOT EXISTS (
	    SELECT 1 FROM blocks b
	    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND (me.max_distance_km IS NULL OR me.latitude IS NULL
	       OR (d.km IS NOT NULL AND d.km <= me.max_distance_km))
	  AND (me.max_distance_km IS NULL OR me.latitude IS NULL
	       OR c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0)`

const meCTE = `
	WITH me AS (
	  SELECT p.user_id, p.gender, p.latitude, p.longitude, ` + "%AGE_P%" + ` AS age,
	         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
	         EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id) AS has_photo
	  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	  WHERE p.user_id = $1
	)`

func expand(sql string) string {
	return strings.NewReplacer(
		"%DIST%", profiles.HaversineKm("me.latitude", "me.longitude", "c.latitude", "c.longitude"),
		"%AGE_C%", profiles.AgeSQL("c.birth_date"),
		"%AGE_P%", profiles.AgeSQL("p.birth_date"),
	).Replace(sql)
}

var (
	candidatesSQL = expand(meCTE + `
	SELECT c.user_id ` + candidateFrom + candidateWhere + `
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
	  AND me.has_photo
	ORDER BY
	  EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') DESC,
	  d.km ASC NULLS LAST,
	  c.last_active_at DESC,
	  c.user_id
	LIMIT $2`)

	eligibleSQL = expand(meCTE + `
	SELECT me.has_photo, EXISTS (SELECT 1 ` + candidateFrom + candidateWhere + ` AND c.user_id = $2)
	FROM me`)
)

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	var hasProfile bool
	var hasPhoto bool
	if err := r.pool.QueryRow(ctx, `
		SELECT true, EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
		FROM profiles WHERE user_id = $1`, userID).Scan(&hasProfile, &hasPhoto); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileIncomplete
		}
		return nil, err
	}
	if !hasPhoto {
		return nil, ErrProfileIncomplete
	}

	rows, err := r.pool.Query(ctx, candidatesSQL, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Opportunistic activity stamp used by the ranking; a failure must not break discovery.
	_ = r.touch(ctx, userID)
	return ids, nil
}

func (r *PGRepository) touch(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE profiles SET last_active_at = NOW() WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
	return err
}

// Swipe records the decision and creates the match atomically. A per-pair advisory lock makes
// two simultaneous likes serialize, so exactly one match row is produced and neither is missed.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))
	`, userID, targetID); err != nil {
		return SwipeOutcome{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		out := SwipeOutcome{AlreadySwiped: true}
		if existing == ActionLike {
			err := tx.QueryRow(ctx, `
				SELECT id FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
			`, userID, targetID).Scan(&out.MatchID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return SwipeOutcome{}, err
			}
			out.Matched = out.MatchID != ""
		}
		return out, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return SwipeOutcome{}, err
	}

	var hasPhoto, eligible bool
	if err := tx.QueryRow(ctx, eligibleSQL, userID, targetID).Scan(&hasPhoto, &eligible); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SwipeOutcome{}, ErrProfileIncomplete
		}
		return SwipeOutcome{}, err
	}
	if !hasPhoto {
		return SwipeOutcome{}, ErrProfileIncomplete
	}
	if !eligible {
		return SwipeOutcome{}, ErrNotEligible
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return SwipeOutcome{}, err
	}

	out := SwipeOutcome{}
	if action == ActionLike {
		var reverse bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
		`, userID, targetID).Scan(&reverse); err != nil {
			return SwipeOutcome{}, err
		}
		if reverse {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b)
				VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id
			`, userID, targetID).Scan(&out.MatchID)
			switch {
			case err == nil:
				out.NewMatch = true
			case errors.Is(err, pgx.ErrNoRows):
				if err := tx.QueryRow(ctx, `
					SELECT id FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
				`, userID, targetID).Scan(&out.MatchID); err != nil {
					return SwipeOutcome{}, err
				}
			default:
				return SwipeOutcome{}, err
			}
			out.Matched = true
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeOutcome{}, err
	}
	return out, nil
}
