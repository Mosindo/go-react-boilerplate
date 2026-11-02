package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnknownInterest = errors.New("unknown interest")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

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

func (r *Repository) Upsert(ctx context.Context, userID string, v validProfile) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, show_distance, discoverable)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
			bio = EXCLUDED.bio, city = EXCLUDED.city, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
			show_distance = EXCLUDED.show_distance, discoverable = EXCLUDED.discoverable, updated_at = NOW()`,
		userID, v.FirstName, v.BirthDate, v.Gender, v.Bio, v.City, v.Latitude, v.Longitude, v.ShowDistance, v.Discoverable)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(v.InterestIDs) > 0 {
		_, err = tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, UNNEST($2::smallint[])`, userID, int32Slice(v.InterestIDs))
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return ErrUnknownInterest
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func int32Slice(in []int) []int16 {
	out := make([]int16, len(in))
	for i, v := range in {
		out[i] = int16(v)
	}
	return out
}

func (r *Repository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()`,
		userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

// Get loads the owner's profile, preferences and photo count.
func (r *Repository) Get(ctx context.Context, userID string) (*Profile, *Preferences, error) {
	var p Profile
	var birth time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude, longitude, show_distance, discoverable, created_at
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.ShowDistance, &p.Discoverable, &p.CreatedAt)
	var prefs *Preferences
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	hasProfile := err == nil
	var pr Preferences
	perr := r.pool.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&pr.InterestedIn, &pr.MinAge, &pr.MaxAge, &pr.MaxDistanceKm)
	if perr == nil {
		prefs = &pr
	} else if !errors.Is(perr, pgx.ErrNoRows) {
		return nil, nil, perr
	}
	if !hasProfile {
		return nil, prefs, nil
	}
	p.BirthDate = birth.Format("2006-01-02")
	p.Age = AgeOn(birth, time.Now())
	interests, err := r.interestsFor(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	p.Interests = interests
	photos, err := r.photosFor(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	p.Photos = photos
	return &p, prefs, nil
}

func (r *Repository) interestsFor(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label`, userID)
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

func (r *Repository) photosFor(ctx context.Context, userID string) ([]PhotoRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PhotoRef{}
	for rows.Next() {
		var p PhotoRef
		if err := rows.Scan(&p.ID, &p.Position); err != nil {
			return nil, err
		}
		p.URL = "/photos/" + p.ID
		out = append(out, p)
	}
	return out, rows.Err()
}
