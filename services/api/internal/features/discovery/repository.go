package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoViewerIncomplete = errors.New("viewer profile incomplete")
	ErrRepoNotEligible      = errors.New("target not eligible")
	ErrRepoAlreadySwiped    = errors.New("already swiped")
)

type Repository interface {
	GetViewer(ctx context.Context, userID string) (Viewer, error)
	Candidates(ctx context.Context, v Viewer, minLat, maxLat, minLon, maxLon float64, exclude []string, limit int) ([]Candidate, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) GetViewer(ctx context.Context, userID string) (Viewer, error) {
	var v Viewer
	var lat, lon *float64
	var minAge, maxAge, dist int16
	var moderation string
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude, p.moderation_status,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
		       (SELECT COUNT(*) FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&v.UserID, &v.Gender, &v.Birthdate, &lat, &lon, &moderation,
		&v.InterestedIn, &minAge, &maxAge, &dist, &v.PhotoCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Viewer{}, ErrRepoViewerIncomplete
	}
	if err != nil {
		return Viewer{}, err
	}
	if lat == nil || lon == nil || v.PhotoCount == 0 {
		return Viewer{}, ErrRepoViewerIncomplete
	}
	v.Latitude, v.Longitude = *lat, *lon
	v.MinAge, v.MaxAge, v.MaxDistanceKm = int(minAge), int(maxAge), int(dist)
	v.Suspended = moderation == "suspended"
	return v, nil
}

// eligibility holds the hard rules shared by the deck and by swipes:
// mutual gender/age preferences, active & discoverable profile with a photo,
// and no block in either direction. $1 is the viewer id; `me`, `c` and `cp`
// are the viewer, candidate profile and candidate preferences.
const eligibility = `
	c.user_id <> me.user_id
	AND c.is_discoverable
	AND c.moderation_status = 'active'
	AND c.latitude IS NOT NULL
	AND c.gender = ANY(me.interested_in)
	AND me.gender = ANY(cp.interested_in)
	AND c.birthdate <= (CURRENT_DATE - make_interval(years => me.min_age::int))
	AND c.birthdate >  (CURRENT_DATE - make_interval(years => me.max_age::int + 1))
	AND me.birthdate <= (CURRENT_DATE - make_interval(years => cp.min_age::int))
	AND me.birthdate >  (CURRENT_DATE - make_interval(years => cp.max_age::int + 1))
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = $1 AND b.blocked_id = c.user_id)
		   OR (b.blocker_id = c.user_id AND b.blocked_id = $1)
	)`

const viewerCTE = `
	WITH me AS (
		SELECT p.user_id, p.gender, p.birthdate, pr.interested_in, pr.min_age, pr.max_age
		FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	)`

func (r *PGRepository) Candidates(ctx context.Context, v Viewer, minLat, maxLat, minLon, maxLon float64, exclude []string, limit int) ([]Candidate, error) {
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := r.pool.Query(ctx, viewerCTE+`
		SELECT c.user_id, c.latitude, c.longitude, c.last_active_at,
			EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = c.user_id AND l.target_id = $1 AND l.action = 'like'),
			(SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
			  WHERE a.user_id = $1 AND b.user_id = c.user_id)
		FROM profiles c
		JOIN preferences cp ON cp.user_id = c.user_id
		CROSS JOIN me
		WHERE `+eligibility+`
		  AND c.latitude BETWEEN $2 AND $3
		  AND c.longitude BETWEEN $4 AND $5
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = c.user_id)
		  AND NOT (c.user_id = ANY($6::uuid[]))
		ORDER BY c.last_active_at DESC
		LIMIT $7
	`, v.UserID, minLat, maxLat, minLon, maxLon, exclude, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.Latitude, &c.Longitude, &c.LastActiveAt, &c.LikedViewer, &c.SharedInterests)
		return c, err
	})
}

// Swipe records a like/pass and creates the match on a mutual like.
//
// A transaction-scoped advisory lock on the unordered pair serializes the two
// people's swipes: without it, two simultaneous likes could each miss the
// other's uncommitted row and no match would ever be created.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error) {
	var res SwipeResult
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))
		`, swiperID, targetID); err != nil {
			return err
		}

		var eligible bool
		if err := tx.QueryRow(ctx, viewerCTE+`
			SELECT EXISTS (
				SELECT 1 FROM profiles c
				JOIN preferences cp ON cp.user_id = c.user_id
				CROSS JOIN me
				WHERE c.user_id = $2 AND `+eligibility+`
			)
		`, swiperID, targetID).Scan(&eligible); err != nil {
			return err
		}
		if !eligible {
			return ErrRepoNotEligible
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
			ON CONFLICT (swiper_id, target_id) DO NOTHING
		`, swiperID, targetID, action)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRepoAlreadySwiped
		}
		if action != ActionLike {
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
		`, swiperID, targetID).Scan(&res.MatchID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already matched (should not happen, kept idempotent)
		}
		if err != nil {
			return err
		}
		res.Matched = true
		return nil
	})
	return res, err
}
