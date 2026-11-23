package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

type Repository interface {
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)
	ListUserPhotos(ctx context.Context, userID string) ([]PhotoRow, error)
	UpdateProfile(ctx context.Context, userID string, update ProfileUpdate) error
	UpdatePreferences(ctx context.Context, userID string, prefs Preferences) error
	UpdateSettings(ctx context.Context, userID string, settings Settings) error
	UpdateLocation(ctx context.Context, userID string, lat, lon *float64, city *string) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]CardRow, []PhotoRow, []InterestRow, error)
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
		SELECT user_id, first_name, birthdate, gender, bio, city, latitude, longitude, is_discoverable, hide_distance
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.IsDiscoverable, &p.HideDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrProfileNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km
		FROM preferences
		WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrProfileNotFound
	}
	if p.InterestedIn == nil {
		p.InterestedIn = []string{}
	}
	return p, err
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1
		ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.Slug, &i.Label)
		return i, err
	})
}

func (r *PGRepository) ListUserPhotos(ctx context.Context, userID string) ([]PhotoRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPhotoRow)
}

func scanPhotoRow(row pgx.CollectableRow) (PhotoRow, error) {
	var p PhotoRow
	err := row.Scan(&p.ID, &p.UserID, &p.Position)
	return p, err
}

func (r *PGRepository) UpdateProfile(ctx context.Context, userID string, u ProfileUpdate) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE profiles SET
			first_name = COALESCE($2, first_name),
			birthdate  = COALESCE($3, birthdate),
			gender     = COALESCE($4, gender),
			bio        = COALESCE($5, bio),
			city       = CASE WHEN $6::text IS NULL THEN city ELSE NULLIF($6::text, '') END,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, u.FirstName, u.Birthdate, u.Gender, u.Bio, u.City)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}

	if u.Interests != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if len(*u.Interests) > 0 {
			tag, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, id FROM interests WHERE slug = ANY($2)
			`, userID, *u.Interests)
			if err != nil {
				return err
			}
			if int(tag.RowsAffected()) != len(*u.Interests) {
				return ErrUnknownInterest
			}
		}
	}
	return tx.Commit(ctx)
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

func (r *PGRepository) UpdateSettings(ctx context.Context, userID string, s Settings) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, hide_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, s.IsDiscoverable, s.HideDistance)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lon *float64, city *string) error {
	var updatedAt *time.Time
	if lat != nil {
		now := time.Now()
		updatedAt = &now
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET
			latitude = $2,
			longitude = $3,
			location_updated_at = $4,
			city = CASE WHEN $5::text IS NULL THEN city ELSE NULLIF($5::text, '') END,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lon, updatedAt, city)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.Slug, &i.Label)
		return i, err
	})
}

// CanView decides whether viewer may see target's profile: never across a
// block or for suspended accounts; otherwise when matched, or when the target
// is discoverable with a complete public profile.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.suspended_at IS NULL
			  AND p.first_name IS NOT NULL AND p.birthdate IS NOT NULL AND p.gender IS NOT NULL
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				p.is_discoverable
				OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// LoadCards batch-loads public profile data for several users in three
// queries regardless of how many users are requested (no N+1).
func (r *PGRepository) LoadCards(ctx context.Context, viewerID string, userIDs []string) ([]CardRow, []PhotoRow, []InterestRow, error) {
	if len(userIDs) == 0 {
		return nil, nil, nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, COALESCE(p.city, ''), p.hide_distance,
		       CASE WHEN v.latitude IS NOT NULL AND p.latitude IS NOT NULL
		            THEN km_between(v.latitude, v.longitude, p.latitude, p.longitude)
		       END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
		  AND p.first_name IS NOT NULL AND p.birthdate IS NOT NULL AND p.gender IS NOT NULL
	`, viewerID, userIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	cards, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (CardRow, error) {
		var c CardRow
		err := row.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.City, &c.HideDistance, &c.DistanceKm)
		return c, err
	})
	if err != nil {
		return nil, nil, nil, err
	}

	photoRows, err := r.db.Query(ctx, `
		SELECT id, user_id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	photos, err := pgx.CollectRows(photoRows, scanPhotoRow)
	if err != nil {
		return nil, nil, nil, err
	}

	interestRows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	interests, err := pgx.CollectRows(interestRows, func(row pgx.CollectableRow) (InterestRow, error) {
		var i InterestRow
		err := row.Scan(&i.UserID, &i.Slug, &i.Label)
		return i, err
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return cards, photos, interests, nil
}
