package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("profile not found")

type Repository interface {
	GetOwn(ctx context.Context, userID string) (storedProfile, error)
	SaveProfile(ctx context.Context, p storedProfile) error
	SavePreferences(ctx context.Context, userID string, prefs Preferences) error
	SetInterests(ctx context.Context, userID string, interestIDs []int) error
	CountExistingInterests(ctx context.Context, interestIDs []int) (int, error)
	SetLocation(ctx context.Context, userID string, lat, lon float64, city *string) error
	ClearLocation(ctx context.Context, userID string) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsByUser(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotoIDsByUser(ctx context.Context, userIDs []string) (map[string][]string, error)
	CardRows(ctx context.Context, viewerID string, userIDs []string) ([]cardRow, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.dbPool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.relationship_goal,
		       p.city, p.latitude IS NOT NULL, p.discoverable, p.show_distance,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(
		&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.RelationshipGoal,
		&p.City, &p.HasLocation, &p.Discoverable, &p.ShowDistance,
		&p.Preferences.InterestedIn, &p.Preferences.MinAge, &p.Preferences.MaxAge, &p.Preferences.MaxDistanceKm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) SaveProfile(ctx context.Context, p storedProfile) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET first_name = $2, birthdate = $3, gender = $4, bio = $5, job_title = $6,
		    relationship_goal = $7, city = $8, discoverable = $9, show_distance = $10, updated_at = NOW()
		WHERE user_id = $1
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.JobTitle, p.RelationshipGoal, p.City, p.Discoverable, p.ShowDistance)
	return err
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, prefs Preferences) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, prefs.InterestedIn, prefs.MinAge, prefs.MaxAge, prefs.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetInterests(ctx context.Context, userID string, interestIDs []int) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(interestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])
			ON CONFLICT DO NOTHING
		`, userID, interestIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) CountExistingInterests(ctx context.Context, interestIDs []int) (int, error) {
	var count int
	err := r.dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1::smallint[])`, interestIDs).Scan(&count)
	return count, err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon float64, city *string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = COALESCE($4, city), location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lon, city)
	return err
}

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET latitude = NULL, longitude = NULL, location_updated_at = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *PGRepository) InterestsByUser(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	result := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.dbPool.Query(ctx, `
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
		var userID string
		var i Interest
		if err := rows.Scan(&userID, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], i)
	}
	return result, rows.Err()
}

func (r *PGRepository) PhotoIDsByUser(ctx context.Context, userIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT user_id, id FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, photoID string
		if err := rows.Scan(&userID, &photoID); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], photoID)
	}
	return result, rows.Err()
}

func (r *PGRepository) CardRows(ctx context.Context, viewerID string, userIDs []string) ([]cardRow, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT c.user_id, c.first_name, c.birthdate, c.gender, c.bio, c.job_title, c.relationship_goal,
		       c.city, c.show_distance,
		       CASE WHEN v.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN `+DistanceSQL("v", "c")+` END
		FROM profiles c
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE c.user_id = ANY($2::uuid[])
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]cardRow, 0, len(userIDs))
	for rows.Next() {
		var row cardRow
		if err := rows.Scan(&row.UserID, &row.FirstName, &row.Birthdate, &row.Gender, &row.Bio, &row.JobTitle,
			&row.RelationshipGoal, &row.City, &row.ShowDistance, &row.DistanceKm); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

// CanView: a member can see a profile when nobody blocked the other and the
// profile is either publicly eligible or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles t
			WHERE t.user_id = $2
			  AND `+NotBlockedSQL("$1::uuid", "t.user_id")+`
			  AND (`+EligibleSQL("t")+` OR EXISTS (
				SELECT 1 FROM matches m
				WHERE m.user_low_id = LEAST($1::uuid, t.user_id) AND m.user_high_id = GREATEST($1::uuid, t.user_id)
			  ))
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
