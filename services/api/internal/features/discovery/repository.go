package discovery

import (
	"context"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// LoadViewer returns the caller's discovery context, or ErrIncompleteProfile when the profile,
// preferences, location or photos are missing.
func (r *Repository) LoadViewer(ctx context.Context, id string) (viewer, error) {
	v := viewer{ID: id}
	err := r.pool.QueryRow(ctx, `
		SELECT p.gender, p.latitude, p.longitude, pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km,
		       date_part('year', age(u.birth_date))::int
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
		  AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL AND u.birth_date IS NOT NULL
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)`, id).
		Scan(&v.Gender, &v.Lat, &v.Lon, &v.InterestedIn, &v.AgeMin, &v.AgeMax, &v.MaxDistance, &v.Age)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrIncompleteProfile
	}
	return v, err
}

// boundingBox returns a lat/lon rectangle containing every point within radiusKm of (lat, lon).
// useLon is false when the longitude range is not a simple interval (poles, antimeridian).
func boundingBox(lat, lon, radiusKm float64) (minLat, maxLat, minLon, maxLon float64, useLon bool) {
	dLat := radiusKm / 111.0 * 1.01
	minLat, maxLat = lat-dLat, lat+dLat
	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 || maxLat >= 90 || minLat <= -90 {
		return minLat, maxLat, 0, 0, false
	}
	dLon := radiusKm / (111.0 * cos) * 1.01
	if lon-dLon < -180 || lon+dLon > 180 {
		return minLat, maxLat, 0, 0, false
	}
	return minLat, maxLat, lon - dLon, lon + dLon, true
}

// Candidates runs the single set-based discovery query. order is a trusted constant from Ranker.
func (r *Repository) Candidates(ctx context.Context, v viewer, order string, fetch int) ([]Candidate, error) {
	minLat, maxLat, minLon, maxLon, useLon := boundingBox(v.Lat, v.Lon, float64(v.MaxDistance))
	query := `
	WITH cand AS (
		SELECT p.user_id, p.first_name, p.gender, p.bio, p.city, p.show_age, p.show_distance,
		       date_part('year', age(u.birth_date))::int AS age,
		       u.last_active_at,
		       COALESCE(sh.c, 0)::int AS shared_count,
		       2 * $13::float8 * asin(LEAST(1.0, sqrt(
		           power(sin(radians(p.latitude - $3::float8) / 2), 2) +
		           cos(radians($3::float8)) * cos(radians(p.latitude)) *
		           power(sin(radians(p.longitude - $4::float8) / 2), 2)))) AS distance_km,
		       LEAST($8::int, pr.max_distance_km) AS radius
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		JOIN preferences pr ON pr.user_id = p.user_id
		LEFT JOIN (
			SELECT ui.user_id, COUNT(*) AS c
			FROM user_interests ui
			JOIN user_interests mine ON mine.interest_id = ui.interest_id AND mine.user_id = $1
			GROUP BY ui.user_id
		) sh ON sh.user_id = p.user_id
		WHERE p.user_id <> $1
		  AND p.discoverable = TRUE
		  AND p.latitude BETWEEN $9::float8 AND $10::float8
		  AND ($14::boolean = FALSE OR p.longitude BETWEEN $11::float8 AND $12::float8)
		  AND p.gender = ANY($2::text[])
		  AND $5::text = ANY(pr.interested_in)
		  AND u.birth_date IS NOT NULL
		  AND date_part('year', age(u.birth_date)) BETWEEN $6::int AND $7::int
		  AND $15::int BETWEEN pr.age_min AND pr.age_max
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		                     OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		  AND NOT EXISTS (SELECT 1 FROM matches m
		                  WHERE m.user_a = LEAST($1::uuid, p.user_id) AND m.user_b = GREATEST($1::uuid, p.user_id))
	)
	SELECT user_id, first_name, gender, bio, city, show_age, show_distance, age, last_active_at, shared_count, distance_km
	FROM cand
	WHERE distance_km <= radius
	ORDER BY ` + order + `
	LIMIT $16`

	rows, err := r.pool.Query(ctx, query,
		v.ID, v.InterestedIn, v.Lat, v.Lon, v.Gender, v.AgeMin, v.AgeMax, v.MaxDistance,
		minLat, maxLat, minLon, maxLon, earthRadius, useLon, v.Age, fetch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Gender, &c.Bio, &c.City, &c.ShowAge, &c.ShowDistance,
			&c.Age, &c.LastActiveAt, &c.SharedInterests, &c.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Interests returns interest slugs per user for the page (one query).
func (r *Repository) Interests(ctx context.Context, ids []string) (map[string][]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ui.user_id, i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY ui.user_id, i.slug`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var uid, slug string
		if err := rows.Scan(&uid, &slug); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], slug)
	}
	return out, rows.Err()
}

type photoRow struct {
	ID       string
	Position int
}

// Photos returns ordered photos per user for the page (one query).
func (r *Repository) Photos(ctx context.Context, ids []string) (map[string][]photoRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]photoRow{}
	for rows.Next() {
		var uid string
		var p photoRow
		if err := rows.Scan(&uid, &p.ID, &p.Position); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}
