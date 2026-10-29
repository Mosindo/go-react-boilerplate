package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile, lat, lng *float64, interestIDs []int16) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
	GetGender(ctx context.Context, userID string) (string, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Querier is satisfied by *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	p := Profile{UserID: userID}
	err := r.pool.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, latitude IS NOT NULL, is_visible, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.HasLocation, &p.IsVisible, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}

	items := []*PublicProfile{{ID: userID}}
	if err := Hydrate(ctx, r.pool, items); err != nil {
		return Profile{}, err
	}
	p.Interests = items[0].Interests
	p.Photos = items[0].Photos
	return p, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile, lat, lng *float64, interestIDs []int16) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, is_visible, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = COALESCE(EXCLUDED.latitude, profiles.latitude),
			longitude = COALESCE(EXCLUDED.longitude, profiles.longitude),
			is_visible = EXCLUDED.is_visible,
			show_distance = EXCLUDED.show_distance,
			updated_at = NOW()
	`, p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City, lat, lng, p.IsVisible, p.ShowDistance)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(interestIDs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i.id FROM interests i WHERE i.id = ANY($2)
		`, p.UserID, interestIDs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(interestIDs) {
			return errUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

var errUnknownInterest = errors.New("unknown interest id")

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
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

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label)
		return i, err
	})
}

func (r *PGRepository) GetGender(ctx context.Context, userID string) (string, error) {
	var g string
	err := r.pool.QueryRow(ctx, `SELECT gender FROM profiles WHERE user_id = $1`, userID).Scan(&g)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return g, err
}

// GetPublicProfile returns another user's profile as seen by viewerID. Hidden
// profiles are only visible to matches; blocked users (in either direction) are
// reported as not found.
func (r *PGRepository) GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var (
		pp       PublicProfile
		birth    time.Time
		showDist bool
		distance *float64
	)
	pp.ID = targetID
	err := r.pool.QueryRow(ctx, `
		SELECT t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance,
		       CASE WHEN v.latitude IS NULL OR t.latitude IS NULL THEN NULL
		            ELSE haversine_km(v.latitude, v.longitude, t.latitude, t.longitude) END
		FROM profiles t
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = $2
		  AND NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = t.user_id)
		           OR (b.blocker_id = t.user_id AND b.blocked_id = $1))
		  AND (t.user_id = $1 OR t.is_visible OR EXISTS (
		        SELECT 1 FROM matches m
		        WHERE m.user_low_id = LEAST($1::uuid, t.user_id) AND m.user_high_id = GREATEST($1::uuid, t.user_id)))
	`, viewerID, targetID).Scan(&pp.FirstName, &birth, &pp.Gender, &pp.Bio, &pp.City, &showDist, &distance)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrNotFound
	}
	if err != nil {
		return PublicProfile{}, err
	}
	pp.Age = AgeAt(birth, time.Now())
	if showDist && distance != nil && viewerID != targetID {
		d := ApproxDistance(*distance)
		pp.DistanceKm = &d
	}
	items := []*PublicProfile{&pp}
	if err := Hydrate(ctx, r.pool, items); err != nil {
		return PublicProfile{}, err
	}
	return pp, nil
}

// Hydrate fills Photos and Interests for every item using two queries in total,
// whatever the number of items (no N+1).
func Hydrate(ctx context.Context, q Querier, items []*PublicProfile) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, len(items))
	byID := make(map[string]*PublicProfile, len(items))
	for i, it := range items {
		ids[i] = it.ID
		it.Photos = []PhotoRef{}
		it.Interests = []Interest{}
		byID[it.ID] = it
	}

	rows, err := q.Query(ctx, `
		SELECT user_id::text, id::text, position FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var uid string
		var ph PhotoRef
		if err := rows.Scan(&uid, &ph.ID, &ph.Position); err != nil {
			rows.Close()
			return err
		}
		ph.URL = PhotoURL(ph.ID)
		byID[uid].Photos = append(byID[uid].Photos, ph)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = q.Query(ctx, `
		SELECT ui.user_id::text, i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY ui.user_id, i.id
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var in Interest
		if err := rows.Scan(&uid, &in.ID, &in.Slug, &in.Label); err != nil {
			return err
		}
		byID[uid].Interests = append(byID[uid].Interests, in)
	}
	return rows.Err()
}
