package profiles

import (
	"context"
	"errors"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	UpsertProfile(ctx context.Context, userID string, p storedProfile, interestSlugs []string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64) (bool, error)
	IsComplete(ctx context.Context, userID string) (bool, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	ListByIDs(ctx context.Context, viewerID string, ids []string) ([]Profile, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
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

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, p storedProfile, slugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio, city = EXCLUDED.city, updated_at = NOW()
	`, userID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(slugs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, slugs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(slugs) {
			return platformerrors.BadRequest("unknown interest")
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultPreferences(), nil
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
		  interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age, max_age = EXCLUDED.max_age,
		  max_distance_km = EXCLUDED.max_distance_km, discoverable = EXCLUDED.discoverable,
		  show_distance = EXCLUDED.show_distance, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.Discoverable, p.ShowDistance)
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng)
	return tag.RowsAffected() == 1, err
}

func (r *PGRepository) IsComplete(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $1 AND p.latitude IS NOT NULL
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		)`, userID).Scan(&ok)
	return ok, err
}

// CanView: the target has a profile with a photo, nobody blocked anybody, and
// the target is discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  JOIN preferences pr ON pr.user_id = p.user_id
		  WHERE p.user_id = $2
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (pr.discoverable OR EXISTS (
		      SELECT 1 FROM matches m
		      WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// ListByIDs hydrates profiles in 3 queries (profiles, interests, photos) no
// matter how many ids are requested, preserving the order of ids.
func (r *PGRepository) ListByIDs(ctx context.Context, viewerID string, ids []string) ([]Profile, error) {
	if len(ids) == 0 {
		return []Profile{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.user_id::text, p.first_name, p.birth_date, p.gender, p.bio, p.city, pr.show_distance,
		       CASE WHEN me.latitude IS NULL OR p.latitude IS NULL THEN NULL
		            ELSE `+DistanceKmSQL("p", "me")+` END
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[string]*Profile, len(ids))
	now := time.Now()
	for rows.Next() {
		var (
			p        Profile
			birth    time.Time
			show     bool
			distance *float64
		)
		if err := rows.Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &show, &distance); err != nil {
			return nil, err
		}
		p.Age = AgeOn(birth, now)
		p.Interests = []Interest{}
		p.Photos = []Photo{}
		if p.UserID == viewerID {
			p.BirthDate = birth.Format("2006-01-02")
		} else if show && distance != nil {
			bucket := BucketDistance(*distance)
			p.DistanceKm = &bucket
		}
		byID[p.UserID] = &p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(byID) == 0 {
		return []Profile{}, nil
	}
	present := make([]string, 0, len(byID))
	for id := range byID {
		present = append(present, id)
	}

	interestRows, err := r.db.Query(ctx, `
		SELECT ui.user_id::text, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label
	`, present)
	if err != nil {
		return nil, err
	}
	defer interestRows.Close()
	for interestRows.Next() {
		var uid string
		var in Interest
		if err := interestRows.Scan(&uid, &in.Slug, &in.Label); err != nil {
			return nil, err
		}
		byID[uid].Interests = append(byID[uid].Interests, in)
	}
	if err := interestRows.Err(); err != nil {
		return nil, err
	}
	interestRows.Close()

	photoRows, err := r.db.Query(ctx, `
		SELECT id::text, user_id::text, position FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, present)
	if err != nil {
		return nil, err
	}
	defer photoRows.Close()
	for photoRows.Next() {
		var ph Photo
		var uid string
		if err := photoRows.Scan(&ph.ID, &uid, &ph.Position); err != nil {
			return nil, err
		}
		ph.URL = "/photos/" + ph.ID + "/file"
		byID[uid].Photos = append(byID[uid].Photos, ph)
	}
	if err := photoRows.Err(); err != nil {
		return nil, err
	}

	out := make([]Profile, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, *p)
		}
	}
	return out, nil
}
