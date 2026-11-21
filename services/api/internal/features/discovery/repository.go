package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileIncomplete = errors.New("complete your profile (with at least one photo) to discover members")

type Repository interface {
	LoadViewer(ctx context.Context, userID string) (Viewer, error)
	ListCandidates(ctx context.Context, q CandidateQuery) ([]Candidate, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) LoadViewer(ctx context.Context, userID string) (Viewer, error) {
	var v Viewer
	err := r.dbPool.QueryRow(ctx, `
		SELECT p.user_id, u.organization_id, p.gender, date_part('year', age(p.birthdate))::int,
		       p.latitude, p.longitude, pr.interested_in, pr.min_age::int, pr.max_age::int, pr.max_distance_km::int
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	`, userID).Scan(&v.UserID, &v.OrganizationID, &v.Gender, &v.Age, &v.Latitude, &v.Longitude,
		&v.InterestedIn, &v.MinAge, &v.MaxAge, &v.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Viewer{}, ErrProfileIncomplete
	}
	return v, err
}

// ListCandidates applies every hard filter in SQL:
//   - same tenant, visible and not suspended, at least one photo
//   - mutual gender preference and mutual age range
//   - distance within both members' radius (when both are located)
//   - never swiped by the viewer (likes and passes are final), so processed
//     profiles and existing matches never resurface
//   - no block in either direction
//
// The pool is pre-sorted by cheap signals and then re-ranked in Go.
func (r *PGRepository) ListCandidates(ctx context.Context, q CandidateQuery) ([]Candidate, error) {
	v := q.Viewer
	exclude := q.ExcludeIDs
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT c.user_id,
		       d.distance_km,
		       EXISTS (
		         SELECT 1 FROM swipes s
		         WHERE s.actor_user_id = c.user_id AND s.target_user_id = $1 AND s.action = 'like'
		       ) AS liked_viewer,
		       (
		         SELECT COUNT(*)::int FROM user_interests a
		         JOIN user_interests b ON b.interest_id = a.interest_id
		         WHERE a.user_id = $1 AND b.user_id = c.user_id
		       ) AS shared_interests,
		       (SELECT COUNT(*)::int FROM photos ph WHERE ph.user_id = c.user_id) AS photo_count,
		       c.bio <> '' AS has_bio,
		       c.last_active_at,
		       c.created_at
		FROM profiles c
		JOIN users cu ON cu.id = c.user_id
		JOIN preferences cp ON cp.user_id = c.user_id
		CROSS JOIN LATERAL (
		  SELECT CASE
		    WHEN $8::double precision IS NULL OR c.latitude IS NULL THEN NULL
		    ELSE haversine_km($8, $9, c.latitude, c.longitude)
		  END AS distance_km
		) d
		WHERE c.user_id <> $1
		  AND cu.organization_id = $2
		  AND c.is_visible
		  AND NOT c.is_suspended
		  AND c.gender = ANY($3::text[])
		  AND $4 = ANY(cp.interested_in)
		  AND c.birthdate <= (CURRENT_DATE - make_interval(years => $5))
		  AND c.birthdate > (CURRENT_DATE - make_interval(years => $6 + 1))
		  AND $7 BETWEEN cp.min_age AND cp.max_age
		  AND (
		    $8::double precision IS NULL
		    OR (
		      c.latitude BETWEEN $8 - $11 AND $8 + $11
		      AND c.longitude BETWEEN $9 - $12 AND $9 + $12
		      AND d.distance_km <= LEAST($10, cp.max_distance_km)
		    )
		  )
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM swipes s WHERE s.actor_user_id = $1 AND s.target_user_id = c.user_id
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = c.user_id)
		       OR (b.blocker_user_id = c.user_id AND b.blocked_user_id = $1)
		  )
		  AND NOT (c.user_id = ANY($13::uuid[]))
		ORDER BY liked_viewer DESC, c.last_active_at DESC, c.user_id
		LIMIT $14
	`, v.UserID, v.OrganizationID, v.InterestedIn, v.Gender, v.MinAge, v.MaxAge, v.Age,
		v.Latitude, v.Longitude, v.MaxDistanceKm, q.LatDelta, q.LngDelta, exclude, q.PoolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.LikedViewer, &c.SharedInterests, &c.PhotoCount,
			&c.HasBio, &c.LastActiveAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
