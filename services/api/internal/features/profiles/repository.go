package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnknownInterest = errors.New("unknown interest")

type Repository interface {
	UpsertProfile(ctx context.Context, userID string, in ProfileInput, birth time.Time) error
	GetProfile(ctx context.Context, userID string) (Profile, error)
	GetSettings(ctx context.Context, userID string) (Settings, error)
	UpsertSettings(ctx context.Context, userID string, s Settings) error
	ListInterests(ctx context.Context) ([]Interest, error)
	// GetPublicProfile renvoie le profil vu par viewerID; ErrProfileNotFound si
	// inexistant, bloque (dans un sens ou dans l'autre) ou non decouvrable sans match.
	GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, float64, bool, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput, birth time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio, city = EXCLUDED.city, latitude = EXCLUDED.latitude,
		  longitude = EXCLUDED.longitude, updated_at = NOW()
	`, userID, in.FirstName, birth, in.Gender, in.Bio, in.City, in.Latitude, in.Longitude); err != nil {
		return err
	}

	d := DefaultSettings()
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (user_id) DO NOTHING
	`, userID, d.InterestedIn, d.MinAge, d.MaxAge, d.MaxDistanceKm); err != nil {
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
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	var p Profile
	var birth time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude, longitude, updated_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, err
	}
	p.BirthDate = birth.Format(birthDateLayout)
	p.Age = AgeAt(birth, time.Now())
	if p.Interests, err = r.interestsOf(ctx, userID); err != nil {
		return Profile{}, err
	}
	if p.PhotoIDs, err = r.photoIDs(ctx, userID); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (r *PGRepository) interestsOf(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label
	`, userID)
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

func (r *PGRepository) photoIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetSettings(ctx context.Context, userID string) (Settings, error) {
	var s Settings
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&s.InterestedIn, &s.MinAge, &s.MaxAge, &s.MaxDistanceKm, &s.Discoverable, &s.ShowDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Settings{}, ErrProfileRequired
		}
		return Settings{}, err
	}
	return s, nil
}

func (r *PGRepository) UpsertSettings(ctx context.Context, userID string, s Settings) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE preferences SET interested_in=$2, min_age=$3, max_age=$4, max_distance_km=$5,
		  discoverable=$6, show_distance=$7, updated_at=NOW()
		WHERE user_id = $1
	`, userID, s.InterestedIn, s.MinAge, s.MaxAge, s.MaxDistanceKm, s.Discoverable, s.ShowDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileRequired
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
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

func (r *PGRepository) GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, float64, bool, error) {
	var p PublicProfile
	var birth time.Time
	var dist *float64
	var showDistance bool
	err := r.pool.QueryRow(ctx, `
		SELECT t.user_id, t.first_name, t.birth_date, t.gender, t.bio, t.city,
		       CASE WHEN v.user_id IS NULL THEN NULL
		            ELSE geo_distance_km(v.latitude, v.longitude, t.latitude, t.longitude) END,
		       tp.show_distance
		FROM profiles t
		JOIN preferences tp ON tp.user_id = t.user_id
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE t.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND ($1 = $2 OR tp.discoverable OR EXISTS (
		        SELECT 1 FROM matches m
		        WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
	`, viewerID, targetID).Scan(&p.UserID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &dist, &showDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublicProfile{}, 0, false, ErrProfileNotFound
		}
		return PublicProfile{}, 0, false, err
	}
	p.Age = AgeAt(birth, time.Now())
	if p.Interests, err = r.interestsOf(ctx, targetID); err != nil {
		return PublicProfile{}, 0, false, err
	}
	if p.PhotoIDs, err = r.photoIDs(ctx, targetID); err != nil {
		return PublicProfile{}, 0, false, err
	}
	d := 0.0
	if dist != nil {
		d = *dist
	}
	return p, d, dist != nil && showDistance, nil
}
