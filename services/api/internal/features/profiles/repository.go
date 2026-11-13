package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

type Repository interface {
	GetOwn(ctx context.Context, userID string) (storedProfile, error)
	ApplyPatch(ctx context.Context, userID string, patch profilePatch) error
	UpdateLocation(ctx context.Context, userID string, lat, lon float64, city string) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]storedCard, error)
	LoadPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	var height *int16
	var ageMin, ageMax, maxDistance int16
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.height_cm,
		       p.relationship_goal, p.city, (p.latitude IS NOT NULL AND p.longitude IS NOT NULL),
		       p.is_paused, p.show_distance, p.is_complete,
		       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(
		&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &height,
		&p.RelationshipGoal, &p.City, &p.HasLocation,
		&p.IsPaused, &p.ShowDistance, &p.IsComplete,
		&p.Preferences.InterestedIn, &ageMin, &ageMax, &maxDistance,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedProfile{}, ErrProfileNotFound
		}
		return storedProfile{}, err
	}
	if height != nil {
		h := int(*height)
		p.HeightCm = &h
	}
	p.Preferences.AgeMin = int(ageMin)
	p.Preferences.AgeMax = int(ageMax)
	p.Preferences.MaxDistanceKm = int(maxDistance)
	if p.Preferences.InterestedIn == nil {
		p.Preferences.InterestedIn = []string{}
	}
	return p, nil
}

func (r *PGRepository) ApplyPatch(ctx context.Context, userID string, patch profilePatch) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE profiles SET
			first_name = COALESCE($2, first_name),
			birthdate = COALESCE($3, birthdate),
			gender = COALESCE($4, gender),
			bio = COALESCE($5, bio),
			job_title = COALESCE($6, job_title),
			height_cm = CASE WHEN $7 THEN NULL ELSE COALESCE($8, height_cm) END,
			relationship_goal = CASE WHEN $9 THEN NULL ELSE COALESCE($10, relationship_goal) END,
			is_paused = COALESCE($11, is_paused),
			show_distance = COALESCE($12, show_distance),
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, patch.FirstName, patch.Birthdate, patch.Gender, patch.Bio, patch.JobTitle,
		patch.ClearHeight, patch.HeightCm, patch.ClearRelationGoal, patch.RelationshipGoal,
		patch.IsPaused, patch.ShowDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}

	if patch.InterestSlugs != nil {
		slugs := *patch.InterestSlugs
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
		if len(slugs) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, id FROM interests WHERE slug = ANY($2)
			`, userID, slugs); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(ctx, `SELECT refresh_profile_completeness($1)`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city string) error {
	tag, err := r.db.Exec(ctx, `
		WITH updated AS (
			UPDATE profiles
			SET latitude = $2, longitude = $3, city = $4, location_updated_at = NOW(), updated_at = NOW()
			WHERE user_id = $1
			RETURNING user_id
		)
		SELECT refresh_profile_completeness(user_id) FROM updated
	`, userID, lat, lon, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, prefs.InterestedIn, prefs.AgeMin, prefs.AgeMax, prefs.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	if _, err := tx.Exec(ctx, `SELECT refresh_profile_completeness($1)`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interest, 0, 48)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// CanView: no block in either direction, and the target is either an active
// match of the viewer or currently discoverable.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			WHERE p.user_id = $2
			  AND $1::uuid <> $2::uuid
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
			  AND p.birthdate IS NOT NULL
			  AND p.gender IS NOT NULL
			  AND (
				EXISTS (
					SELECT 1 FROM matches m
					WHERE m.unmatched_at IS NULL
					  AND m.user_a_id = LEAST($1::uuid, $2::uuid)
					  AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
				)
				OR (p.is_complete AND NOT p.is_paused AND NOT p.moderation_hidden)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// LoadCards returns display data for the given users, with the raw distance
// from the viewer when both have a location.
func (r *PGRepository) LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]storedCard, error) {
	if len(userIDs) == 0 {
		return []storedCard{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.height_cm,
		       p.relationship_goal, p.city,
		       CASE WHEN p.latitude IS NULL OR v.latitude IS NULL THEN NULL ELSE
		         6371 * 2 * asin(sqrt(
		           power(sin(radians(p.latitude - v.latitude) / 2), 2) +
		           cos(radians(v.latitude)) * cos(radians(p.latitude)) *
		           power(sin(radians(p.longitude - v.longitude) / 2), 2)
		         ))
		       END,
		       p.show_distance, u.is_demo, p.last_active_at
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2)
		  AND p.birthdate IS NOT NULL
		  AND p.gender IS NOT NULL
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := make([]storedCard, 0, len(userIDs))
	for rows.Next() {
		var c storedCard
		var height *int16
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &height,
			&c.RelationshipGoal, &c.City, &c.DistanceKm, &c.ShowDistance, &c.IsDemo, &c.LastActiveAt); err != nil {
			return nil, err
		}
		if height != nil {
			h := int(*height)
			c.HeightCm = &h
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *PGRepository) LoadPhotos(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	if len(userIDs) == 0 {
		return []storedPhoto{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, width, height
		FROM photos
		WHERE user_id = ANY($1)
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedPhoto, 0, len(userIDs)*3)
	for rows.Next() {
		var p storedPhoto
		var position int16
		if err := rows.Scan(&p.ID, &p.UserID, &position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		p.Position = int(position)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) LoadInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label, i.category
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY ui.user_id, i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

// TouchActivity records recent activity at most every five minutes.
func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
