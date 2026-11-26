package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoNotFound        = errors.New("profile not found")
	errRepoUnknownInterest = errors.New("unknown interest slug")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (*storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertProfile(ctx context.Context, p storedProfile, interestSlugs []string) error
	UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng *float64) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error
	ListInterests(ctx context.Context) ([]Interest, error)
	LoadProfiles(ctx context.Context, userIDs []string) (map[string]storedProfile, error)
	LoadPhotos(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error)
	LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const profileColumns = `p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city,
	p.latitude, p.longitude, p.discoverable, p.show_distance, u.is_demo`

func scanProfile(row pgx.Row) (storedProfile, error) {
	var p storedProfile
	err := row.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&p.Latitude, &p.Longitude, &p.Discoverable, &p.ShowDistance, &p.IsDemo)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (*storedProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `
		SELECT `+profileColumns+` FROM profiles p JOIN users u ON u.id = p.user_id WHERE p.user_id = $1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
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

// UpsertProfile writes the profile and replaces its interests atomically.
// It returns how many of the requested interest slugs exist in the catalogue.
func (r *PGRepository) UpsertProfile(ctx context.Context, p storedProfile, interestSlugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.City)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, p.UserID, interestSlugs)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(interestSlugs) {
		return errRepoUnknownInterest
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, prefs.InterestedIn, prefs.MinAge, prefs.MaxAge, prefs.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng *float64) error {
	var updatedAt any
	if lat != nil {
		updatedAt = time.Now()
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = $4, updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return nil
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error {
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
		return errRepoNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.Slug, &i.Label, &i.Category)
		return i, err
	})
}

func (r *PGRepository) LoadProfiles(ctx context.Context, userIDs []string) (map[string]storedProfile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profileColumns+` FROM profiles p JOIN users u ON u.id = p.user_id WHERE p.user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]storedProfile, len(userIDs))
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out[p.UserID] = p
	}
	return out, rows.Err()
}

func (r *PGRepository) LoadPhotos(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, storage_key, width, height, position
		FROM photos WHERE user_id = ANY($1)
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]storedPhoto, len(userIDs))
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Width, &p.Height, &p.Position); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label, i.category
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
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
		if err := rows.Scan(&userID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], i)
	}
	return out, rows.Err()
}

// CanView implements the profile visibility rule: no block in either direction, the target
// is not suspended, and the viewer has a legitimate reason to see them (match, discoverable
// profile, or the target liked the viewer).
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.moderation_status <> 'suspended'
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				$1 = $2
				OR (p.discoverable AND u.moderation_status = 'active')
				OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1, $2)::uuid AND m.user_b_id = GREATEST($1, $2)::uuid)
				OR (
					EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like')
					AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = $2)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
