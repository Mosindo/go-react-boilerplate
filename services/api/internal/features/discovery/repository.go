package discovery

import (
	"context"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportHideThreshold hides profiles reported by this many distinct members
// until a moderator reviews them.
const ReportHideThreshold = 3

type Repository interface {
	ViewerComplete(ctx context.Context, viewerID string) (bool, error)
	Candidates(ctx context.Context, viewerID string, poolSize int) ([]Candidate, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ViewerComplete(ctx context.Context, viewerID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles v WHERE v.user_id = $1 AND `+profiles.CompleteSQL("v")+`)
	`, viewerID).Scan(&ok)
	return ok, err
}

// Candidates applies every hard filter in SQL (mutual gender and age
// preferences, mutual distance, blocks, already-swiped, report threshold,
// dormancy) and returns a bounded pool that the Scorer then ranks.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, poolSize int) ([]Candidate, error) {
	rows, err := r.dbPool.Query(ctx, `
		WITH viewer AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		),
		pool AS (
			SELECT c.user_id, c.latitude, c.longitude, cp.max_distance_km AS their_max_km,
			       u.last_active_at, u.created_at,
			       CASE WHEN v.latitude IS NOT NULL AND c.latitude IS NOT NULL
			            THEN `+profiles.DistanceSQL("v", "c")+` END AS distance_km,
			       v.max_distance_km AS my_max_km,
			       v.latitude AS my_lat
			FROM viewer v
			JOIN profiles c ON c.user_id <> v.user_id
			JOIN preferences cp ON cp.user_id = c.user_id
			JOIN users u ON u.id = c.user_id
			WHERE `+profiles.EligibleSQL("c")+`
			  AND c.gender = ANY(v.interested_in)
			  AND v.gender = ANY(cp.interested_in)
			  AND c.birthdate <= (CURRENT_DATE - make_interval(years => v.min_age))
			  AND c.birthdate >  (CURRENT_DATE - make_interval(years => v.max_age + 1))
			  AND v.birthdate <= (CURRENT_DATE - make_interval(years => cp.min_age))
			  AND v.birthdate >  (CURRENT_DATE - make_interval(years => cp.max_age + 1))
			  AND (v.latitude IS NULL OR (
			        c.latitude IS NOT NULL
			        AND c.latitude BETWEEN v.latitude - v.max_distance_km / 111.0 AND v.latitude + v.max_distance_km / 111.0))
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = v.user_id AND s.target_id = c.user_id)
			  AND `+profiles.NotBlockedSQL("v.user_id", "c.user_id")+`
			  AND (SELECT COUNT(DISTINCT r.reporter_id) FROM reports r
			       WHERE r.reported_id = c.user_id AND r.status = 'open') < $3
			  AND u.last_active_at > NOW() - INTERVAL '180 days'
		)
		SELECT pool.user_id,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = pool.user_id AND s.target_id = $1 AND s.action = 'like'),
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = pool.user_id
		         WHERE a.user_id = $1),
		       pool.distance_km, pool.last_active_at, pool.created_at
		FROM pool
		WHERE pool.my_lat IS NULL
		   OR (pool.distance_km IS NOT NULL
		       AND pool.distance_km <= pool.my_max_km
		       AND pool.distance_km <= pool.their_max_km)
		ORDER BY pool.last_active_at DESC
		LIMIT $2
	`, viewerID, poolSize, ReportHideThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]Candidate, 0, poolSize)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.LikedViewer, &c.SharedInterests, &c.DistanceKm, &c.LastActiveAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}
