package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("profile not found")

type Repository interface {
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	UpsertProfile(ctx context.Context, p StoredProfile) error
	GetPreferences(ctx context.Context, userID string) (StoredPreferences, error)
	UpsertPreferences(ctx context.Context, userID string, p StoredPreferences) error
	SetPrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error
	ListInterests(ctx context.Context) ([]Interest, error)
	PhotoIDs(ctx context.Context, userID string) ([]string, error)
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]StoredPublic, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.occupation, p.looking_for, p.city,
		       p.latitude, p.longitude, p.discoverable, p.show_distance,
		       ARRAY(SELECT i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		             WHERE ui.user_id = p.user_id ORDER BY i.slug)
		FROM profiles p
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.Occupation, &p.LookingFor, &p.City,
		&p.Latitude, &p.Longitude, &p.Discoverable, &p.ShowDistance, &p.Interests)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredProfile{}, ErrRepositoryNotFound
		}
		return StoredProfile{}, err
	}
	return p, nil
}

// UpsertProfile writes the profile and replaces the interest set atomically.
// Privacy flags are deliberately not touched here; SetPrivacy owns them.
func (r *PGRepository) UpsertProfile(ctx context.Context, p StoredProfile) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, occupation, looking_for, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name,
		  gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio,
		  occupation = EXCLUDED.occupation,
		  looking_for = EXCLUDED.looking_for,
		  city = EXCLUDED.city,
		  latitude = EXCLUDED.latitude,
		  longitude = EXCLUDED.longitude,
		  updated_at = NOW()
	`, p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.Occupation, p.LookingFor, p.City, p.Latitude, p.Longitude); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(p.Interests) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i.id FROM interests i WHERE i.slug = ANY($2::text[])
		`, p.UserID, p.Interests); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (StoredPreferences, error) {
	var p StoredPreferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPreferences{}, ErrRepositoryNotFound
		}
		return StoredPreferences{}, err
	}
	return p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p StoredPreferences) error {
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

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error {
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
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Interest
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotoIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// PublicByIDs loads many profiles in one query (no N+1). Distance is computed in SQL
// from the stored, already coarse coordinates and only when the target allows it.
func (r *PGRepository) PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]StoredPublic, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.occupation, p.looking_for, p.city,
		       CASE WHEN p.show_distance AND v.user_id IS NOT NULL
		            THEN geo_distance_km(v.latitude, v.longitude, p.latitude, p.longitude) END,
		       ARRAY(SELECT ph.id::text FROM photos ph WHERE ph.user_id = p.user_id ORDER BY ph.position),
		       ARRAY(SELECT i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		             WHERE ui.user_id = p.user_id ORDER BY i.slug)
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StoredPublic
	for rows.Next() {
		var s StoredPublic
		if err := rows.Scan(&s.UserID, &s.FirstName, &s.BirthDate, &s.Gender, &s.Bio, &s.Occupation, &s.LookingFor, &s.City,
			&s.DistanceKm, &s.PhotoIDs, &s.Interests); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CanView decides whether viewer may open target's full profile: never across a
// block, and only if the target is discoverable, matched with the viewer, or has
// already liked the viewer (so the "likes you" list can be opened).
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (p.discoverable
		         OR EXISTS (SELECT 1 FROM matches m
		                    WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
		         OR EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'
	`, userID)
	return err
}
