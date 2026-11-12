package profiles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HaversineSQL returns an SQL expression computing the great-circle distance in km.
func HaversineSQL(lat1, lng1, lat2, lng2 string) string {
	return fmt.Sprintf(`(12742 * asin(LEAST(1, sqrt(
		power(sin(radians(%[3]s - %[1]s) / 2), 2) +
		cos(radians(%[1]s)) * cos(radians(%[3]s)) * power(sin(radians(%[4]s - %[2]s) / 2), 2)
	))))`, lat1, lng1, lat2, lng2)
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	ReplaceInterests(ctx context.Context, userID string, ids []int) error
	UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city *string) error
	UpdatePrivacy(ctx context.Context, userID string, p Privacy) error
	Touch(ctx context.Context, userID string) error
	InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error)
	PublicRows(ctx context.Context, viewerID string, userIDs []string) (map[string]publicRow, error)
	CanView(ctx context.Context, viewerID, organizationID, targetID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, occupation, relationship_goal, city,
		       latitude, longitude, location_updated_at, discoverable, show_age, show_distance,
		       created_at, updated_at
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.Occupation, &p.RelationshipGoal, &p.City,
		&p.Latitude, &p.Longitude, &p.LocationUpdatedAt, &p.Privacy.Discoverable, &p.Privacy.ShowAge, &p.Privacy.ShowDistance,
		&p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrProfileNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var existing *time.Time
	err = tx.QueryRow(ctx, `SELECT birthdate FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&existing)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if existing != nil && !existing.Equal(in.Birthdate) {
		return ErrBirthdateLocked
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, occupation, relationship_goal, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			occupation = EXCLUDED.occupation,
			relationship_goal = EXCLUDED.relationship_goal,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, userID, in.FirstName, in.Birthdate, in.Gender, in.Bio, in.Occupation, in.RelationshipGoal, in.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p := Preferences{InterestedIn: []string{}, MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: 50}
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	return p, err
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label, &i.Category)
		return i, err
	})
}

func (r *PGRepository) ReplaceInterests(ctx context.Context, userID string, ids []int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(ids) > 0 {
		var known int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE id = ANY($1::smallint[])`, ids).Scan(&known); err != nil {
			return err
		}
		if known != len(ids) {
			return ErrUnknownInterest
		}
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

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng *float64, city *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2,
		    longitude = $3,
		    location_updated_at = CASE WHEN $2::float8 IS NULL THEN NULL ELSE NOW() END,
		    city = COALESCE($4, city),
		    updated_at = NOW()
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

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, p Privacy) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET discoverable = $2, show_age = $3, show_distance = $4, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.Discoverable, p.ShowAge, p.ShowDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (r *PGRepository) Touch(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

func (r *PGRepository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	rows, err := r.db.Query(ctx, `
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
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error) {
	out := make(map[string][]storedPhoto, len(userIDs))
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, storage_key, width, height, position
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.UserID, &p.ID, &p.StorageKey, &p.Width, &p.Height, &p.Position); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) PublicRows(ctx context.Context, viewerID string, userIDs []string) (map[string]publicRow, error) {
	out := make(map[string]publicRow, len(userIDs))
	rows, err := r.db.Query(ctx, `
		WITH viewer AS (
			SELECT latitude, longitude FROM profiles WHERE user_id = $1
		)
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.occupation, p.relationship_goal, p.city,
		       p.show_age, p.show_distance,
		       CASE
		         WHEN v.latitude IS NULL OR p.latitude IS NULL THEN NULL
		         ELSE `+HaversineSQL("v.latitude", "v.longitude", "p.latitude", "p.longitude")+`
		       END AS distance_km
		FROM profiles p
		LEFT JOIN viewer v ON TRUE
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p publicRow
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.Occupation, &p.RelationshipGoal, &p.City,
			&p.ShowAge, &p.ShowDistance, &p.DistanceKm); err != nil {
			return nil, err
		}
		out[p.UserID] = p
	}
	return out, rows.Err()
}

// CanView: same tenant, no block in either direction, and the target is
// either discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, organizationID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.organization_id = $2
			WHERE p.user_id = $3
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $3)
				   OR (b.blocker_user_id = $3 AND b.blocked_user_id = $1)
			  )
			  AND (
				p.discoverable
				OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_low_id = LEAST($1::uuid, $3::uuid) AND m.user_high_id = GREATEST($1::uuid, $3::uuid)
				)
			  )
		)
	`, viewerID, organizationID, targetID).Scan(&ok)
	return ok, err
}
