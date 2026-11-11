package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profile not found")
	ErrUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterestCatalog(ctx context.Context) ([]Interest, error)
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)
	ReplaceUserInterests(ctx context.Context, userID string, slugs []string) error
	SetLocation(ctx context.Context, userID string, lat, lng float64) error
	ClearLocation(ctx context.Context, userID string) error
	SetVisibility(ctx context.Context, userID string, discoverable, showDistance *bool) error
	ListPhotos(ctx context.Context, userIDs []string) ([]Photo, error)
	ListProfiles(ctx context.Context, userIDs []string) ([]Profile, error)
	ListInterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	TouchActive(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const profileColumns = `user_id, first_name, birthdate, gender, bio, job_title, city, latitude, longitude, location_updated_at, discoverable, show_distance`

func scanProfile(row pgx.Row) (Profile, error) {
	var p Profile
	err := row.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&p.Latitude, &p.Longitude, &p.LocationUpdatedAt, &p.Discoverable, &p.ShowDistance)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birthdate = EXCLUDED.birthdate,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.City)
	return err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}

func collectInterests(rows pgx.Rows) ([]Interest, error) {
	defer rows.Close()
	out := make([]Interest, 0)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListInterestCatalog(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func (r *PGRepository) ReplaceUserInterests(ctx context.Context, userID string, slugs []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
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
		_, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, slugs)
		return err
	})
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = NULL, longitude = NULL, location_updated_at = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	return err
}

func (r *PGRepository) SetVisibility(ctx context.Context, userID string, discoverable, showDistance *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET
			discoverable = COALESCE($2, discoverable),
			show_distance = COALESCE($3, show_distance),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ListPhotos(ctx context.Context, userIDs []string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, storage_key, position, width, height
		FROM photos WHERE user_id = ANY($1)
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Photo, 0)
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListProfiles(ctx context.Context, userIDs []string) ([]Profile, error) {
	rows, err := r.db.Query(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = ANY($1)`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Profile, 0, len(userIDs))
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListInterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1) ORDER BY i.label
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

// CanView: no block in either direction, target has a profile, and target is
// either discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				p.discoverable OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_low_id = LEAST($1::uuid, $2::uuid) AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < $2
	`, userID, time.Now().Add(-time.Minute))
	return err
}
