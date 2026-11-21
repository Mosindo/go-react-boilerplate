package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrUserNotFound    = errors.New("user not found")
)

type Repository interface {
	GetEmail(ctx context.Context, userID string) (string, error)
	GetProfile(ctx context.Context, userID string) (*Profile, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	ListOwnPhotos(ctx context.Context, userID string) ([]Photo, error)
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	UpsertProfile(ctx context.Context, userID string, input ProfileInput) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	ReplaceInterests(ctx context.Context, userID string, interestIDs []int) error
	SetLocation(ctx context.Context, userID string, lat, lng *float64) error
	UpdateSettings(ctx context.Context, userID string, isVisible, showDistance *bool) error
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]CardRecord, error)
	TouchActive(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) GetEmail(ctx context.Context, userID string) (string, error) {
	var email string
	err := r.dbPool.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, userID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return email, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	var p Profile
	err := r.dbPool.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, city, job_title, height_cm::int,
		       relationship_goal, latitude IS NOT NULL, is_visible, show_distance, is_demo, created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.City, &p.JobTitle, &p.HeightCm,
		&p.RelationshipGoal, &p.HasLocation, &p.IsVisible, &p.ShowDistance, &p.IsDemo, &p.CreatedAt, &p.UpdatedAt)
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
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, min_age::int, max_age::int, max_distance_km::int
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) ListOwnPhotos(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, position::int, width, height FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	photos := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT i.id::int, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1
		ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id::int, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func collectInterests(rows pgx.Rows) ([]Interest, error) {
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, city, job_title, height_cm, relationship_goal)
		SELECT id, $2, $3, $4, $5, $6, $7, $8, $9 FROM users WHERE id = $1
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birthdate = EXCLUDED.birthdate,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			job_title = EXCLUDED.job_title,
			height_cm = EXCLUDED.height_cm,
			relationship_goal = EXCLUDED.relationship_goal,
			updated_at = NOW()
	`, userID, in.FirstName, in.Birthdate, in.Gender, in.Bio, in.City, in.JobTitle, in.HeightCm, in.RelationshipGoal)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	// Keep the legacy users.display_name in sync for generic modules.
	if _, err := tx.Exec(ctx, `UPDATE users SET display_name = $2, updated_at = NOW() WHERE id = $1`, userID, in.FirstName); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) ReplaceInterests(ctx context.Context, userID string, interestIDs []int) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(interestIDs) > 0 {
		var known int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1::smallint[])`, interestIDs).Scan(&known); err != nil {
			return err
		}
		if known != len(interestIDs) {
			return ErrUnknownInterest
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(interestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])
		`, userID, interestIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng *float64) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3,
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

func (r *PGRepository) UpdateSettings(ctx context.Context, userID string, isVisible, showDistance *bool) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET is_visible = COALESCE($2, is_visible),
		    show_distance = COALESCE($3, show_distance),
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, isVisible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

// CanView enforces who may open a full profile: same tenant, no block in
// either direction, and either an active match or a visible, non-suspended
// profile.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var allowed bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles t
			JOIN users tu ON tu.id = t.user_id
			JOIN users vu ON vu.id = $1
			WHERE t.user_id = $2
			  AND tu.organization_id = vu.organization_id
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
			  AND (
				t.user_id = $1
				OR (t.is_visible AND NOT t.is_suspended)
				OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_low_id = LEAST($1::uuid, $2::uuid) AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&allowed)
	return allowed, err
}

// LoadCards loads public profiles for userIDs in three queries (profiles,
// photos, interests) regardless of the number of ids: no N+1.
func (r *PGRepository) LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]CardRecord, error) {
	if len(userIDs) == 0 {
		return []CardRecord{}, nil
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT p.user_id, p.first_name, date_part('year', age(p.birthdate))::int, p.gender, p.bio, p.city, p.job_title,
		       p.height_cm::int, p.relationship_goal, p.is_demo,
		       CASE WHEN p.show_distance AND v.latitude IS NOT NULL AND p.latitude IS NOT NULL
		            THEN haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*CardRecord, len(userIDs))
	for rows.Next() {
		var c CardRecord
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.JobTitle, &c.HeightCm,
			&c.RelationshipGoal, &c.IsDemo, &c.DistanceKm); err != nil {
			rows.Close()
			return nil, err
		}
		c.Photos = []Photo{}
		c.Interests = []Interest{}
		byID[c.UserID] = &c
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	photoRows, err := r.dbPool.Query(ctx, `
		SELECT user_id, id, position::int, width, height
		FROM photos WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	for photoRows.Next() {
		var userID string
		var p Photo
		if err := photoRows.Scan(&userID, &p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			photoRows.Close()
			return nil, err
		}
		if c, ok := byID[userID]; ok {
			c.Photos = append(c.Photos, p)
		}
	}
	photoRows.Close()
	if err := photoRows.Err(); err != nil {
		return nil, err
	}

	interestRows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.id::int, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	for interestRows.Next() {
		var userID string
		var i Interest
		if err := interestRows.Scan(&userID, &i.ID, &i.Slug, &i.Label); err != nil {
			interestRows.Close()
			return nil, err
		}
		if c, ok := byID[userID]; ok {
			c.Interests = append(c.Interests, i)
		}
	}
	interestRows.Close()
	if err := interestRows.Err(); err != nil {
		return nil, err
	}

	out := make([]CardRecord, 0, len(userIDs))
	for _, id := range userIDs {
		if c, ok := byID[id]; ok {
			out = append(out, *c)
		}
	}
	return out, nil
}

// TouchActive refreshes last_active_at at most every 5 minutes to avoid
// write amplification on hot paths.
func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < $2
	`, userID, time.Now().Add(-5*time.Minute))
	return err
}
