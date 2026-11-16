package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type storedProfile struct {
	UserID         string
	FirstName      string
	Birthdate      time.Time
	Gender         string
	Bio            string
	JobTitle       string
	HeightCm       *int
	LookingFor     string
	City           string
	Latitude       *float64
	Longitude      *float64
	IsDiscoverable bool
}

type storedPhoto struct {
	ID       string
	UserID   string
	Position int
	Width    int
	Height   int
}

type storedCardInterest struct {
	UserID string
	CardInterest
}

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	UpsertProfile(ctx context.Context, userID string, rec profileRecord) error
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64) error
	SetDiscoverable(ctx context.Context, userID string, discoverable bool) error
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)

	LoadProfiles(ctx context.Context, userIDs []string) ([]storedProfile, error)
	LoadPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	LoadCardInterests(ctx context.Context, viewerID string, userIDs []string) ([]storedCardInterest, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
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

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, rec profileRecord) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, height_cm, looking_for, city)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (user_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				birthdate = EXCLUDED.birthdate,
				gender = EXCLUDED.gender,
				bio = EXCLUDED.bio,
				job_title = EXCLUDED.job_title,
				height_cm = EXCLUDED.height_cm,
				looking_for = EXCLUDED.looking_for,
				city = EXCLUDED.city,
				updated_at = NOW()
		`, userID, rec.FirstName, rec.Birthdate, rec.Gender, rec.Bio, rec.JobTitle, rec.HeightCm, rec.LookingFor, rec.City); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if len(rec.InterestIDs) > 0 {
			// Unknown ids are silently ignored by the join on interests.
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, id FROM interests WHERE id = ANY($2::smallint[])
			`, userID, rec.InterestIDs); err != nil {
				return err
			}
		}
		return nil
	})
}

const profileColumns = `user_id, first_name, birthdate, gender, bio, job_title, height_cm, looking_for, city, latitude, longitude, is_discoverable`

func scanProfile(row pgx.Row) (storedProfile, error) {
	var p storedProfile
	err := row.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.HeightCm, &p.LookingFor, &p.City, &p.Latitude, &p.Longitude, &p.IsDiscoverable)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (storedProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
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

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
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

func (r *PGRepository) SetDiscoverable(ctx context.Context, userID string, discoverable bool) error {
	tag, err := r.db.Exec(ctx, `UPDATE profiles SET is_discoverable = $2, updated_at = NOW() WHERE user_id = $1`, userID, discoverable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label)
		return i, err
	})
}

func (r *PGRepository) LoadProfiles(ctx context.Context, userIDs []string) ([]storedProfile, error) {
	rows, err := r.db.Query(ctx, `SELECT `+profileColumns+` FROM profiles WHERE user_id = ANY($1::uuid[])`, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedProfile, error) { return scanProfile(row) })
}

func (r *PGRepository) LoadPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, width, height FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedPhoto, error) {
		var p storedPhoto
		err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.Width, &p.Height)
		return p, err
	})
}

func (r *PGRepository) LoadCardInterests(ctx context.Context, viewerID string, userIDs []string) ([]storedCardInterest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.label, (vi.interest_id IS NOT NULL) AS shared
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		LEFT JOIN user_interests vi ON vi.user_id = $1 AND vi.interest_id = ui.interest_id
		WHERE ui.user_id = ANY($2::uuid[])
		ORDER BY ui.user_id, shared DESC, i.label
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedCardInterest, error) {
		var s storedCardInterest
		err := row.Scan(&s.UserID, &s.ID, &s.Label, &s.Shared)
		return s, err
	})
}

// CanView: a profile is visible when neither user blocked the other and the
// target is either discoverable (with a photo) or already matched with the viewer.
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
				$1 = $2
				OR (p.is_discoverable AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))
				OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.unmatched_at IS NULL
					  AND m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
