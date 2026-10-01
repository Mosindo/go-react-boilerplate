package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrUnknownInterest = errors.New("unknown interest")
)

// DistanceKmSQL is the great-circle distance between two lat/lng pairs in km.
// Kept in one place so discovery and profile views agree.
func DistanceKmSQL(lat1, lng1, lat2, lng2 string) string {
	return fmt.Sprintf(`(6371 * 2 * asin(sqrt(LEAST(1.0,
		power(sin(radians(%[3]s - %[1]s) / 2), 2) +
		cos(radians(%[1]s)) * cos(radians(%[3]s)) * power(sin(radians(%[4]s - %[2]s) / 2), 2)))))`,
		lat1, lng1, lat2, lng2)
}

type Repository interface {
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	Upsert(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lng *float64, city string) error
	ListInterests(ctx context.Context) ([]InterestRef, error)
	ListPublic(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error)
	// Viewable reports whether viewer may open target's profile.
	Viewable(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const publicSelect = `
	SELECT p.user_id,
	       p.first_name,
	       date_part('year', age(p.birth_date))::int,
	       p.gender,
	       p.bio,
	       p.city,
	       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
	            THEN (SELECT CASE WHEN d < 10 THEN GREATEST(1, round(d))::int ELSE (round(d / 5) * 5)::int END
	                  FROM (SELECT ` + `%s` + ` AS d) x)
	       END,
	       COALESCE((SELECT json_agg(json_build_object('slug', i.slug, 'label', i.label) ORDER BY i.label)
	                 FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
	                 WHERE ui.user_id = p.user_id), '[]'::json),
	       COALESCE((SELECT json_agg(json_build_object('id', ph.id, 'position', ph.position) ORDER BY ph.position)
	                 FROM photos ph WHERE ph.user_id = p.user_id), '[]'::json)
	FROM profiles p
	LEFT JOIN profiles v ON v.user_id = $1`

func (r *PGRepository) ListPublic(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	if len(ids) == 0 {
		return []PublicProfile{}, nil
	}
	query := fmt.Sprintf(publicSelect, DistanceKmSQL("v.latitude", "v.longitude", "p.latitude", "p.longitude")) +
		` WHERE p.user_id = ANY($2::uuid[])`
	rows, err := r.pool.Query(ctx, query, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PublicProfile, 0, len(ids))
	for rows.Next() {
		var p PublicProfile
		var interestsJSON, photosJSON []byte
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &p.DistanceKm, &interestsJSON, &photosJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(interestsJSON, &p.Interests); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(photosJSON, &p.Photos); err != nil {
			return nil, err
		}
		for i := range p.Photos {
			p.Photos[i].URL = "/photos/" + p.Photos[i].ID + "/file"
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Viewable(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.status = 'active'
			WHERE p.user_id = $2
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (
				$1 = $2
				OR p.discoverable
				OR EXISTS (SELECT 1 FROM matches m
				           WHERE m.unmatched_at IS NULL
				             AND m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
			  )
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	var p OwnProfile
	var birth time.Time
	var interestsJSON, photosJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, date_part('year', age(p.birth_date))::int,
		       p.gender, p.bio, p.city, p.latitude IS NOT NULL, p.discoverable, p.show_distance,
		       p.created_at, p.updated_at,
		       COALESCE((SELECT json_agg(json_build_object('slug', i.slug, 'label', i.label) ORDER BY i.label)
		                 FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		                 WHERE ui.user_id = p.user_id), '[]'::json),
		       COALESCE((SELECT json_agg(json_build_object('id', ph.id, 'position', ph.position) ORDER BY ph.position)
		                 FROM photos ph WHERE ph.user_id = p.user_id), '[]'::json)
		FROM profiles p
		WHERE p.user_id = $1`, userID).Scan(
		&p.UserID, &p.FirstName, &birth, &p.Age, &p.Gender, &p.Bio, &p.City, &p.HasLocation,
		&p.Discoverable, &p.ShowDistance, &p.CreatedAt, &p.UpdatedAt, &interestsJSON, &photosJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}
	p.BirthDate = birth.Format("2006-01-02")
	if err := json.Unmarshal(interestsJSON, &p.Interests); err != nil {
		return OwnProfile{}, err
	}
	if err := json.Unmarshal(photosJSON, &p.Photos); err != nil {
		return OwnProfile{}, err
	}
	for i := range p.Photos {
		p.Photos[i].URL = "/photos/" + p.Photos[i].ID + "/file"
	}
	p.Complete = len(p.Photos) > 0
	return p, nil
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// birth_date is deliberately absent from the UPDATE branch: it is immutable.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, TRUE), COALESCE($8, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			discoverable = COALESCE($7, profiles.discoverable),
			show_distance = COALESCE($8, profiles.show_distance),
			updated_at = NOW()`,
		userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Discoverable, in.ShowDistance); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}

	if in.Interests != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if len(in.Interests) > 0 {
			tag, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, id FROM interests WHERE slug = ANY($2::text[])`, userID, in.Interests)
			if err != nil {
				return err
			}
			if int(tag.RowsAffected()) != len(in.Interests) {
				return ErrUnknownInterest
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p := Preferences{InterestedIn: []string{}}
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km
		FROM preferences WHERE user_id = $1`, userID).Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrProfileNotFound
		}
		return Preferences{}, err
	}
	return p, nil
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng *float64, city string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = CASE WHEN $4 <> '' THEN $4 ELSE city END, updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]InterestRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InterestRef, 0, 32)
	for rows.Next() {
		var i InterestRef
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
