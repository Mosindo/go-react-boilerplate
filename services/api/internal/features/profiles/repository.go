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
	ListInterests(ctx context.Context) ([]Interest, error)
	Overview(ctx context.Context, userID string) (Overview, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput, birth time.Time) error
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, p Privacy) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64) error
	ClearLocation(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

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

func (r *PGRepository) Overview(ctx context.Context, userID string) (Overview, error) {
	var o Overview
	var p Profile
	var birth time.Time
	var hasLoc bool
	var priv Privacy
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, latitude IS NOT NULL,
		       is_discoverable, show_distance, updated_at
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &hasLoc, &priv.IsDiscoverable, &priv.ShowDistance, &p.UpdatedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return o, err
	}
	if err == nil {
		p.BirthDate = birth.Format("2006-01-02")
		p.HasLocation = hasLoc
		p.Interests = []string{}
		p.Photos = []PhotoRef{}
		rows, err := r.db.Query(ctx, `
			SELECT i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
			WHERE ui.user_id = $1 ORDER BY i.slug`, userID)
		if err != nil {
			return o, err
		}
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				rows.Close()
				return o, err
			}
			p.Interests = append(p.Interests, s)
		}
		rows.Close()
		prows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
		if err != nil {
			return o, err
		}
		for prows.Next() {
			var ph PhotoRef
			if err := prows.Scan(&ph.ID, &ph.Position); err != nil {
				prows.Close()
				return o, err
			}
			p.Photos = append(p.Photos, ph)
		}
		prows.Close()
		o.Profile = &p
		o.Privacy = &priv
	}

	var pref Preferences
	err = r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&pref.InterestedIn, &pref.MinAge, &pref.MaxAge, &pref.MaxDistanceKm)
	if err == nil {
		o.Preferences = &pref
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return o, err
	}
	return o, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput, birth time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city, updated_at = NOW()`,
		userID, in.FirstName, birth, in.Gender, in.Bio, in.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)`, userID, in.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.Interests) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age, max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()`,
		userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, p Privacy) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1`,
		userID, p.IsDiscoverable, p.ShowDistance)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.db.Exec(ctx, `UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1`, userID, lat, lng)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE profiles SET latitude = NULL, longitude = NULL, updated_at = NOW() WHERE user_id = $1`, userID)
	return err
}
