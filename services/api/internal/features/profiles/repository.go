package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("not found")

type StoredPhoto struct {
	ID       string
	Position int
	Width    int
	Height   int
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error
	UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city *string) error
	ListInterests(ctx context.Context) ([]Interest, error)
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)
	ReplaceUserInterests(ctx context.Context, userID string, slugs []string) error
	ListPhotos(ctx context.Context, userID string) ([]StoredPhoto, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	var height *int16
	err := r.dbPool.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal,
		       latitude, longitude, location_updated_at, is_discoverable, show_distance, moderation_hidden
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City, &height,
		&p.RelationshipGoal, &p.Latitude, &p.Longitude, &p.LocationUpdatedAt, &p.IsDiscoverable, &p.ShowDistance, &p.ModerationHidden)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrRepositoryNotFound
	}
	if height != nil {
		h := int(*height)
		p.HeightCm = &h
	}
	return p, err
}

// UpsertProfile never updates birthdate on conflict: it is immutable.
func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			city = EXCLUDED.city,
			height_cm = EXCLUDED.height_cm,
			relationship_goal = EXCLUDED.relationship_goal,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.City, p.HeightCm, p.RelationshipGoal)
	return err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET
			is_discoverable = COALESCE($2, is_discoverable),
			show_distance = COALESCE($3, show_distance),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

// UpdateLocation stores already-rounded coordinates; nil clears the location.
func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city *string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET
			latitude = $2,
			longitude = $3,
			location_updated_at = CASE WHEN $2::double precision IS NULL THEN NULL ELSE NOW() END,
			city = COALESCE($4, city),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	return r.queryInterests(ctx, `SELECT slug, label, category FROM interests ORDER BY category, label`)
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	return r.queryInterests(ctx, `
		SELECT i.slug, i.label, i.category
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1
		ORDER BY i.label
	`, userID)
}

func (r *PGRepository) queryInterests(ctx context.Context, query string, args ...any) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interest, 0)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) ReplaceUserInterests(ctx context.Context, userID string, slugs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var known int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE slug = ANY($1)`, slugs).Scan(&known); err != nil {
		return err
	}
	if known != len(slugs) {
		return ErrUnknownInterest
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, slugs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ListPhotos(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, position, width, height FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StoredPhoto, 0)
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
