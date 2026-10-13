package discover

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Strategy selects how eligible candidates are ordered. It is the extension
// point for better recommendations: add a strategy, map it to an ORDER BY, and
// select it in the service. Every strategy shares the same eligibility filter.
type Strategy string

const (
	// StrategyBalanced shows people who already liked you first, then recently
	// active users, with a stable tiebreak.
	StrategyBalanced Strategy = "balanced"
)

var orderBy = map[Strategy]string{
	StrategyBalanced: `(liked_me.from_user_id IS NOT NULL) DESC, u.last_active_at DESC, p.user_id`,
}

type Repository interface {
	// Candidates returns up to limit user ids the viewer may be shown. When
	// targetID is non-empty the result is restricted to that user (eligibility check).
	Candidates(ctx context.Context, viewerID, targetID string, limit int, strategy Strategy) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// The eligibility rules are enforced here, server side, and nowhere else:
// mutual gender/age/distance preferences, visible and photographed profiles,
// no block in either direction, and nobody the viewer already liked or passed.
const candidatesSQL = `
SELECT p.user_id::text
FROM profiles me
JOIN preferences mp ON mp.user_id = me.user_id
JOIN profiles p ON p.user_id <> me.user_id
JOIN users u ON u.id = p.user_id AND u.status = 'active'
JOIN preferences pp ON pp.user_id = p.user_id
CROSS JOIN LATERAL (
	SELECT CASE WHEN me.latitude IS NOT NULL AND p.latitude IS NOT NULL
	            THEN haversine_km(me.latitude, me.longitude, p.latitude, p.longitude) END AS km
) d
LEFT JOIN likes liked_me
       ON liked_me.from_user_id = p.user_id AND liked_me.to_user_id = me.user_id AND liked_me.action = 'like'
WHERE me.user_id = $1::uuid
  AND p.is_visible
  AND p.gender = ANY (mp.interested_in)
  AND me.gender = ANY (pp.interested_in)
  AND p.birth_date <= (CURRENT_DATE - make_interval(years => mp.min_age))::date
  AND p.birth_date >  (CURRENT_DATE - make_interval(years => mp.max_age + 1))::date
  AND me.birth_date <= (CURRENT_DATE - make_interval(years => pp.min_age))::date
  AND me.birth_date >  (CURRENT_DATE - make_interval(years => pp.max_age + 1))::date
  AND (
	mp.max_distance_km IS NULL OR me.latitude IS NULL OR p.latitude IS NULL
	OR p.latitude BETWEEN me.latitude - mp.max_distance_km / 110.0 AND me.latitude + mp.max_distance_km / 110.0
  )
  AND (d.km IS NULL OR (
	(mp.max_distance_km IS NULL OR d.km <= mp.max_distance_km)
	AND (pp.max_distance_km IS NULL OR d.km <= pp.max_distance_km)
  ))
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = me.user_id)
  AND NOT EXISTS (SELECT 1 FROM likes l WHERE l.from_user_id = me.user_id AND l.to_user_id = p.user_id)
  AND NOT EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
	   OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id)
  )
  AND ($2::text = '' OR p.user_id = NULLIF($2, '')::uuid)
ORDER BY %s
LIMIT $3`

func (r *PGRepository) Candidates(ctx context.Context, viewerID, targetID string, limit int, strategy Strategy) ([]string, error) {
	order, ok := orderBy[strategy]
	if !ok {
		order = orderBy[StrategyBalanced]
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(candidatesSQL, order), viewerID, targetID, limit)
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
