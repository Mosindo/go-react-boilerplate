package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profiles: not found")
	ErrPhotoLimit      = errors.New("profiles: photo limit reached")
	ErrUnknownInterest = errors.New("profiles: unknown interest")
	ErrBadPhotoSet     = errors.New("profiles: ids must match the user's photos")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosFor(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error)
	Catalog(ctx context.Context) ([]Interest, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	AddPhoto(ctx context.Context, userID, key string, size, w, h int) (storedPhoto, error)
	DeletePhoto(ctx context.Context, userID, photoID string) (string, error)
	ReorderPhotos(ctx context.Context, userID string, ids []string) error
	PhotoKey(ctx context.Context, photoID string) (string, error)
	// Cards returns display data keyed by user id, with distance computed relative to the viewer.
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]cardRow, error)
	// CanView applies visibility rules: no block in either direction and the target is a complete,
	// discoverable profile, or already connected (match / incoming like).
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type cardRow struct {
	UserID     string
	FirstName  string
	Age        int
	Gender     string
	Bio        string
	City       string
	DistanceKm *float64
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude IS NOT NULL, show_distance, discoverable
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.HasLocation, &p.ShowDistance, &p.Discoverable)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY ui.user_id, i.label`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest)
	for rows.Next() {
		var uid string
		var it Interest
		if err := rows.Scan(&uid, &it.Slug, &it.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], it)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosFor(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, storage_key FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]storedPhoto)
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.Key); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Catalog(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var it Interest
		if err := rows.Scan(&it.Slug, &it.Label); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, show_distance, discoverable)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (user_id) DO UPDATE SET first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city, latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude, show_distance = EXCLUDED.show_distance,
			discoverable = EXCLUDED.discoverable, updated_at = NOW()`,
		userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude, in.ShowDistance, in.Discoverable); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()`,
		userID, in.Preferences.InterestedIn, in.Preferences.AgeMin, in.Preferences.AgeMax, in.Preferences.MaxDistanceKm); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestSlugs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2::text[])`, userID, in.InterestSlugs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.InterestSlugs) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) AddPhoto(ctx context.Context, userID, key string, size, w, h int) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialise concurrent uploads of the same user so the limit cannot be bypassed.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return storedPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotos {
		return storedPhoto{}, ErrPhotoLimit
	}
	p := storedPhoto{UserID: userID, Key: key, Position: count}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, size_bytes, width, height)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, userID, count, key, size, w, h).Scan(&p.ID); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) DeletePhoto(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = s.rn FROM (
			SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::int AS rn FROM photos WHERE user_id = $1
		) s WHERE p.id = s.id`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) ReorderPhotos(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrBadPhotoSet
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = (t.ord - 1)::int
		FROM unnest($2::uuid[]) WITH ORDINALITY AS t(id, ord)
		WHERE p.id = t.id AND p.user_id = $1`, userID, ids)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(ids) {
		return ErrBadPhotoSet
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) PhotoKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

func (r *PGRepository) Cards(ctx context.Context, viewerID string, ids []string) (map[string]cardRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, date_part('year', age(p.birth_date))::int, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		            THEN geo_distance_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]cardRow, len(ids))
	for rows.Next() {
		var c cardRow
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.DistanceKm); err != nil {
			return nil, err
		}
		out[c.UserID] = c
	}
	return out, rows.Err()
}

func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (p.discoverable
		         OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid))
		         OR EXISTS (SELECT 1 FROM likes l WHERE l.from_user_id = $2 AND l.to_user_id = $1 AND l.action = 'like'))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

var _ = time.Now
