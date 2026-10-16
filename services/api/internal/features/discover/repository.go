package discover

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoProfile = errors.New("complete your profile before discovering people")

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// candidatesSQL selects who may be shown to $1, best first. Every hard rule lives in
// the WHERE clause, so a client can never be served someone it should not see:
//
//   - mutual gender interest, age range and distance (both people's preferences)
//   - hidden profiles, profiles without a photo, self, blocked (either way)
//   - anyone already swiped on (this also covers matches)
//
// Ranking is isolated in rankingOrder so the recommendation can evolve on its own.
const candidatesSQL = `
WITH me AS (
  SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude,
         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
), candidates AS (
  SELECT c.user_id, c.last_active_at,
         geo_distance_km(me.latitude, me.longitude, c.latitude, c.longitude) AS km,
         EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') AS liked_me,
         (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
          WHERE a.user_id = me.user_id AND b.user_id = c.user_id) AS shared_interests,
         cp.max_distance_km AS their_max_km
  FROM me
  JOIN profiles c ON c.user_id <> me.user_id AND c.discoverable
  JOIN preferences cp ON cp.user_id = c.user_id
  WHERE c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
    AND c.gender = ANY (me.interested_in)
    AND me.gender = ANY (cp.interested_in)
    AND date_part('year', age(c.birth_date)) BETWEEN me.min_age AND me.max_age
    AND date_part('year', age(me.birth_date)) BETWEEN cp.min_age AND cp.max_age
    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = me.user_id AND s.target_id = c.user_id)
    AND NOT EXISTS (SELECT 1 FROM blocks b
                    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
                       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
)
SELECT user_id::text FROM candidates
WHERE km <= (SELECT max_distance_km FROM me) AND km <= their_max_km
ORDER BY ` + rankingOrder + `
LIMIT $2`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	var hasProfile bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles p JOIN preferences pr USING (user_id) WHERE p.user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, userID).Scan(&hasProfile); err != nil {
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
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return ids, nil
}
