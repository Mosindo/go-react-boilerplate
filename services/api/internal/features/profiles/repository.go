package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository interface {
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	CreateProfile(ctx context.Context, userID string, p storedProfile) error
	UpdateProfile(ctx context.Context, userID, firstName, gender string, bio, city *string, isVisible *bool) error
	SetInterests(ctx context.Context, userID string, slugs []string) (int, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	UserInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	UserPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	// PublicViews returns public profiles of ids (order not guaranteed) with the
	// distance from the viewer, excluding anyone blocked in either direction.
	PublicViews(ctx context.Context, viewerID string, ids []string) ([]publicRow, error)
	// CanView reports whether viewer may open target's profile.
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type publicRow struct {
	UserID     string
	FirstName  string
	BirthDate  time.Time
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
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude IS NOT NULL, is_visible
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.HasLocation, &p.IsVisible)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) CreateProfile(ctx context.Context, userID string, p storedProfile) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, is_visible)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City, p.IsVisible)
	return err
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID, firstName, gender string, bio, city *string, isVisible *bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET first_name = $2, gender = $3,
		  bio = COALESCE($4, bio), city = COALESCE($5, city), is_visible = COALESCE($6, is_visible),
		  updated_at = NOW()
		WHERE user_id = $1
	`, userID, firstName, gender, bio, city, isVisible)
	return err
}

// SetInterests replaces the user's interests and returns how many slugs were known.
func (r *PGRepository) SetInterests(ctx context.Context, userID string, slugs []string) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, slugs)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), tx.Commit(ctx)
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

func (r *PGRepository) UserInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest)
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) UserPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		  max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = COALESCE($4, city), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) PublicViews(ctx context.Context, viewerID string, ids []string) ([]publicRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city,
		  CASE WHEN me.latitude IS NULL OR p.latitude IS NULL THEN NULL ELSE
		    6371 * 2 * asin(sqrt(least(1, power(sin(radians(p.latitude - me.latitude) / 2), 2)
		      + cos(radians(me.latitude)) * cos(radians(p.latitude)) * power(sin(radians(p.longitude - me.longitude) / 2), 2))))
		  END
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []publicRow
	for rows.Next() {
		var row publicRow
		if err := rows.Scan(&row.UserID, &row.FirstName, &row.BirthDate, &row.Gender, &row.Bio, &row.City, &row.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2
		    AND (p.is_visible OR EXISTS (
		      SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)))
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)))
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
