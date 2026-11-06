package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetPrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetMine(ctx context.Context, userID string) (MyProfileResponse, error)
	GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
	LoadCards(ctx context.Context, userIDs []string) (map[string][]Photo, map[string][]Interest, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func PhotoURL(id string) string { return "/photos/" + id }

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, lat, lng)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city,
			lat = EXCLUDED.lat, lng = EXCLUDED.lng, updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, in.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.Interests) {
			return ValidationError{Message: "unknown interest"}
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoProfile
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
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

func (r *PGRepository) GetMine(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp := MyProfileResponse{
		Preferences: Preferences{InterestedIn: []string{"man", "nonbinary", "woman"}, MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: DefaultMaxDist},
		Interests:   []Interest{},
		Photos:      []Photo{},
	}
	var p OwnProfile
	var birth time.Time
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, is_discoverable, show_distance, created_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &p.IsDiscoverable, &p.ShowDistance, &p.CreatedAt)
	switch {
	case err == nil:
		p.BirthDate = birth.Format("2006-01-02")
		resp.Profile = &p
	case !errors.Is(err, pgx.ErrNoRows):
		return resp, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&resp.Preferences.InterestedIn, &resp.Preferences.MinAge, &resp.Preferences.MaxAge, &resp.Preferences.MaxDistanceKm)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resp, err
	}

	photos, interests, err := r.LoadCards(ctx, []string{userID})
	if err != nil {
		return resp, err
	}
	resp.Photos, resp.Interests = photos[userID], interests[userID]
	if resp.Photos == nil {
		resp.Photos = []Photo{}
	}
	if resp.Interests == nil {
		resp.Interests = []Interest{}
	}
	return resp, nil
}

// LoadCards fetches photos and interests for many users with two queries
// (never one per user).
func (r *PGRepository) LoadCards(ctx context.Context, userIDs []string) (map[string][]Photo, map[string][]Interest, error) {
	photos := make(map[string][]Photo, len(userIDs))
	interests := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return photos, interests, nil
	}
	rows, err := r.db.Query(ctx, `SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var uid string
		var ph Photo
		if err := rows.Scan(&uid, &ph.ID, &ph.Position); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ph.URL = PhotoURL(ph.ID)
		photos[uid] = append(photos[uid], ph)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}

	rows, err = r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var in Interest
		if err := rows.Scan(&uid, &in.Slug, &in.Label); err != nil {
			return nil, nil, err
		}
		interests[uid] = append(interests[uid], in)
	}
	return photos, interests, rows.Err()
}

func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var p PublicProfile
	var birth time.Time
	var dist *float64
	var showDistance bool
	err := r.db.QueryRow(ctx, `
		SELECT t.user_id, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance,
		       CASE WHEN v.user_id IS NULL THEN NULL ELSE
		         2 * 6371 * asin(least(1, sqrt(
		           power(sin(radians(t.lat - v.lat) / 2), 2) +
		           cos(radians(v.lat)) * cos(radians(t.lat)) * power(sin(radians(t.lng - v.lng) / 2), 2)))) END
		FROM profiles t
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = $2
		  AND ($1 = $2 OR (
		    NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (t.is_discoverable OR EXISTS (
		      SELECT 1 FROM matches m WHERE m.user_a = least($1::uuid, $2::uuid) AND m.user_b = greatest($1::uuid, $2::uuid))))
		  )
	`, viewerID, targetID).Scan(&p.ID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &showDistance, &dist)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
			return PublicProfile{}, ErrProfileNotFound
		}
		return PublicProfile{}, err
	}
	p.Age = AgeAt(birth, time.Now())
	if dist != nil && showDistance && viewerID != targetID {
		b := DistanceBucket(*dist)
		p.DistanceKm = &b
	}
	photos, interests, err := r.LoadCards(ctx, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	p.Photos, p.Interests = photos[targetID], interests[targetID]
	if p.Photos == nil {
		p.Photos = []Photo{}
	}
	if p.Interests == nil {
		p.Interests = []Interest{}
	}
	return p, nil
}
