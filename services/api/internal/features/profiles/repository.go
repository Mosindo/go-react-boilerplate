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

type StoredPhoto struct {
	ID       string
	UserID   string
	Position int
	Width    int
	Height   int
}

type storedProfile struct {
	Profile
	birthdate time.Time
}

type storedCard struct {
	PublicProfile
	rawDistance  *float64
	showDistance bool
}

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetProfile(ctx context.Context, userID string) (*storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city string) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error
	PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error)
	InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]*storedCard, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
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

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (*storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birthdate, gender, bio, job_title, height_cm, looking_for, city,
		       latitude IS NOT NULL, discoverable, show_distance, updated_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &p.birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.HeightCm, &p.LookingFor, &p.City,
		&p.HasLocation, &p.Discoverable, &p.ShowDistance, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProfile writes the profile and replaces interests atomically. A
// preferences row with sensible defaults is created on first save.
func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, height_cm, looking_for)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name,
		  birthdate = EXCLUDED.birthdate,
		  gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio,
		  job_title = EXCLUDED.job_title,
		  height_cm = EXCLUDED.height_cm,
		  looking_for = EXCLUDED.looking_for,
		  updated_at = NOW()
	`, userID, in.FirstName, in.Birthdate, in.Gender, in.Bio, in.JobTitle, in.HeightCm, in.LookingFor); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, max_distance_km)
		VALUES ($1, ARRAY['man', 'nonbinary', 'woman']::TEXT[], $2)
		ON CONFLICT (user_id) DO NOTHING
	`, userID, DefaultDistanceKm); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2::TEXT[])
		`, userID, in.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.Interests) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = $4,
		    location_updated_at = CASE WHEN $2::DOUBLE PRECISION IS NULL THEN NULL ELSE NOW() END,
		    updated_at = NOW()
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

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET discoverable = COALESCE($2, discoverable),
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

func (r *PGRepository) PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, width, height
		FROM photos WHERE user_id = ANY($1::UUID[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]StoredPhoto, len(userIDs))
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label, i.category
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::UUID[])
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

func (r *PGRepository) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]*storedCard, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, age_years(p.birthdate), p.gender, p.bio, p.job_title, p.height_cm,
		       p.looking_for, p.city, p.show_distance,
		       distance_km(v.latitude, v.longitude, p.latitude, p.longitude)
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::UUID[])
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]*storedCard, len(userIDs))
	for rows.Next() {
		var c storedCard
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.JobTitle, &c.HeightCm,
			&c.LookingFor, &c.City, &c.showDistance, &c.rawDistance); err != nil {
			return nil, err
		}
		out[c.UserID] = &c
	}
	return out, rows.Err()
}

// CanView: no block in either direction, target has a profile, and target
// is either discoverable and not suspended, or already matched with viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
			    $1 = $2
			    OR (p.discoverable AND u.suspended_at IS NULL)
			    OR EXISTS (
			      SELECT 1 FROM matches m
			      WHERE m.user_a_id = LEAST($1::UUID, $2::UUID) AND m.user_b_id = GREATEST($1::UUID, $2::UUID)
			    )
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
