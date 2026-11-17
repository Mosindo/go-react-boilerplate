package discovery

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Candidates(ctx context.Context, viewerID string, box BoundingBox, poolSize int) ([]Candidate, error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// Candidates applies every hard eligibility rule in one query:
//   - both profiles complete, candidate discoverable, at least one photo;
//   - mutual gender and age preferences;
//   - distance within both members' maximum distance (bounding box + haversine);
//   - never already swiped by the viewer, never passed on the viewer;
//   - no block in either direction.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, box BoundingBox, poolSize int) ([]Candidate, error) {
	rows, err := r.dbPool.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		),
		eligible AS (
			SELECT c.user_id, u.last_active_at, cpr.max_distance_km AS their_max,
			       6371 * 2 * asin(least(1, sqrt(
			           power(sin(radians(c.latitude - me.latitude) / 2), 2) +
			           cos(radians(me.latitude)) * cos(radians(c.latitude)) *
			           power(sin(radians(c.longitude - me.longitude) / 2), 2)
			       ))) AS distance_km
			FROM me
			JOIN profiles c ON c.user_id <> me.user_id
			JOIN preferences cpr ON cpr.user_id = c.user_id
			JOIN users u ON u.id = c.user_id
			WHERE c.discoverable
			  AND c.first_name <> '' AND c.birthdate IS NOT NULL AND c.gender IS NOT NULL
			  AND c.latitude IS NOT NULL
			  AND c.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cpr.interested_in)
			  AND c.birthdate <= (current_date - make_interval(years => me.age_min::int))::date
			  AND c.birthdate >  (current_date - make_interval(years => me.age_max::int + 1))::date
			  AND me.birthdate <= (current_date - make_interval(years => cpr.age_min::int))::date
			  AND me.birthdate >  (current_date - make_interval(years => cpr.age_max::int + 1))::date
			  AND c.latitude BETWEEN $2 AND $3
			  AND (NOT $6::boolean OR c.longitude BETWEEN $4 AND $5)
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'pass')
			  AND NOT EXISTS (
			      SELECT 1 FROM blocks b
			      WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
			         OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
		)
		SELECT e.user_id, e.distance_km, e.last_active_at,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = $1
		         WHERE a.user_id = e.user_id) AS shared,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = e.user_id AND s.target_id = $1 AND s.action = 'like') AS liked_me
		FROM eligible e, me
		WHERE e.distance_km <= me.max_distance_km
		  AND e.distance_km <= e.their_max
		ORDER BY e.last_active_at DESC, e.user_id
		LIMIT $7
	`, viewerID, box.MinLat, box.MaxLat, box.MinLng, box.MaxLng, box.CheckLongitude, poolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.LastActiveAt, &c.SharedInterests, &c.LikedViewer); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
