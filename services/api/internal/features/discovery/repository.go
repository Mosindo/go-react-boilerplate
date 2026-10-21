package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoProfile = errors.New("discovery: caller has no profile")

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]candidate, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// candidatesSQL implements the recommendation policy in one place so a smarter ranking can replace
// the ORDER BY later without touching handlers. Eligibility is mutual: orientation, age range and
// distance must suit both members. Profiles already swiped, blocked (either way), paused or without
// photo never appear. Ranking: members who already liked you first, then closer, then recently active.
var candidatesSQL = `
WITH me AS (
	SELECT p.gender, p.latitude AS lat, p.longitude AS lng, u.birth_date,
	       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	FROM profiles p
	JOIN users u ON u.id = p.user_id
	JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1
), cand AS (
	SELECT p.user_id, u.last_active_at, me.max_distance_km AS my_max, pr.max_distance_km AS their_max,
	       CASE WHEN me.lat IS NOT NULL AND p.latitude IS NOT NULL
	            THEN ` + geo.SQLDistance("me.lat", "me.lng", "p.latitude", "p.longitude") + ` END AS distance_km,
	       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = p.user_id AND s.target_id = $1 AND s.action = 'like') AS liked_me
	FROM me, profiles p
	JOIN users u ON u.id = p.user_id AND u.status = 'active'
	JOIN preferences pr ON pr.user_id = p.user_id AND pr.discoverable
	WHERE p.user_id <> $1
	  AND p.gender = ANY(me.interested_in)
	  AND me.gender = ANY(pr.interested_in)
	  AND date_part('year', age(u.birth_date)) BETWEEN me.min_age AND me.max_age
	  AND date_part('year', age(me.birth_date)) BETWEEN pr.min_age AND pr.max_age
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
)
SELECT user_id, distance_km
FROM cand
WHERE distance_km IS NULL
   OR ((my_max IS NULL OR distance_km <= my_max) AND (their_max IS NULL OR distance_km <= their_max))
ORDER BY liked_me DESC,
         floor(COALESCE(distance_km, 1000) / 10) ASC,
         last_active_at DESC,
         md5(user_id::text || $1::text)
LIMIT $2`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]candidate, error) {
	var hasProfile bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id WHERE p.user_id = $1)`, userID).Scan(&hasProfile); err != nil {
		return nil, err
	}
	if !hasProfile {
		return nil, ErrNoProfile
	}
	rows, err := r.db.Query(ctx, candidatesSQL, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidates(rows)
}

func scanCandidates(rows pgx.Rows) ([]candidate, error) {
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
