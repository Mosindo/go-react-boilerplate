package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterests = errors.New("unknown interests")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, p Profile) error
	UpdatePrivacy(ctx context.Context, userID string, showAge, showDistance, discoverable bool) error
	SetLocation(ctx context.Context, userID string, lat, lng float64, city string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterestCatalog(ctx context.Context) ([]Interest, error)
	GetInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	SetInterests(ctx context.Context, userID string, ids []int16) error
	GetPhotos(ctx context.Context, userIDs []string) (map[string][]PhotoRef, error)
	GetViewable(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birthdate, gender, bio, city,
		       latitude IS NOT NULL, show_age, show_distance, discoverable
		FROM profiles
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.City,
		&p.HasLocation, &p.ShowAge, &p.ShowDistance, &p.Discoverable)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, p Profile) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name,
		    birthdate = EXCLUDED.birthdate,
		    gender = EXCLUDED.gender,
		    bio = EXCLUDED.bio,
		    updated_at = NOW()
	`, p.UserID, p.FirstName, p.Birthdate, p.Gender, p.Bio)
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, showAge, showDistance, discoverable bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET show_age = $2, show_distance = $3, discoverable = $4, updated_at = NOW()
		WHERE user_id = $1
	`, userID, showAge, showDistance, discoverable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = $4, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km
		FROM preferences WHERE user_id = $1
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
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in,
		    min_age = EXCLUDED.min_age,
		    max_age = EXCLUDED.max_age,
		    max_distance_km = EXCLUDED.max_distance_km,
		    updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) ListInterestCatalog(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Interest
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// GetInterests loads interests for many users in one query (no N+1).
func (r *PGRepository) GetInterests(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
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

func (r *PGRepository) SetInterests(ctx context.Context, userID string, ids []int16) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(ids) > 0 {
		var found int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM interests WHERE id = ANY($1::smallint[])`, ids).Scan(&found); err != nil {
			return err
		}
		if found != len(ids) {
			return ErrUnknownInterests
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

// GetPhotos loads photo references for many users in one query, ordered by position.
func (r *PGRepository) GetPhotos(ctx context.Context, userIDs []string) (map[string][]PhotoRef, error) {
	out := make(map[string][]PhotoRef, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position
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
		var p PhotoRef
		if err := rows.Scan(&uid, &p.ID, &p.Position); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}

// viewableSQL is the single visibility rule for profiles and photos: the target
// must be an active, non-blocked user who is either discoverable with at least
// one photo, matched with the viewer, or has liked the viewer. The viewer is
// $1 and the target $2.
const viewableSQL = `
	JOIN users u ON u.id = p.user_id AND u.status = 'active'
	WHERE p.user_id = $2
	  AND NOT EXISTS (
	    SELECT 1 FROM blocks b
	    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
	  )
	  AND (
	    (p.discoverable AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))
	    OR EXISTS (
	      SELECT 1 FROM matches m
	      WHERE m.ended_at IS NULL AND m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
	    )
	    OR EXISTS (
	      SELECT 1 FROM swipes s WHERE s.from_user_id = $2 AND s.to_user_id = $1 AND s.action = 'like'
	    )
	  )`

func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	if viewerID == targetID {
		return true, nil
	}
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles p `+viewableSQL+`)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// GetViewable returns the public view of another user's profile, or ErrNotFound
// when the viewer is not allowed to see it.
func (r *PGRepository) GetViewable(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var (
		pp           PublicProfile
		birthdate    time.Time
		showAge      bool
		showDistance bool
		distance     *float64
	)
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.city, p.show_age, p.show_distance,
		       CASE WHEN p.latitude IS NOT NULL AND me.latitude IS NOT NULL
		            THEN haversine_km(me.latitude, me.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		`+viewableSQL, viewerID, targetID).Scan(
		&pp.UserID, &pp.FirstName, &birthdate, &pp.Gender, &pp.Bio, &pp.City, &showAge, &showDistance, &distance)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrNotFound
	}
	if err != nil {
		return PublicProfile{}, err
	}
	if showAge {
		age := AgeOn(birthdate, time.Now())
		pp.Age = &age
	}
	if showDistance && distance != nil {
		km := ApproxDistanceKm(*distance)
		pp.DistanceKm = &km
	} else {
		pp.City = ""
	}
	return pp, nil
}
