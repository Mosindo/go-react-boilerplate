package profiles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotFound        = errors.New("profiles: not found")
	ErrRepoPhotoLimit      = errors.New("profiles: photo limit reached")
	ErrRepoUnknownInterest = errors.New("profiles: unknown interest")
	ErrRepoBadPhotoOrder   = errors.New("profiles: invalid photo order")
)

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error
	ListInterests(ctx context.Context) ([]Interest, error)

	AddPhoto(ctx context.Context, userID, id, key string, width, height int) (StoredPhoto, error)
	ReplacePhoto(ctx context.Context, userID, photoID, newKey string, width, height int) (oldKey string, err error)
	DeletePhoto(ctx context.Context, userID, photoID string) (key string, err error)
	ReorderPhotos(ctx context.Context, userID string, ids []string) error
	PhotoKeys(ctx context.Context, userID string) ([]string, error)
	// FindPhotoForViewer returns the photo only when viewerID owns it or no block
	// exists between the viewer and the owner.
	FindPhotoForViewer(ctx context.Context, photoID, viewerID string) (StoredPhoto, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const profileSelect = `
	SELECT p.user_id, p.first_name, p.birth_date, EXTRACT(YEAR FROM age(p.birth_date))::int,
	       p.gender, p.bio, p.city, (p.latitude IS NOT NULL), p.discoverable, p.show_distance, p.created_at,
	       COALESCE(pr.interested_in, ARRAY['man','woman','non_binary']::text[]),
	       COALESCE(pr.age_min, 18), COALESCE(pr.age_max, 99), COALESCE(pr.max_distance_km, 50)
	FROM profiles p
	LEFT JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1`

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	var birth time.Time
	err := r.db.QueryRow(ctx, profileSelect, userID).Scan(
		&p.UserID, &p.FirstName, &birth, &p.Age, &p.Gender, &p.Bio, &p.City, &p.HasLocation,
		&p.Discoverable, &p.ShowDistance, &p.CreatedAt,
		&p.Preferences.InterestedIn, &p.Preferences.AgeMin, &p.Preferences.AgeMax, &p.Preferences.MaxDistanceKM,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrRepoNotFound
		}
		return Profile{}, err
	}
	p.BirthDate = birth.Format("2006-01-02")

	interests, err := LoadInterests(ctx, r.db, []string{userID})
	if err != nil {
		return Profile{}, err
	}
	photos, err := LoadPhotos(ctx, r.db, []string{userID})
	if err != nil {
		return Profile{}, err
	}
	p.Interests = interests[userID]
	p.Photos = photos[userID]
	p.Complete = len(p.Photos) > 0
	return p, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in)
		VALUES ($1, ARRAY['man','woman','non_binary'])
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, in.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.Interests) {
			return ErrRepoUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, age_min = $3, age_max = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKM)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance *bool) error {
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
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
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

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// LoadInterests batch-loads interest slugs for many users in one query (no N+1).
func LoadInterests(ctx context.Context, q queryer, userIDs []string) (map[string][]string, error) {
	out := make(map[string][]string, len(userIDs))
	for _, id := range userIDs {
		out[id] = []string{}
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT ui.user_id, i.slug
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, slug string
		if err := rows.Scan(&uid, &slug); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], slug)
	}
	return out, rows.Err()
}

// LoadPhotos batch-loads ordered photos for many users in one query (no N+1).
func LoadPhotos(ctx context.Context, q queryer, userIDs []string) (map[string][]Photo, error) {
	out := make(map[string][]Photo, len(userIDs))
	for _, id := range userIDs {
		out[id] = []Photo{}
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id, id, position, width, height
		FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var p Photo
		if err := rows.Scan(&uid, &p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		p.URL = PhotoURL(p.ID)
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}

func PhotoURL(id string) string { return "/photos/" + id }

func (r *PGRepository) AddPhoto(ctx context.Context, userID, id, key string, width, height int) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Serialize concurrent uploads of the same user on the profile row.
	var locked string
	if err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPhoto{}, ErrRepoNotFound
		}
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrRepoPhotoLimit
	}
	photo := StoredPhoto{ID: id, UserID: userID, StorageKey: key, Position: count, Width: width, Height: height}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (id, user_id, storage_key, position, width, height) VALUES ($1, $2, $3, $4, $5, $6)
	`, id, userID, key, count, width, height); err != nil {
		return StoredPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredPhoto{}, err
	}
	return photo, nil
}

func (r *PGRepository) ReplacePhoto(ctx context.Context, userID, photoID, newKey string, width, height int) (string, error) {
	var oldKey string
	err := r.db.QueryRow(ctx, `
		UPDATE photos p SET storage_key = $3, width = $4, height = $5
		FROM photos old
		WHERE p.id = $2 AND p.user_id = $1 AND old.id = p.id
		RETURNING old.storage_key
	`, userID, photoID, newKey, width, height).Scan(&oldKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepoNotFound
		}
		return "", err
	}
	return oldKey, nil
}

func (r *PGRepository) DeletePhoto(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var key string
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepoNotFound
		}
		return "", err
	}
	// Close the gap so positions stay contiguous (0 is the main photo).
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) ReorderPhotos(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		return err
	}
	existing := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) != len(existing) {
		return ErrRepoBadPhotoOrder
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := existing[id]; !ok {
			return ErrRepoBadPhotoOrder
		}
		if _, dup := seen[id]; dup {
			return ErrRepoBadPhotoOrder
		}
		seen[id] = struct{}{}
	}
	// The unique (user_id, position) constraint is deferred, so rows can be rewritten in any order.
	for pos, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, pos); err != nil {
			return fmt.Errorf("reorder photo: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) PhotoKeys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *PGRepository) FindPhotoForViewer(ctx context.Context, photoID, viewerID string) (StoredPhoto, error) {
	var p StoredPhoto
	err := r.db.QueryRow(ctx, `
		SELECT ph.id, ph.user_id, ph.storage_key, ph.position, ph.width, ph.height
		FROM photos ph
		WHERE ph.id = $1
		  AND (ph.user_id = $2 OR NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $2 AND b.blocked_id = ph.user_id)
		       OR (b.blocker_id = ph.user_id AND b.blocked_id = $2)))
	`, photoID, viewerID).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPhoto{}, ErrRepoNotFound
		}
		return StoredPhoto{}, err
	}
	return p, nil
}
