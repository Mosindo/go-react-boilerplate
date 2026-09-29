package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterest  = errors.New("unknown interest")
)

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	// Own loads the caller's profile (nil, ErrNotFound when onboarding has not started).
	Own(ctx context.Context, userID string) (*storedProfile, error)
	Upsert(ctx context.Context, userID string, in ProfileInput) error
	SetLocation(ctx context.Context, userID string, lat, lng float64) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	SetPreferences(ctx context.Context, userID string, p Preferences) (Preferences, error)
	// Visible loads another user's profile from the viewer's point of view.
	Visible(ctx context.Context, viewerID, targetID string) (*storedProfile, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

const profileSelect = `
	SELECT p.user_id, p.first_name, p.gender, p.bio, p.city,
	       date_part('year', age(u.birth_date))::int,
	       p.show_age, p.show_distance, p.discoverable, p.latitude, p.longitude, p.updated_at`

func (r *PGRepository) Own(ctx context.Context, userID string) (*storedProfile, error) {
	var p storedProfile
	err := r.pool.QueryRow(ctx, profileSelect+`
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1`, userID).Scan(
		&p.UserID, &p.FirstName, &p.Gender, &p.Bio, &p.City, &p.Age,
		&p.ShowAge, &p.ShowDistance, &p.Discoverable, &p.Lat, &p.Lng, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err := r.fill(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) Visible(ctx context.Context, viewerID, targetID string) (*storedProfile, error) {
	var p storedProfile
	err := r.pool.QueryRow(ctx, profileSelect+`,
		v.latitude, v.longitude,
		EXISTS (SELECT 1 FROM matches m
		        WHERE (m.user_a = $1 AND m.user_b = $2) OR (m.user_a = $2 AND m.user_b = $1)),
		EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = $2`, viewerID, targetID).Scan(
		&p.UserID, &p.FirstName, &p.Gender, &p.Bio, &p.City, &p.Age,
		&p.ShowAge, &p.ShowDistance, &p.Discoverable, &p.Lat, &p.Lng, &p.UpdatedAt,
		&p.ViewerLat, &p.ViewerLng, &p.Matched, &p.Blocked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	// Do not spend two more queries on profiles the caller may not see.
	if p.Blocked || (viewerID != targetID && !p.Matched && !p.Discoverable) {
		return nil, ErrNotFound
	}
	if err := r.fill(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// fill loads interests and photos: one query each, regardless of anything else.
func (r *PGRepository) fill(ctx context.Context, p *storedProfile) error {
	rows, err := r.pool.Query(ctx, `
		SELECT i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.slug`, p.UserID)
	if err != nil {
		return err
	}
	p.Interests = []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return err
		}
		p.Interests = append(p.Interests, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	prow, err := r.pool.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, p.UserID)
	if err != nil {
		return err
	}
	defer prow.Close()
	p.Photos = []storedPhoto{}
	for prow.Next() {
		var ph storedPhoto
		if err := prow.Scan(&ph.ID, &ph.Position); err != nil {
			return err
		}
		p.Photos = append(p.Photos, ph)
	}
	return prow.Err()
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ids := []int16{}
	if len(in.Interests) > 0 {
		rows, err := tx.Query(ctx, `SELECT id FROM interests WHERE slug = ANY($1)`, in.Interests)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int16
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ids) != len(in.Interests) {
			return ErrUnknownInterest
		}
	}

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO profiles (user_id, first_name, gender, bio, city, show_distance, show_age, discoverable)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, TRUE), COALESCE($7, TRUE), COALESCE($8, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			show_distance = COALESCE($6, profiles.show_distance),
			show_age = COALESCE($7, profiles.show_age),
			discoverable = COALESCE($8, profiles.discoverable),
			updated_at = NOW()
		RETURNING (xmax = 0)`,
		userID, in.FirstName, string(in.Gender), in.Bio, in.City, in.ShowDistance, in.ShowAge, in.Discoverable).Scan(&inserted)
	if err != nil {
		return err
	}
	if inserted {
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in) VALUES ($1, ARRAY['woman','man','non_binary'])
			ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, unnest($2::smallint[])`, userID, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPrefs(row pgx.Row) (Preferences, error) {
	var p Preferences
	var genders []string
	var ageMin, ageMax int16
	if err := row.Scan(&genders, &ageMin, &ageMax, &p.MaxDistanceKm); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrNotFound
		}
		return Preferences{}, err
	}
	for _, g := range genders {
		p.InterestedIn = append(p.InterestedIn, Gender(g))
	}
	p.AgeMin, p.AgeMax = int(ageMin), int(ageMax)
	return p, nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	return scanPrefs(r.pool.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID))
}

func (r *PGRepository) SetPreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	genders := make([]string, len(p.InterestedIn))
	for i, g := range p.InterestedIn {
		genders[i] = string(g)
	}
	// Preferences exist iff the profile exists (created together), so a miss means "no profile yet".
	return scanPrefs(r.pool.QueryRow(ctx, `
		UPDATE preferences SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
		RETURNING interested_in, age_min, age_max, max_distance_km`,
		userID, genders, int16(p.AgeMin), int16(p.AgeMax), p.MaxDistanceKm))
}
