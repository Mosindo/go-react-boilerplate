package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

// cardRow is the raw data needed to build a Card for a given viewer.
type cardRow struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     *float64
	Longitude    *float64
	ShowDistance bool
	IsDemo       bool
	LastActiveAt time.Time
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CountInterests(ctx context.Context, ids []int) (int, error)
	ReplaceUserInterests(ctx context.Context, userID string, ids []int) error
	SetLocation(ctx context.Context, userID string, lat, lng *float64) error
	SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error
	TouchActivity(ctx context.Context, userID string) error
	ViewerCoordinates(ctx context.Context, userID string) (*float64, *float64, error)
	CardRows(ctx context.Context, userIDs []string) ([]cardRow, error)
	PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error)
	InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	CanView(ctx context.Context, organizationID, viewerID, targetID string) (bool, error)
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
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, COALESCE(p.city, ''),
		       p.latitude IS NOT NULL, p.is_visible, p.show_distance, u.is_demo, p.created_at, p.updated_at
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.City,
		&p.HasLocation, &p.IsVisible, &p.ShowDistance, &p.IsDemo, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, err
	}
	return p, nil
}

// UpsertProfile creates or updates the profile and guarantees a preferences
// row exists so discovery can always rely on it.
func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name,
		    birthdate = EXCLUDED.birthdate,
		    gender = EXCLUDED.gender,
		    bio = EXCLUDED.bio,
		    city = EXCLUDED.city,
		    updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, p.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var prefs Preferences
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km
		FROM preferences
		WHERE user_id = $1
	`, userID).Scan(&prefs.InterestedIn, &prefs.MinAge, &prefs.MaxAge, &prefs.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrProfileNotFound
		}
		return Preferences{}, err
	}
	return prefs, nil
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, prefs.InterestedIn, prefs.MinAge, prefs.MaxAge, prefs.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interest, 0, 40)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) CountInterests(ctx context.Context, ids []int) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1)`, ids).Scan(&count)
	return count, err
}

func (r *PGRepository) ReplaceUserInterests(ctx context.Context, userID string, ids []int) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])
		`, userID, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng *float64) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2,
		    longitude = $3,
		    location_updated_at = CASE WHEN $2::double precision IS NULL THEN NULL ELSE NOW() END,
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET is_visible = $2, show_distance = $3, updated_at = NOW()
		WHERE user_id = $1
	`, userID, visible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// TouchActivity records coarse activity (at most once per 5 minutes).
func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

func (r *PGRepository) ViewerCoordinates(ctx context.Context, userID string) (*float64, *float64, error) {
	var lat, lng *float64
	err := r.dbPool.QueryRow(ctx, `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, userID).Scan(&lat, &lng)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return lat, lng, nil
}

func (r *PGRepository) CardRows(ctx context.Context, userIDs []string) ([]cardRow, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, COALESCE(p.city, ''),
		       p.latitude, p.longitude, p.show_distance, u.is_demo, p.last_active_at
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]cardRow, 0, len(userIDs))
	for rows.Next() {
		var c cardRow
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.City,
			&c.Latitude, &c.Longitude, &c.ShowDistance, &c.IsDemo, &c.LastActiveAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT user_id, id, position, width, height
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Photo, len(userIDs))
	for rows.Next() {
		var userID string
		var p Photo
		if err := rows.Scan(&userID, &p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest, len(userIDs))
	for rows.Next() {
		var userID string
		var i Interest
		if err := rows.Scan(&userID, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], i)
	}
	return out, rows.Err()
}

// CanView: a profile is visible to a viewer when nobody blocked the other and
// either they are matched, or the profile is discoverable (visible, not
// hidden by moderation, with at least one photo) in the same organization.
func (r *PGRepository) CanView(ctx context.Context, organizationID, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.organization_id = $3
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
			  AND (
				EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_low_id = LEAST($1::uuid, $2::uuid)
					  AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
				)
				OR (
					p.is_visible
					AND p.moderation_hidden_at IS NULL
					AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
				)
			  )
		)
	`, viewerID, targetID, organizationID).Scan(&ok)
	return ok, err
}
