package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("profile not found")
	ErrUnknownInterest   = errors.New("unknown interest")
	errProfileNotCreated = errors.New("profile does not exist yet")
)

// HaversineKm is a SQL expression for the great-circle distance in km between two
// (lat, lng) column pairs. It is shared by every query that filters or ranks by distance.
func HaversineKm(lat1, lng1, lat2, lng2 string) string {
	return `(12742 * asin(sqrt(least(1.0,
		power(sin(radians((` + lat2 + `) - (` + lat1 + `)) / 2), 2) +
		cos(radians(` + lat1 + `)) * cos(radians(` + lat2 + `)) *
		power(sin(radians((` + lng2 + `) - (` + lng1 + `)) / 2), 2)))))`
}

// AgeSQL is a SQL expression computing a full-years age from a birth_date column.
func AgeSQL(birthDateColumn string) string {
	return `date_part('year', age(` + birthDateColumn + `))::int`
}

type StoredProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     *float64
	Longitude    *float64
	IsVisible    bool
	ShowDistance bool
}

type UpsertInput struct {
	UserID    string
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
	Interests []string
	Latitude  *float64
	Longitude *float64
}

type Repository interface {
	Upsert(ctx context.Context, in UpsertInput) error
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	SetLocation(ctx context.Context, userID string, lat, lng float64) error
	SetPreferences(ctx context.Context, userID string, p Preferences) error
	SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error
	GetPublic(ctx context.Context, viewerID string, ids []string, requireVisible bool) (map[string]PublicProfile, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	FirstName(ctx context.Context, userID string) (string, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Upsert(ctx context.Context, in UpsertInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = COALESCE(EXCLUDED.latitude, profiles.latitude),
			longitude = COALESCE(EXCLUDED.longitude, profiles.longitude),
			updated_at = NOW()
	`, in.UserID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, in.UserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, in.UserID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, in.UserID, in.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.Interests) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	var (
		p        OwnProfile
		birth    time.Time
		hasPhoto bool
		lat      *float64
	)
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, `+AgeSQL("p.birth_date")+`, p.gender, p.bio, p.city,
		       p.latitude, p.is_visible, p.show_distance,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&p.ID, &p.FirstName, &birth, &p.Age, &p.Gender, &p.Bio, &p.City,
		&lat, &p.IsVisible, &p.ShowDistance,
		&p.Preferences.InterestedIn, &p.Preferences.MinAge, &p.Preferences.MaxAge, &p.Preferences.MaxDistanceKm,
		&hasPhoto)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrNotFound
		}
		return OwnProfile{}, err
	}
	p.BirthDate = birth.Format("2006-01-02")
	p.HasLocation = lat != nil
	p.Discoverable = hasPhoto && p.IsVisible

	interests, photos, err := r.loadRelations(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	p.Interests = interests[userID]
	p.Photos = photos[userID]
	return p, nil
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) SetPreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET is_visible = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, visible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interest, 0, 32)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetPublic loads the public view of several users in three queries (profile, interests,
// photos) regardless of how many ids are requested. Blocked users (in either direction)
// are never returned. With requireVisible, hidden profiles are skipped too.
func (r *PGRepository) GetPublic(ctx context.Context, viewerID string, ids []string, requireVisible bool) (map[string]PublicProfile, error) {
	result := make(map[string]PublicProfile, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, `+AgeSQL("p.birth_date")+`, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		            THEN `+HaversineKm("v.latitude", "v.longitude", "p.latitude", "p.longitude")+`
		       END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2)
		  AND (NOT $3 OR p.is_visible)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		       OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
	`, viewerID, ids, requireVisible)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make([]string, 0, len(ids))
	for rows.Next() {
		var p PublicProfile
		var dist *float64
		if err := rows.Scan(&p.ID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &dist); err != nil {
			return nil, err
		}
		p.DistanceKm = ApproxDistance(dist)
		result[p.ID] = p
		found = append(found, p.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	interests, photos, err := r.loadRelations(ctx, found)
	if err != nil {
		return nil, err
	}
	for id, p := range result {
		p.Interests = interests[id]
		p.Photos = photos[id]
		result[id] = p
	}
	return result, nil
}

func (r *PGRepository) loadRelations(ctx context.Context, ids []string) (map[string][]Interest, map[string][]Photo, error) {
	interests := make(map[string][]Interest, len(ids))
	photos := make(map[string][]Photo, len(ids))
	if len(ids) == 0 {
		return interests, photos, nil
	}

	irows, err := r.pool.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY i.label
	`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer irows.Close()
	for irows.Next() {
		var uid string
		var i Interest
		if err := irows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return nil, nil, err
		}
		interests[uid] = append(interests[uid], i)
	}
	if err := irows.Err(); err != nil {
		return nil, nil, err
	}
	irows.Close()

	prows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1) ORDER BY user_id, position
	`, ids)
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
	if err := prows.Err(); err != nil {
		return nil, nil, err
	}
	for _, id := range ids {
		if interests[id] == nil {
			interests[id] = []Interest{}
		}
		if photos[id] == nil {
			photos[id] = []Photo{}
		}
	}
	return interests, photos, nil
}
