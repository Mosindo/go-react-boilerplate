package discovery

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	// Candidates returns up to limit user ids that match both sides' preferences
	// and that the viewer has not swiped on, blocked or been blocked by.
	Candidates(ctx context.Context, viewerID string, limit int) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// The ranking lives entirely in the ORDER BY clause so it can be replaced by a
// smarter recommender (see Ranker docs in README) without touching the filters:
//  1. people who already liked the viewer come first (higher match odds),
//  2. recently active people,
//  3. nearest people,
//  4. a per-viewer daily pseudo-random shuffle to break ties.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.latitude, p.longitude,
			       date_part('year', age(p.birth_date))::int AS age,
			       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
			FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1::uuid
		)
		SELECT c.user_id
		FROM me
		JOIN profiles c ON c.user_id <> me.user_id AND c.is_visible
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users u ON u.id = c.user_id AND u.disabled_at IS NULL
		LEFT JOIN LATERAL (
			SELECT CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL
			            THEN haversine_km(me.latitude, me.longitude, c.latitude, c.longitude) END AS km
		) d ON TRUE
		WHERE c.gender = ANY (me.interested_in)
		  AND me.gender = ANY (cp.interested_in)
		  AND date_part('year', age(c.birth_date)) BETWEEN me.age_min AND me.age_max
		  AND me.age BETWEEN cp.age_min AND cp.age_max
		  AND (d.km IS NULL OR d.km <= LEAST(me.max_distance_km, cp.max_distance_km))
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
		  AND NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
		           OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
		ORDER BY
		  EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') DESC,
		  (u.last_active_at > NOW() - INTERVAL '7 days') DESC,
		  d.km ASC NULLS LAST,
		  md5(c.user_id::text || me.user_id::text || CURRENT_DATE::text)
		LIMIT $2
	`, viewerID, limit)
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
