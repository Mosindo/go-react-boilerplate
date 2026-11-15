package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CompleteProfileSQL is the SQL predicate (on alias p) for a profile that may
// appear to others. Shared with the matching feature.
const CompleteProfileSQL = `(p.first_name <> '' AND p.birthdate IS NOT NULL AND p.gender IS NOT NULL
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))`

type CardRow struct {
	UserID       string
	FirstName    string
	Age          int
	Gender       string
	Bio          string
	JobTitle     string
	HeightCm     *int
	City         string
	HideDistance bool
	IsDemo       bool
	DistanceKm   *float64
}

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	CountInterests(ctx context.Context, ids []int) (int, error)
	GetOwnProfile(ctx context.Context, userID string) (OwnProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdateProfile(ctx context.Context, userID string, update ProfileUpdate) error
	UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city string) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, isPaused, hideDistance bool) error
	IsVisibleTo(ctx context.Context, viewerID, targetID string) (bool, error)
	LoadCardRows(ctx context.Context, viewerID string, userIDs []string) ([]CardRow, error)
	LoadPhotos(ctx context.Context, userIDs []string) (map[string][]Photo, error)
	LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label)
		return i, err
	})
}

func (r *PGRepository) CountInterests(ctx context.Context, ids []int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1)`, ids).Scan(&n)
	return n, err
}

func (r *PGRepository) GetOwnProfile(ctx context.Context, userID string) (OwnProfile, error) {
	var p OwnProfile
	var height *int16
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, job_title, height_cm, city,
		       approx_latitude IS NOT NULL, is_paused, hide_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &height, &p.City, &p.HasLocation, &p.IsPaused, &p.HideDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}
	if height != nil {
		h := int(*height)
		p.HeightCm = &h
	}
	photos, err := r.LoadPhotos(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := r.LoadInterests(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	p.Photos = photos[userID]
	p.Interests = interests[userID]
	return p, nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	var minAge, maxAge, dist int16
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &minAge, &maxAge, &dist)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrProfileNotFound
		}
		return Preferences{}, err
	}
	p.MinAge, p.MaxAge, p.MaxDistanceKm = int(minAge), int(maxAge), int(dist)
	return p, nil
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID string, u ProfileUpdate) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE profiles
		SET first_name = $2, birthdate = $3, gender = $4, bio = $5, job_title = $6,
		    height_cm = $7, city = $8, updated_at = NOW()
		WHERE user_id = $1
	`, userID, u.FirstName, u.Birthdate, u.Gender, u.Bio, u.JobTitle, u.HeightCm, u.City)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1 AND NOT (interest_id = ANY($2))`, userID, u.InterestIDs); err != nil {
		return err
	}
	if len(u.InterestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])
			ON CONFLICT DO NOTHING
		`, userID, u.InterestIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city string) error {
	var updatedAt *time.Time
	if lat != nil {
		now := time.Now()
		updatedAt = &now
	}
	_, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET approx_latitude = $2, approx_longitude = $3, location_updated_at = $4,
		    city = CASE WHEN $5 = '' THEN city ELSE $5 END, updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, updatedAt, city)
	return err
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, isPaused, hideDistance bool) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_paused = $2, hide_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, isPaused, hideDistance)
	return err
}

func (r *PGRepository) IsVisibleTo(ctx context.Context, viewerID, targetID string) (bool, error) {
	var visible bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid))
				OR (p.is_paused = FALSE AND `+CompleteProfileSQL+`)
			  )
		)
	`, viewerID, targetID).Scan(&visible)
	return visible, err
}

func (r *PGRepository) LoadCardRows(ctx context.Context, viewerID string, userIDs []string) ([]CardRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, COALESCE(date_part('year', age(p.birthdate))::int, 0),
		       COALESCE(p.gender, ''), p.bio, p.job_title, p.height_cm, p.city, p.hide_distance, u.is_demo,
		       CASE WHEN v.approx_latitude IS NOT NULL AND p.approx_latitude IS NOT NULL
		            THEN distance_km(v.approx_latitude, v.approx_longitude, p.approx_latitude, p.approx_longitude)
		       END
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2)
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CardRow, error) {
		var c CardRow
		var height *int16
		err := row.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.JobTitle, &height, &c.City, &c.HideDistance, &c.IsDemo, &c.DistanceKm)
		if height != nil {
			h := int(*height)
			c.HeightCm = &h
		}
		return c, err
	})
}

func (r *PGRepository) LoadPhotos(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, storage_key, width, height, position
		FROM photos WHERE user_id = ANY($1)
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Photo)
	for rows.Next() {
		var userID string
		var p Photo
		var pos int16
		if err := rows.Scan(&userID, &p.ID, &p.StorageKey, &p.Width, &p.Height, &pos); err != nil {
			return nil, err
		}
		p.Position = int(pos)
		out[userID] = append(out[userID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest)
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
