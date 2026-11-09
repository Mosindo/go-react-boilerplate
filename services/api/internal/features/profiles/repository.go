package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/access"
	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profile not found")
	ErrUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	Upsert(ctx context.Context, userID string, in ProfileInput, birth time.Time) error
	SetLocation(ctx context.Context, userID string, lat, lon float64) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
	// PublicBatch returns public cards for many users with a fixed number of queries.
	PublicBatch(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error)
	// LoadDetails fills interests and photos for many profiles with two queries (no N+1).
	LoadDetails(ctx context.Context, ids []string) (map[string][]Interest, map[string][]Photo, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	var p OwnProfile
	var birth time.Time
	var lat *float64
	err := r.pool.QueryRow(ctx, `
		SELECT user_id::text, first_name, birth_date, gender, bio, city, latitude, is_visible, show_distance, created_at
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &lat, &p.IsVisible, &p.ShowDistance, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrNotFound
		}
		return OwnProfile{}, err
	}
	p.BirthDate = birth.Format("2006-01-02")
	p.Age = AgeOn(birth, time.Now())
	p.HasLocation = lat != nil
	interests, photos, err := r.LoadDetails(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	p.Interests, p.Photos = interests[userID], photos[userID]
	if p.Interests == nil {
		p.Interests = []Interest{}
	}
	if p.Photos == nil {
		p.Photos = []Photo{}
	}
	if p.Preferences, err = r.GetPreferences(ctx, userID); err != nil {
		return OwnProfile{}, err
	}
	p.OnboardingComplete = p.Preferences != nil && len(p.Photos) > 0
	return p, nil
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, in ProfileInput, birth time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	visible, showDistance := true, true
	if in.IsVisible != nil {
		visible = *in.IsVisible
	}
	if in.ShowDistance != nil {
		showDistance = *in.ShowDistance
	}
	// Coordinates are only overwritten when supplied; omitted coordinates keep the stored value.
	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, is_visible, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio, city = EXCLUDED.city,
		  latitude = COALESCE(EXCLUDED.latitude, profiles.latitude),
		  longitude = COALESCE(EXCLUDED.longitude, profiles.longitude),
		  is_visible = $9, show_distance = $10, updated_at = NOW()`,
		userID, in.FirstName, birth, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude, visible, showDistance)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE id = ANY($2::int[])`, userID, in.InterestIDs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.InterestIDs) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon float64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1`,
		userID, geo.RoundCoord(lat), geo.RoundCoord(lon))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	var minAge, maxAge int16
	err := r.pool.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &minAge, &maxAge, &p.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.MinAge, p.MaxAge = int(minAge), int(maxAge)
	return &p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		  max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()`,
		userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var p PublicProfile
	var birth time.Time
	var km *float64
	var showDistance bool
	err := r.pool.QueryRow(ctx, `
		SELECT t.user_id::text, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance,
		  CASE WHEN v.latitude IS NULL OR t.latitude IS NULL THEN NULL ELSE
		    6371 * acos(LEAST(1, GREATEST(-1,
		      cos(radians(v.latitude)) * cos(radians(t.latitude)) * cos(radians(t.longitude) - radians(v.longitude))
		      + sin(radians(v.latitude)) * sin(radians(t.latitude))))) END
		FROM profiles t
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = $2 AND `+access.CanView("$1", "$2"), viewerID, targetID).
		Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &showDistance, &km)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicProfile{}, ErrNotFound
		}
		return PublicProfile{}, err
	}
	p.Age = AgeOn(birth, time.Now())
	if km != nil && showDistance && viewerID != targetID {
		d := geo.ApproxKm(*km)
		p.DistanceKm = &d
	}
	interests, photos, err := r.LoadDetails(ctx, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	p.Interests, p.Photos = interests[targetID], photos[targetID]
	if p.Interests == nil {
		p.Interests = []Interest{}
	}
	if p.Photos == nil {
		p.Photos = []Photo{}
	}
	return p, nil
}

func (r *PGRepository) LoadDetails(ctx context.Context, ids []string) (map[string][]Interest, map[string][]Photo, error) {
	interests := map[string][]Interest{}
	photos := map[string][]Photo{}
	if len(ids) == 0 {
		return interests, photos, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ui.user_id::text, i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, ids)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label); err != nil {
			rows.Close()
			return nil, nil, err
		}
		interests[uid] = append(interests[uid], i)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	prows, err := r.pool.Query(ctx, `
		SELECT user_id::text, id::text, position FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var uid string
		var ph Photo
		if err := prows.Scan(&uid, &ph.ID, &ph.Position); err != nil {
			return nil, nil, err
		}
		ph.URL = "/photos/" + ph.ID
		photos[uid] = append(photos[uid], ph)
	}
	return interests, photos, prows.Err()
}

func (r *PGRepository) PublicBatch(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error) {
	out := map[string]PublicProfile{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.user_id::text, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance,
		  CASE WHEN v.latitude IS NULL OR t.latitude IS NULL THEN NULL ELSE
		    6371 * acos(LEAST(1, GREATEST(-1,
		      cos(radians(v.latitude)) * cos(radians(t.latitude)) * cos(radians(t.longitude) - radians(v.longitude))
		      + sin(radians(v.latitude)) * sin(radians(t.latitude))))) END
		FROM profiles t
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for rows.Next() {
		var p PublicProfile
		var birth time.Time
		var km *float64
		var showDistance bool
		if err := rows.Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &showDistance, &km); err != nil {
			rows.Close()
			return nil, err
		}
		p.Age = AgeOn(birth, now)
		if km != nil && showDistance {
			d := geo.ApproxKm(*km)
			p.DistanceKm = &d
		}
		out[p.UserID] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	interests, photos, err := r.LoadDetails(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, p := range out {
		p.Interests, p.Photos = interests[id], photos[id]
		if p.Interests == nil {
			p.Interests = []Interest{}
		}
		if p.Photos == nil {
			p.Photos = []Photo{}
		}
		out[id] = p
	}
	return out, nil
}
