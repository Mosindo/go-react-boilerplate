package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository interface {
	Get(ctx context.Context, userID string) (storedProfile, error)
	Upsert(ctx context.Context, p storedProfile, interestSlugs []string) error
	UpdateSettings(ctx context.Context, userID string, visible, showDistance *bool) error
	ListInterests(ctx context.Context) ([]InterestRef, error)
	InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]InterestRef, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Get(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city, p.latitude, p.longitude,
		       p.is_visible, p.show_distance, p.created_at, p.updated_at,
		       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude,
		&p.IsVisible, &p.ShowDistance, &p.CreatedAt, &p.UpdatedAt,
		&p.Preferences.InterestedIn, &p.Preferences.AgeMin, &p.Preferences.AgeMax, &p.Preferences.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) Upsert(ctx context.Context, p storedProfile, interestSlugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City, p.Latitude, p.Longitude); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, p.UserID, p.Preferences.InterestedIn, p.Preferences.AgeMin, p.Preferences.AgeMax, p.Preferences.MaxDistanceKm); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(interestSlugs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, p.UserID, interestSlugs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateSettings(ctx context.Context, userID string, visible, showDistance *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET
			is_visible = COALESCE($2, is_visible),
			show_distance = COALESCE($3, show_distance),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, visible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]InterestRef, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InterestRef{}
	for rows.Next() {
		var i InterestRef
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]InterestRef, error) {
	out := make(map[string][]InterestRef, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.id
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i InterestRef
		if err := rows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}
