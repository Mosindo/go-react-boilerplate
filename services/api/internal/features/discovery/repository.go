package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrViewerIncomplete = errors.New("complete your profile to discover people")

type ViewerState struct {
	HasProfile     bool
	HasPreferences bool
	HasPhotos      bool
	IsDiscoverable bool
}

type Repository interface {
	ViewerState(ctx context.Context, viewerID string) (ViewerState, error)
	TouchActivity(ctx context.Context, viewerID string) error
	Candidates(ctx context.Context, viewerID string, poolSize int, includeDemo bool) ([]Candidate, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ViewerState(ctx context.Context, viewerID string) (ViewerState, error) {
	var s ViewerState
	err := r.db.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM profiles WHERE user_id = $1),
			EXISTS (SELECT 1 FROM preferences WHERE user_id = $1),
			EXISTS (SELECT 1 FROM photos WHERE user_id = $1),
			COALESCE((SELECT is_discoverable FROM profiles WHERE user_id = $1), FALSE)
	`, viewerID).Scan(&s.HasProfile, &s.HasPreferences, &s.HasPhotos, &s.IsDiscoverable)
	return s, err
}

func (r *PGRepository) TouchActivity(ctx context.Context, viewerID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, viewerID)
	return err
}

// Candidates applies every hard rule in SQL:
//   - mutual gender preferences and mutual age ranges;
//   - distance within both users' maximum (when the viewer shared a location);
//   - candidate is discoverable and has at least one photo;
//   - never the viewer, never someone already swiped, never a blocked pair;
//   - demo accounts only when includeDemo is set (never in production).
//
// The bounding box lets the planner use idx_profiles_location before the exact
// haversine check. The pool is capped and pre-sorted by recent activity.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, poolSize int, includeDemo bool) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km,
			       date_part('year', age(CURRENT_DATE, p.birthdate))::int AS age
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		),
		pool AS (
			SELECT c.user_id, c.last_active_at, cp.max_distance_km AS their_max,
			       CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN
			         6371 * 2 * asin(least(1, sqrt(
			           power(sin(radians(c.latitude - me.latitude) / 2), 2) +
			           cos(radians(me.latitude)) * cos(radians(c.latitude)) *
			           power(sin(radians(c.longitude - me.longitude) / 2), 2)
			         )))
			       END AS distance_km,
			       me.max_distance_km AS my_max
			FROM me
			JOIN profiles c ON c.user_id <> me.user_id
			JOIN preferences cp ON cp.user_id = c.user_id
			JOIN users u ON u.id = c.user_id
			WHERE c.is_discoverable
			  AND ($3 OR NOT u.is_demo)
			  AND c.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cp.interested_in)
			  AND c.birthdate <= (CURRENT_DATE - make_interval(years => me.age_min))
			  AND c.birthdate > (CURRENT_DATE - make_interval(years => me.age_max + 1))
			  AND me.age BETWEEN cp.age_min AND cp.age_max
			  AND (
			    me.latitude IS NULL OR (
			      c.latitude IS NOT NULL
			      AND c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
			      AND c.longitude BETWEEN me.longitude - me.max_distance_km / (111.0 * greatest(cos(radians(me.latitude)), 0.01))
			                          AND me.longitude + me.max_distance_km / (111.0 * greatest(cos(radians(me.latitude)), 0.01))
			    )
			  )
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
			       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id)
			  )
		)
		SELECT pool.user_id, pool.last_active_at,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = pool.user_id AND s.target_id = $1 AND s.action = 'like') AS liked_me,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = $1
		         WHERE a.user_id = pool.user_id)::int AS shared,
		       pool.distance_km, pool.my_max
		FROM pool
		WHERE pool.distance_km IS NULL OR pool.distance_km <= least(pool.my_max, pool.their_max)
		ORDER BY pool.last_active_at DESC, pool.user_id
		LIMIT $2
	`, viewerID, poolSize, includeDemo)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.LastActiveAt, &c.LikedViewer, &c.SharedInterests, &c.DistanceKm, &c.MaxDistanceKm)
		return c, err
	})
}
