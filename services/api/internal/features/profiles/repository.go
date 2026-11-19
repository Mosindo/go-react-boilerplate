package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
}

type cardRow struct {
	StoredProfile
	ShowDistance bool
}

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertProfile(ctx context.Context, userID string, p normalizedProfile) error
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	ClearLocation(ctx context.Context, userID string) error
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosFor(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error)
	CardRows(ctx context.Context, userIDs []string) ([]cardRow, error)
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

const profileColumns = `p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.looking_for, p.city, p.latitude, p.longitude`

func scanProfile(row pgx.Row, extra ...any) (StoredProfile, error) {
	var p StoredProfile
	dest := append([]any{&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.LookingFor, &p.City, &p.Latitude, &p.Longitude}, extra...)
	err := row.Scan(dest...)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles p WHERE p.user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km, is_discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm, &p.IsDiscoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrNotFound
	}
	return p, err
}

// UpsertProfile creates or updates the profile. The birthdate is only written on
// creation: it cannot be changed afterwards to prevent age manipulation.
func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, p normalizedProfile) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, looking_for, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			looking_for = EXCLUDED.looking_for,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, userID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.LookingFor, p.City); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in) VALUES ($1, ARRAY['woman','man','nonbinary'])
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(p.InterestIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i.id FROM interests i WHERE i.id = ANY($2::smallint[])
		`, userID, p.InterestIDs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(p.InterestIDs) {
			return ErrInvalidInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5,
		    is_discoverable = $6, show_distance = $7, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm, p.IsDiscoverable, p.ShowDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = COALESCE($4, city),
		    location_updated_at = NOW(), updated_at = NOW()
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

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = NULL, longitude = NULL, location_updated_at = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	return err
}

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosFor(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error) {
	out := make(map[string][]StoredPhoto, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, storage_key, position, width, height
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) CardRows(ctx context.Context, userIDs []string) ([]cardRow, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+profileColumns+`, COALESCE(pr.show_distance, TRUE)
		FROM profiles p
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []cardRow
	for rows.Next() {
		var c cardRow
		p, err := scanProfile(rows, &c.ShowDistance)
		if err != nil {
			return nil, err
		}
		c.StoredProfile = p
		out = append(out, c)
	}
	return out, rows.Err()
}

// CanView: a profile is visible when neither user blocked the other and either the
// users have an active match, or the target is discoverable with at least one photo.
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
				EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
					  AND m.unmatched_at IS NULL
				)
				OR (
					EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = $2 AND pr.is_discoverable)
					AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = $2)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// timeNow is overridable in tests.
var timeNow = time.Now
