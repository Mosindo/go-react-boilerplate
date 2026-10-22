package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterests = errors.New("unknown interest")
)

type Repository interface {
	GetSelf(ctx context.Context, userID string) (SelfProfile, error)
	Upsert(ctx context.Context, profile StoredProfile) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// Querier is satisfied by pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

const ageSQL = "EXTRACT(YEAR FROM age(CURRENT_DATE, p.birthdate))::int"

func (r *PGRepository) GetSelf(ctx context.Context, userID string) (SelfProfile, error) {
	var p SelfProfile
	var birthdate time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, `+ageSQL+`, p.gender, p.bio, p.city,
		       p.latitude, p.longitude, p.is_discoverable, p.show_distance, p.updated_at
		FROM profiles p
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &birthdate, &p.Age, &p.Gender, &p.Bio, &p.City,
		&p.Latitude, &p.Longitude, &p.IsDiscoverable, &p.ShowDistance, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SelfProfile{}, ErrNotFound
		}
		return SelfProfile{}, err
	}
	p.Birthdate = birthdate.Format("2006-01-02")

	interests, err := LoadInterests(ctx, r.pool, []string{userID})
	if err != nil {
		return SelfProfile{}, err
	}
	photos, err := LoadPhotos(ctx, r.pool, []string{userID})
	if err != nil {
		return SelfProfile{}, err
	}
	p.Interests = interests[userID]
	p.Photos = photos[userID]
	return p, nil
}

func (r *PGRepository) Upsert(ctx context.Context, p StoredProfile) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, city, latitude, longitude, is_discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birthdate = EXCLUDED.birthdate,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			is_discoverable = EXCLUDED.is_discoverable,
			show_distance = EXCLUDED.show_distance,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.City, p.Latitude, p.Longitude, p.IsDiscoverable, p.ShowDistance); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(p.InterestIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i.id FROM interests i WHERE i.id = ANY($2)
		`, p.UserID, p.InterestIDs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(p.InterestIDs) {
			return ErrUnknownInterests
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var prefs Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&prefs.InterestedIn, &prefs.MinAge, &prefs.MaxAge, &prefs.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, ErrNotFound
		}
		return Preferences{}, err
	}
	return prefs, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, prefs.InterestedIn, prefs.MinAge, prefs.MaxAge, prefs.MaxDistanceKm)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Interest, 0, 32)
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetPublic returns another user's profile as seen by viewerID. It answers
// ErrNotFound for blocked, hidden, incomplete or suspended profiles so that
// existence is never leaked.
func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var p PublicProfile
	var distance *float64
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, `+ageSQL+`, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND v.user_id IS NOT NULL
		            THEN `+geo.SQLDistanceKm("v.latitude", "v.longitude", "p.latitude", "p.longitude")+` END
		FROM profiles p
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = $2
		  AND NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (
		        p.user_id = $1
		        OR (p.is_discoverable AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))
		        OR EXISTS (SELECT 1 FROM matches m
		                   WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid))
		  )
	`, viewerID, targetID).Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &distance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicProfile{}, ErrNotFound
		}
		return PublicProfile{}, err
	}
	if distance != nil && viewerID != targetID {
		d := geo.DisplayDistance(*distance)
		p.DistanceKm = &d
	}

	interests, err := LoadInterests(ctx, r.pool, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	photos, err := LoadPhotos(ctx, r.pool, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	p.Interests = interests[targetID]
	p.Photos = photos[targetID]
	return p, nil
}

// LoadInterests fetches the interests of many users in one query.
func LoadInterests(ctx context.Context, q Querier, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	for _, id := range userIDs {
		out[id] = []Interest{}
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
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
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

// LoadPhotos fetches the photo references of many users in one query, ordered by position.
func LoadPhotos(ctx context.Context, q Querier, userIDs []string) (map[string][]PhotoRef, error) {
	out := make(map[string][]PhotoRef, len(userIDs))
	for _, id := range userIDs {
		out[id] = []PhotoRef{}
	}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id, id, position, width, height, left(md5(storage_key), 8)
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var ph PhotoRef
		if err := rows.Scan(&uid, &ph.ID, &ph.Position, &ph.Width, &ph.Height, &ph.Version); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], ph)
	}
	return out, rows.Err()
}
