package profiles

import (
	"context"
	"errors"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("profile not found")
)

type StoredProfile struct {
	UserID         string
	DisplayName    string
	BirthDate      time.Time
	Gender         string
	Bio            string
	City           string
	HasLocation    bool
	IsDiscoverable bool
	ShowDistance   bool
	Preferences    Preferences
}

type StoredPublic struct {
	UserID      string
	DisplayName string
	BirthDate   time.Time
	Gender      string
	Bio         string
	City        string
	DistanceKm  *float64
}

type Repository interface {
	GetOwn(ctx context.Context, userID string) (StoredProfile, error)
	Save(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	SavePreferences(ctx context.Context, userID string, prefs Preferences) error
	ListInterestCatalog(ctx context.Context) ([]Interest, error)
	GetPublic(ctx context.Context, viewerID, targetID string) (StoredPublic, error)
	PhotoIDs(ctx context.Context, userIDs []string) (map[string][]string, error)
	Interests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id::text, p.display_name, p.birth_date, p.gender, p.bio, p.city,
		       p.latitude IS NOT NULL, p.is_discoverable, p.show_distance,
		       COALESCE(pr.interested_in, '{}'), COALESCE(pr.min_age, 18),
		       COALESCE(pr.max_age, 99), COALESCE(pr.max_distance_km, 50)
		FROM profiles p
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.DisplayName, &p.BirthDate, &p.Gender, &p.Bio, &p.City,
		&p.HasLocation, &p.IsDiscoverable, &p.ShowDistance,
		&p.Preferences.InterestedIn, &p.Preferences.MinAge, &p.Preferences.MaxAge, &p.Preferences.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) Save(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists && in.Latitude == nil {
		return platformerrors.Invalid("location", "is required to create a profile")
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, display_name, birth_date, gender, bio, city, latitude, longitude, is_discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, TRUE), COALESCE($10, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = COALESCE($7, profiles.latitude),
			longitude = COALESCE($8, profiles.longitude),
			is_discoverable = COALESCE($9, profiles.is_discoverable),
			show_distance = COALESCE($10, profiles.show_distance),
			updated_at = NOW()
	`, userID, in.DisplayName, in.BirthDate, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude, in.IsDiscoverable, in.ShowDistance); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}

	var known int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE slug = ANY($1)`, in.InterestSlugs).Scan(&known); err != nil {
		return err
	}
	if known != len(in.InterestSlugs) {
		return platformerrors.Invalid("interests", "contains an unknown interest")
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM user_interests
		WHERE user_id = $1 AND interest_id NOT IN (SELECT id FROM interests WHERE slug = ANY($2))
	`, userID, in.InterestSlugs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
		ON CONFLICT DO NOTHING
	`, userID, in.InterestSlugs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := r.GetOwn(ctx, userID)
	if err != nil {
		return Preferences{}, err
	}
	return p.Preferences, nil
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, prefs Preferences) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		SELECT user_id, $2, $3, $4, $5 FROM profiles WHERE user_id = $1
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, prefs.InterestedIn, prefs.MinAge, prefs.MaxAge, prefs.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ListInterestCatalog(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Interest
	for rows.Next() {
		var it Interest
		if err := rows.Scan(&it.ID, &it.Slug, &it.Label); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetPublic returns another user's profile only when the viewer may see it:
// no block in either direction, and the target is either openly discoverable
// (complete, with a photo) or already matched with the viewer.
func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (StoredPublic, error) {
	var p StoredPublic
	err := r.pool.QueryRow(ctx, `
		SELECT t.user_id::text, t.display_name, t.birth_date, t.gender, t.bio, t.city,
		       CASE WHEN t.show_distance AND v.latitude IS NOT NULL AND t.latitude IS NOT NULL THEN
		         6371 * 2 * asin(sqrt(
		           power(sin(radians(t.latitude - v.latitude) / 2), 2) +
		           cos(radians(v.latitude)) * cos(radians(t.latitude)) *
		           power(sin(radians(t.longitude - v.longitude) / 2), 2)))
		       END
		FROM profiles t
		JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (
		    (t.is_discoverable AND t.latitude IS NOT NULL AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id))
		    OR EXISTS (
		      SELECT 1 FROM matches m
		      WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
	`, viewerID, targetID).Scan(&p.UserID, &p.DisplayName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.DistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPublic{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) PhotoIDs(ctx context.Context, userIDs []string) (map[string][]string, error) {
	return LoadPhotoIDs(ctx, r.pool, userIDs)
}

func (r *PGRepository) Interests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	return LoadInterests(ctx, r.pool, userIDs)
}
