package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StoredPhoto struct {
	ID       string
	UserID   string
	Position int
	Width    int
	Height   int
}

type StoredInterest struct {
	UserID string
	Interest
}

type ProfilePatch struct {
	FirstName   *string
	Birthdate   *time.Time
	Gender      *string
	Bio         *string
	JobTitle    *string
	InterestIDs *[]int
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	GetProfiles(ctx context.Context, userIDs []string) ([]StoredProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	ListPhotos(ctx context.Context, userIDs []string) ([]StoredPhoto, error)
	ListUserInterests(ctx context.Context, userIDs []string) ([]StoredInterest, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	CountKnownInterests(ctx context.Context, ids []int) (int, error)
	UpdateProfile(ctx context.Context, userID string, patch ProfilePatch) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	UpdateSettings(ctx context.Context, userID string, discoverable, showDistance *bool) error
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const profileColumns = `
	p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city,
	p.latitude, p.longitude, p.discoverable, p.show_distance, u.is_demo`

func scanProfile(row pgx.Row) (StoredProfile, error) {
	var p StoredProfile
	err := row.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&p.Latitude, &p.Longitude, &p.Discoverable, &p.ShowDistance, &p.IsDemo)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	p, err := scanProfile(r.dbPool.QueryRow(ctx, `
		SELECT `+profileColumns+`
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrProfileNotFound
	}
	return p, err
}

func (r *PGRepository) GetProfiles(ctx context.Context, userIDs []string) ([]StoredProfile, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+profileColumns+`
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredProfile{}
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrProfileNotFound
	}
	return p, err
}

func (r *PGRepository) ListPhotos(ctx context.Context, userIDs []string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, user_id, position, width, height
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredPhoto{}
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userIDs []string) ([]StoredInterest, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label, i.category
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredInterest{}
	for rows.Next() {
		var si StoredInterest
		if err := rows.Scan(&si.UserID, &si.ID, &si.Slug, &si.Label, &si.Category); err != nil {
			return nil, err
		}
		out = append(out, si)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id, slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) CountKnownInterests(ctx context.Context, ids []int) (int, error) {
	var n int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1::smallint[])`, ids).Scan(&n)
	return n, err
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID string, patch ProfilePatch) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE profiles SET
			first_name = COALESCE($2, first_name),
			birthdate  = COALESCE($3, birthdate),
			gender     = COALESCE($4, gender),
			bio        = COALESCE($5, bio),
			job_title  = COALESCE($6, job_title),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, patch.FirstName, patch.Birthdate, patch.Gender, patch.Bio, patch.JobTitle)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}

	if patch.InterestIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if len(*patch.InterestIDs) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, unnest($2::smallint[])
				ON CONFLICT DO NOTHING
			`, userID, *patch.InterestIDs); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = COALESCE($4, city),
		       location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdateSettings(ctx context.Context, userID string, discoverable, showDistance *bool) error {
	tag, err := r.dbPool.Exec(ctx, `
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
		return ErrProfileNotFound
	}
	return nil
}

// CanView allows a member to open a profile when no block exists in either direction
// and they are matched, or the target is discoverable with a complete profile.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT
			NOT EXISTS (
				SELECT 1 FROM blocks
				WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
			)
			AND (
				EXISTS (
					SELECT 1 FROM matches
					WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
					  AND ended_at IS NULL
				)
				OR EXISTS (
					SELECT 1 FROM profiles p
					WHERE p.user_id = $2 AND p.discoverable AND p.first_name <> '' AND p.birthdate IS NOT NULL
					  AND p.gender IS NOT NULL AND p.latitude IS NOT NULL
					  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
				)
			)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
