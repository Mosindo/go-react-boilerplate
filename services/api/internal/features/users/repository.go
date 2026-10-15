package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound        = errors.New("users: not found")
	ErrRepositoryUnknownInterest = errors.New("users: unknown interest")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	SaveProfile(ctx context.Context, p Profile, interestSlugs []string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	SavePreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	ClearLocation(ctx context.Context, userID string) error
	SetPrivacy(ctx context.Context, userID string, isVisible, showDistance *bool) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotoCount(ctx context.Context, userID string) (int, error)
	TouchActive(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.dbPool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude, longitude, is_visible, show_distance
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.IsVisible, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrRepositoryNotFound
	}
	return p, err
}

// SaveProfile upserts the profile and replaces the interests atomically.
// The birth date is only written on creation: it cannot be edited afterwards.
func (r *PGRepository) SaveProfile(ctx context.Context, p Profile, interestSlugs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name,
		    gender = EXCLUDED.gender,
		    bio = EXCLUDED.bio,
		    updated_at = NOW()
	`, p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(interestSlugs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, p.UserID, interestSlugs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(interestSlugs) {
			return ErrRepositoryUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km
		FROM preferences
		WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in,
		    min_age = EXCLUDED.min_age,
		    max_age = EXCLUDED.max_age,
		    max_distance_km = EXCLUDED.max_distance_km,
		    updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = $4, updated_at = NOW()
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

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = NULL, longitude = NULL, city = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, isVisible, showDistance *bool) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET is_visible = COALESCE($2, is_visible),
		    show_distance = COALESCE($3, show_distance),
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, isVisible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Interest, 0, 32)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// InterestsForUsers loads interests for many users in one query (no N+1).
func (r *PGRepository) InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	result := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var i Interest
		if err := rows.Scan(&userID, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], i)
	}
	return result, rows.Err()
}

func (r *PGRepository) PhotoCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
