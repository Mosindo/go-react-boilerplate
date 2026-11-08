package profiles

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("repository: not found")

type Repository interface {
	// GetStored loads one profile. viewerID may equal userID (no distance) or be another user.
	GetStored(ctx context.Context, viewerID, userID string) (storedProfile, error)
	GetStoredMany(ctx context.Context, viewerID string, ids []string) ([]storedProfile, error)
	// CanView reports whether viewer may see target: not blocked either way and the target is
	// discoverable or matched with the viewer.
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsFor(ctx context.Context, ids []string) (map[string][]Interest, error)
	PhotosFor(ctx context.Context, ids []string) (map[string][]PhotoRef, error)
	UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const storedSelect = `
	SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city, p.discoverable, p.show_distance,
	       p.latitude IS NOT NULL,
	       CASE WHEN p.user_id <> $1 AND p.show_distance AND p.latitude IS NOT NULL AND me.latitude IS NOT NULL
	            THEN GREATEST(1, ROUND(geo_distance_km(me.latitude, me.longitude, p.latitude, p.longitude)))::int END,
	       (SELECT COUNT(*) FROM photos ph WHERE ph.user_id = p.user_id)::int,
	       COALESCE(pr.interested_in, '{}')
	FROM profiles p
	LEFT JOIN profiles me ON me.user_id = $1
	LEFT JOIN preferences pr ON pr.user_id = p.user_id`

func scanStored(row pgx.Row) (storedProfile, error) {
	var s storedProfile
	err := row.Scan(&s.UserID, &s.FirstName, &s.BirthDate, &s.Gender, &s.Bio, &s.City, &s.Discoverable,
		&s.ShowDistance, &s.HasLocation, &s.DistanceKm, &s.PhotoCount, &s.InterestedIn)
	return s, err
}

func (r *PGRepository) GetStored(ctx context.Context, viewerID, userID string) (storedProfile, error) {
	s, err := scanStored(r.db.QueryRow(ctx, storedSelect+` WHERE p.user_id = $2`, viewerID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) GetStoredMany(ctx context.Context, viewerID string, ids []string) ([]storedProfile, error) {
	rows, err := r.db.Query(ctx, storedSelect+` WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedProfile
	for rows.Next() {
		s, err := scanStored(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles t
		  WHERE t.user_id = $2
		    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (t.discoverable OR EXISTS (
		          SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsFor(ctx context.Context, ids []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Interest{}
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosFor(ctx context.Context, ids []string) (map[string][]PhotoRef, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, position, substr(storage_key, 1, 8) FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]PhotoRef{}
	for rows.Next() {
		var uid string
		var p PhotoRef
		var version string
		if err := rows.Scan(&uid, &p.ID, &p.Position, &version); err != nil {
			return nil, err
		}
		// The version query parameter changes when the file is replaced, so clients may cache aggressively.
		p.URL = "/photos/" + p.ID + "/file?v=" + version
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		UPDATE profiles SET
		  first_name = COALESCE($2, first_name),
		  gender = COALESCE($3, gender),
		  bio = COALESCE($4, bio),
		  city = COALESCE($5, city),
		  discoverable = COALESCE($6, discoverable),
		  show_distance = COALESCE($7, show_distance),
		  updated_at = NOW()
		WHERE user_id = $1`, userID, req.FirstName, req.Gender, req.Bio, req.City, req.Discoverable, req.ShowDistance)
	if err != nil {
		return err
	}
	if req.Interests != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2::text[])`, userID, *req.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(*req.Interests) {
			return fmt.Errorf("unknown interest slug")
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = COALESCE($4, city), updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lng, city)
	return err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}
