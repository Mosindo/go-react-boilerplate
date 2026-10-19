package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	Upsert(ctx context.Context, userID string, in UpsertInput) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CountInterests(ctx context.Context, ids []int16) (int, error)
	GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error)

	ListPhotos(ctx context.Context, userID string) ([]StoredPhoto, error)
	GetPhotoForViewer(ctx context.Context, viewerID, photoID string) (StoredPhoto, error)
	AddPhoto(ctx context.Context, userID, storageKey string, position *int, width, height, size int) (added StoredPhoto, replaced *StoredPhoto, err error)
	DeletePhoto(ctx context.Context, userID, photoID string) (StoredPhoto, error)
	ReorderPhotos(ctx context.Context, userID string, orderedIDs []string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	var out OwnProfile
	var birth time.Time
	var hasLoc bool
	p, err := ScanCard(r.pool.QueryRow(ctx, `
		SELECT `+CardColumns+`, p.birth_date, p.gender, p.is_visible, p.lat IS NOT NULL
		FROM profiles p WHERE p.user_id = $1`, userID), &birth, &out.Gender, &out.IsVisible, &hasLoc)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrProfileNotFound
		}
		return OwnProfile{}, err
	}
	out.PublicProfile = p
	out.BirthDate = birth.Format("2006-01-02")
	out.HasLocation = hasLoc
	out.Preferences, err = r.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	return out, nil
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, in UpsertInput) error {
	var lat, lng any
	if in.Location != nil {
		lat, lng = in.Location.Lat, in.Location.Lng
	}
	visible := true
	if in.IsVisible != nil {
		visible = *in.IsVisible
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Location is only overwritten when the client sends it; sending the
	// profile without a location keeps the previous (coarse) one.
	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, lat, lng, is_visible)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			lat = COALESCE(EXCLUDED.lat, profiles.lat),
			lng = COALESCE(EXCLUDED.lng, profiles.lng),
			is_visible = $9,
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, lat, lng, visible)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, UNNEST($2::smallint[])
		`, userID, in.Interests); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, show_distance
		FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm, &p.ShowDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			show_distance = EXCLUDED.show_distance,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.ShowDistance)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
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

func (r *PGRepository) CountInterests(ctx context.Context, ids []int16) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM interests WHERE id = ANY($1)`, ids).Scan(&n)
	return n, err
}

// GetPublic returns another user's profile. Blocked pairs (either direction),
// hidden profiles and missing profiles are all reported as not found so the
// endpoint cannot be used to probe block state.
func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var dist *float64
	var show bool
	p, err := ScanCard(r.pool.QueryRow(ctx, `
		SELECT `+CardColumns+`,
			CASE WHEN v.lat IS NULL OR p.lat IS NULL THEN NULL ELSE
				2 * 6371 * asin(sqrt(least(1.0,
					power(sin(radians(p.lat - v.lat) / 2), 2) +
					cos(radians(v.lat)) * cos(radians(p.lat)) * power(sin(radians(p.lng - v.lng) / 2), 2)))) END,
			COALESCE(pr.show_distance, TRUE)
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $2
		  AND ($1 = $2 OR (p.is_visible
		       AND NOT EXISTS (SELECT 1 FROM blocks b
		            WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))))
	`, viewerID, targetID), &dist, &show)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return PublicProfile{}, ErrProfileNotFound
		}
		return PublicProfile{}, err
	}
	if dist != nil && show && viewerID != targetID {
		d := ApproxDistanceKm(*dist)
		p.DistanceKm = &d
	}
	return p, nil
}

func (r *PGRepository) ListPhotos(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, storage_key, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredPhoto{}
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPhotoForViewer enforces photo access: owners always, other users only when
// neither side blocked the other.
func (r *PGRepository) GetPhotoForViewer(ctx context.Context, viewerID, photoID string) (StoredPhoto, error) {
	var p StoredPhoto
	err := r.pool.QueryRow(ctx, `
		SELECT ph.id, ph.user_id, ph.storage_key, ph.position
		FROM photos ph
		WHERE ph.id = $2
		  AND (ph.user_id = $1 OR NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id)
		           OR (b.blocker_id = ph.user_id AND b.blocked_id = $1)))
	`, viewerID, photoID).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return StoredPhoto{}, ErrPhotoNotFound
		}
		return StoredPhoto{}, err
	}
	return p, nil
}

// AddPhoto inserts a photo at the given position (replacing the occupant) or
// appends it when position is nil. Serialised per user with an advisory lock so
// concurrent uploads cannot exceed the limit or collide on positions.
func (r *PGRepository) AddPhoto(ctx context.Context, userID, storageKey string, position *int, width, height, size int) (StoredPhoto, *StoredPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID); err != nil {
		return StoredPhoto{}, nil, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, nil, err
	}

	var replaced *StoredPhoto
	target := count
	if position != nil {
		if *position < 0 || *position >= MaxPhotos || *position > count {
			return StoredPhoto{}, nil, ErrInvalidInput
		}
		target = *position
		var old StoredPhoto
		err := tx.QueryRow(ctx, `
			DELETE FROM photos WHERE user_id = $1 AND position = $2
			RETURNING id, user_id, storage_key, position`, userID, target).
			Scan(&old.ID, &old.UserID, &old.StorageKey, &old.Position)
		if err == nil {
			replaced = &old
			count--
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return StoredPhoto{}, nil, err
		}
	}
	if replaced == nil && count >= MaxPhotos {
		return StoredPhoto{}, nil, ErrPhotoLimit
	}

	var added StoredPhoto
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, storage_key, position`,
		userID, storageKey, target, width, height, size).
		Scan(&added.ID, &added.UserID, &added.StorageKey, &added.Position)
	if err != nil {
		return StoredPhoto{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredPhoto{}, nil, err
	}
	return added, replaced, nil
}

func (r *PGRepository) DeletePhoto(ctx context.Context, userID, photoID string) (StoredPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID); err != nil {
		return StoredPhoto{}, err
	}
	var p StoredPhoto
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, storage_key, position`, photoID, userID).
		Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return StoredPhoto{}, ErrPhotoNotFound
		}
		return StoredPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_id_position_key DEFERRED`); err != nil {
		return StoredPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = sub.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) sub
		WHERE photos.id = sub.id`, userID); err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) ReorderPhotos(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID); err != nil {
		return err
	}
	var existing int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, orderedIDs).Scan(&existing); err != nil {
		if isInvalidUUID(err) {
			return ErrInvalidInput
		}
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if existing != len(orderedIDs) || total != len(orderedIDs) {
		return ErrInvalidInput
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_id_position_key DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = o.ord - 1
		FROM UNNEST($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE photos.id = o.id AND photos.user_id = $1`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
