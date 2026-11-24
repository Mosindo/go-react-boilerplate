package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("profile not found")

// CardRow is the raw data needed to render a profile card for a viewer.
type CardRow struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal string
	City             string
	ShowDistance     bool
	DistanceKm       *float64
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	UserInterests(ctx context.Context, userID string) ([]Interest, error)
	ReplaceInterests(ctx context.Context, userID string, ids []int) error
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	CardRows(ctx context.Context, viewerID string, targetIDs []string) ([]CardRow, error)
	CardInterests(ctx context.Context, viewerID string, targetIDs []string) (map[string][]CardInterest, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, job_title, height_cm, relationship_goal,
		       city, latitude, longitude, is_discoverable, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.HeightCm,
		&p.RelationshipGoal, &p.City, &p.Latitude, &p.Longitude, &p.IsDiscoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, height_cm, relationship_goal, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birthdate = EXCLUDED.birthdate,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			height_cm = EXCLUDED.height_cm,
			relationship_goal = EXCLUDED.relationship_goal,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.HeightCm, p.RelationshipGoal, p.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, p.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = COALESCE($4, city), location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{InterestedIn: append([]string(nil), Genders...), MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: 50}, nil
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func scanInterests(rows pgx.Rows) ([]Interest, error) {
	defer rows.Close()
	out := make([]Interest, 0)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	return scanInterests(rows)
}

func (r *PGRepository) UserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, i.slug, i.label, i.category
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1
		ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return scanInterests(rows)
}

func (r *PGRepository) ReplaceInterests(ctx context.Context, userID string, ids []int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var known int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1::smallint[])`, ids).Scan(&known); err != nil {
		return err
	}
	if known != len(ids) {
		return ErrUnknownInterest
	}
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

// CanView: a profile is visible when neither member blocked the other and the
// target is discoverable, or the two members are matched.
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
				p.is_discoverable OR $1 = $2 OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_low_id = LEAST($1::uuid, $2::uuid) AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) CardRows(ctx context.Context, viewerID string, targetIDs []string) ([]CardRow, error) {
	if len(targetIDs) == 0 {
		return []CardRow{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.height_cm,
		       p.relationship_goal, p.city, p.show_distance,
		       CASE WHEN v.latitude IS NOT NULL AND p.latitude IS NOT NULL
		            THEN distance_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, targetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CardRow, 0, len(targetIDs))
	for rows.Next() {
		var c CardRow
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &c.HeightCm,
			&c.RelationshipGoal, &c.City, &c.ShowDistance, &c.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) CardInterests(ctx context.Context, viewerID string, targetIDs []string) (map[string][]CardInterest, error) {
	result := make(map[string][]CardInterest, len(targetIDs))
	if len(targetIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label, i.category,
		       EXISTS (SELECT 1 FROM user_interests mine WHERE mine.user_id = $1 AND mine.interest_id = i.id)
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($2::uuid[])
		ORDER BY ui.user_id, i.label
	`, viewerID, targetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var ci CardInterest
		if err := rows.Scan(&userID, &ci.ID, &ci.Slug, &ci.Label, &ci.Category, &ci.Shared); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], ci)
	}
	return result, rows.Err()
}
