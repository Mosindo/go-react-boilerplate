package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HaversineKM is the SQL expression for the great-circle distance in km between two profile
// aliases. It is only ever evaluated server-side: exact distances never leave the API.
func HaversineKM(a, b string) string {
	return `(2 * 6371 * asin(least(1, sqrt(
		power(sin(radians(` + a + `.latitude - ` + b + `.latitude) / 2), 2) +
		cos(radians(` + b + `.latitude)) * cos(radians(` + a + `.latitude)) *
		power(sin(radians(` + a + `.longitude - ` + b + `.longitude) / 2), 2)))))`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func photoRef(id string, pos int) PhotoRef {
	return PhotoRef{ID: id, Position: pos, URL: "/photos/" + id + "/image", ThumbURL: "/photos/" + id + "/thumb"}
}

func (r *Repository) Upsert(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, is_visible, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, TRUE), COALESCE($8, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			is_visible = COALESCE($7, profiles.is_visible),
			show_distance = COALESCE($8, profiles.show_distance),
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.IsVisible, in.ShowDistance); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, ARRAY['man','woman','non_binary'], 18, $2, $3)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, DefaultMaxAge, DefaultMaxKm); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileMissing
	}
	return nil
}

func (r *Repository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = CASE WHEN $4 <> '' THEN $4 ELSE city END, updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileMissing
	}
	return nil
}

func (r *Repository) SetInterests(ctx context.Context, userID string, ids []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(ids) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i.id FROM interests i WHERE i.id = ANY($2)
		`, userID, ids)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(ids) {
			return ErrInvalidInterests
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListInterests(ctx context.Context) ([]Interest, error) {
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

// GetSelf returns the caller's own profile; Exists is false until onboarding creates it.
func (r *Repository) GetSelf(ctx context.Context, userID string) (SelfProfile, error) {
	sp := SelfProfile{UserID: userID, Interests: []Interest{}, Photos: []PhotoRef{}, Missing: []string{}}
	var birth time.Time
	var hasLoc bool
	err := r.pool.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, latitude IS NOT NULL, is_visible, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&sp.FirstName, &birth, &sp.Gender, &sp.Bio, &sp.City, &hasLoc, &sp.IsVisible, &sp.ShowDistance)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return sp, err
	}
	if err == nil {
		sp.Exists = true
		sp.HasLocation = hasLoc
		sp.BirthDate = birth.Format("2006-01-02")
		sp.Age = AgeOn(birth, time.Now())
	}
	sp.Preferences = Preferences{InterestedIn: []string{}, MinAge: MinAge, MaxAge: DefaultMaxAge, MaxDistanceKm: DefaultMaxKm}
	err = r.pool.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&sp.Preferences.InterestedIn, &sp.Preferences.MinAge, &sp.Preferences.MaxAge, &sp.Preferences.MaxDistanceKm)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return sp, err
	}

	photos, interests, err := r.loadAttachments(ctx, []string{userID})
	if err != nil {
		return sp, err
	}
	sp.Photos = photos[userID]
	sp.Interests = interests[userID]
	if sp.Photos == nil {
		sp.Photos = []PhotoRef{}
	}
	if sp.Interests == nil {
		sp.Interests = []Interest{}
	}
	sp.Missing = MissingFields(sp.Exists, len(sp.Photos))
	sp.Complete = len(sp.Missing) == 0
	return sp, nil
}

// CardsByIDs returns public cards for ids as seen by viewerID, in a fixed number of queries
// regardless of len(ids) (no N+1). Unknown ids are simply absent from the result.
func (r *Repository) CardsByIDs(ctx context.Context, viewerID string, ids []string) (map[string]Card, error) {
	cards := make(map[string]Card, len(ids))
	if len(ids) == 0 {
		return cards, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		            THEN `+HaversineKM("p", "v")+` END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now()
	for rows.Next() {
		var c Card
		var birth time.Time
		var km *float64
		if err := rows.Scan(&c.UserID, &c.FirstName, &birth, &c.Gender, &c.Bio, &c.City, &km); err != nil {
			return nil, err
		}
		c.Age = AgeOn(birth, now)
		if km != nil {
			b := BucketDistanceKm(*km)
			c.DistanceKm = &b
		}
		c.Photos, c.Interests = []PhotoRef{}, []Interest{}
		cards[c.UserID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	photos, interests, err := r.loadAttachments(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, c := range cards {
		if p := photos[id]; p != nil {
			c.Photos = p
		}
		if i := interests[id]; i != nil {
			c.Interests = i
		}
		cards[id] = c
	}
	return cards, nil
}

func (r *Repository) loadAttachments(ctx context.Context, ids []string) (map[string][]PhotoRef, map[string][]Interest, error) {
	photos := map[string][]PhotoRef{}
	rows, err := r.pool.Query(ctx, `SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var uid, pid string
		var pos int
		if err := rows.Scan(&uid, &pid, &pos); err != nil {
			rows.Close()
			return nil, nil, err
		}
		photos[uid] = append(photos[uid], photoRef(pid, pos))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	interests := map[string][]Interest{}
	rows, err = r.pool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label
	`, ids)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, nil, err
		}
		interests[uid] = append(interests[uid], i)
	}
	return photos, interests, rows.Err()
}
