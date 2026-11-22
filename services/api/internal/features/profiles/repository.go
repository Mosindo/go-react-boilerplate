package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetOwn(ctx context.Context, userID string) (storedProfile, error)
	UpdateProfile(ctx context.Context, userID string, u profileUpdate) error
	SetInterests(ctx context.Context, userID string, slugs []string) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city string) error
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosFor(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]storedCard, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

// profileUpdate holds already-validated values (nil = unchanged).
type profileUpdate struct {
	FirstName    *string
	Birthdate    *time.Time
	Gender       *string
	Bio          *string
	JobTitle     *string
	Discoverable *bool
	ShowDistance *bool
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ensure creates the profile and preferences rows lazily.
func ensure(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT u.id, p.first_name, p.birthdate, p.gender,
		       COALESCE(p.bio, ''), COALESCE(p.job_title, ''), COALESCE(p.city, ''),
		       p.latitude, p.longitude,
		       COALESCE(p.discoverable, TRUE), COALESCE(p.show_distance, TRUE),
		       COALESCE(p.moderation_status, 'active'),
		       COALESCE(pr.interested_in, '{}'), COALESCE(pr.min_age, 18), COALESCE(pr.max_age, 99),
		       COALESCE(pr.max_distance_km, 50),
		       (SELECT COUNT(*) FROM photos ph WHERE ph.user_id = u.id)
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN preferences pr ON pr.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(
		&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender,
		&p.Bio, &p.JobTitle, &p.City,
		&p.Latitude, &p.Longitude,
		&p.Discoverable, &p.ShowDistance, &p.ModerationStatus,
		&p.Preferences.InterestedIn, &p.Preferences.MinAge, &p.Preferences.MaxAge,
		&p.Preferences.MaxDistanceKm, &p.PhotoCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrProfileNotFound
	}
	return p, err
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID string, u profileUpdate) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := ensure(ctx, tx, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE profiles SET
				first_name = COALESCE($2, first_name),
				birthdate = COALESCE($3, birthdate),
				gender = COALESCE($4, gender),
				bio = COALESCE($5, bio),
				job_title = COALESCE($6, job_title),
				discoverable = COALESCE($7, discoverable),
				show_distance = COALESCE($8, show_distance),
				updated_at = NOW()
			WHERE user_id = $1
		`, userID, u.FirstName, u.Birthdate, u.Gender, u.Bio, u.JobTitle, u.Discoverable, u.ShowDistance)
		return err
	})
}

func (r *PGRepository) SetInterests(ctx context.Context, userID string, slugs []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
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
		_, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, slugs)
		return err
	})
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := ensure(ctx, tx, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE profiles
			SET latitude = $2, longitude = $3, city = $4, location_updated_at = NOW(), updated_at = NOW()
			WHERE user_id = $1
		`, userID, lat, lng, city)
		return err
	})
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := ensure(ctx, tx, userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE preferences
			SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
			WHERE user_id = $1
		`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
		return err
	})
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

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label, i.category
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
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosFor(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error) {
	out := make(map[string][]storedPhoto, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, width, height
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
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

// Cards loads public rows for userIDs with distance relative to the viewer.
// Order follows the input slice.
func (r *PGRepository) Cards(ctx context.Context, viewerID string, userIDs []string) ([]storedCard, error) {
	if len(userIDs) == 0 {
		return []storedCard{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city,
		       CASE WHEN me.latitude IS NULL OR p.latitude IS NULL THEN NULL
		            ELSE geo_distance_km(me.latitude, me.longitude, p.latitude, p.longitude) END,
		       p.show_distance, u.is_demo
		FROM unnest($2::uuid[]) WITH ORDINALITY AS ids(id, ord)
		JOIN profiles p ON p.user_id = ids.id
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.first_name IS NOT NULL AND p.birthdate IS NOT NULL AND p.gender IS NOT NULL
		ORDER BY ids.ord
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedCard, error) {
		var c storedCard
		err := row.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &c.City,
			&c.DistanceKm, &c.ShowDistance, &c.IsDemo)
		return c, err
	})
}

// CanView decides whether viewer may open target's full profile: never when
// a block exists; always when matched or when target liked the viewer;
// otherwise only if target is publicly discoverable in the same workspace.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users tu ON tu.id = p.user_id
			JOIN users vu ON vu.id = $1
			WHERE p.user_id = $2
			  AND tu.organization_id = vu.organization_id
			  AND NOT `+SQLBlockedBetween("$1::uuid", "$2::uuid")+`
			  AND (
			    `+SQLMatched("$1::uuid", "$2::uuid")+`
			    OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like')
			    OR (p.discoverable AND `+SQLProfileComplete("p", "pr")+`)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
