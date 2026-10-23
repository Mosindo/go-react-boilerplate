package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound         = errors.New("profile not found")
	ErrRepositoryUnknownInterests = errors.New("unknown interests")
)

type Repository interface {
	Get(ctx context.Context, userID string) (StoredProfile, error)
	Upsert(ctx context.Context, p StoredProfile, clearLocation bool, interestSlugs []string) error
	InterestsByUser(ctx context.Context, userID string) ([]Interest, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	SetPreferences(ctx context.Context, userID string, p Preferences) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Get(ctx context.Context, userID string) (StoredProfile, error) {
	p := StoredProfile{UserID: userID}
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, latitude, longitude
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) Upsert(ctx context.Context, p StoredProfile, clearLocation bool, slugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Omitted coordinates keep the stored ones; clearLocation removes them.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name,
		  birth_date = EXCLUDED.birth_date,
		  gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio,
		  city = EXCLUDED.city,
		  latitude = CASE WHEN $9 THEN NULL ELSE COALESCE(EXCLUDED.latitude, profiles.latitude) END,
		  longitude = CASE WHEN $9 THEN NULL ELSE COALESCE(EXCLUDED.longitude, profiles.longitude) END,
		  updated_at = NOW()
	`, p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City, p.Latitude, p.Longitude, clearLocation); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(slugs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2::text[])
		`, p.UserID, slugs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(slugs) {
			return ErrRepositoryUnknownInterests
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) InterestsByUser(ctx context.Context, userID string) ([]Interest, error) {
	return r.queryInterests(ctx, `
		SELECT i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label
	`, userID)
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	return r.queryInterests(ctx, `SELECT slug, label FROM interests ORDER BY label`)
}

func (r *PGRepository) queryInterests(ctx context.Context, sql string, args ...any) ([]Interest, error) {
	rows, err := r.db.Query(ctx, sql, args...)
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

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) SetPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
		  interested_in = EXCLUDED.interested_in,
		  min_age = EXCLUDED.min_age,
		  max_age = EXCLUDED.max_age,
		  max_distance_km = EXCLUDED.max_distance_km,
		  discoverable = EXCLUDED.discoverable,
		  show_distance = EXCLUDED.show_distance,
		  updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.Discoverable, p.ShowDistance)
	return err
}
