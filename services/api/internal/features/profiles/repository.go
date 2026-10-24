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
	ErrUnknownInterest  = errors.New("unknown interest")
	ErrBirthDateChanged = errors.New("birth date cannot be changed")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lon *float64) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosForUsers(ctx context.Context, userIDs []string) (map[string][]PhotoRef, error)
	// CardProfiles loads profiles of the given users that the viewer is allowed to see
	// (active, complete, not blocked in either direction, discoverable or matched, or the viewer itself).
	CardProfiles(ctx context.Context, viewerID string, userIDs []string) ([]storedProfile, error)
	HasLocationAndComplete(ctx context.Context, userID string) (hasLocation, complete bool, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const profileColumns = `p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city, p.latitude, p.longitude, p.discoverable, p.show_distance`

func scanProfile(row pgx.Row) (storedProfile, error) {
	var p storedProfile
	err := row.Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.Discoverable, &p.ShowDistance)
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (storedProfile, error) {
	p, err := scanProfile(r.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles p WHERE p.user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var existing time.Time
	err = tx.QueryRow(ctx, `SELECT birth_date FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&existing)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, discoverable, show_distance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.Discoverable, in.ShowDistance); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if !existing.Equal(in.BirthDate) {
			return ErrBirthDateChanged
		}
		if _, err := tx.Exec(ctx, `
			UPDATE profiles
			SET first_name = $2, gender = $3, bio = $4, city = $5, discoverable = $6, show_distance = $7, updated_at = NOW()
			WHERE user_id = $1
		`, userID, in.FirstName, in.Gender, in.Bio, in.City, in.Discoverable, in.ShowDistance); err != nil {
			return err
		}
	}

	if err := replaceInterests(ctx, tx, userID, in.InterestSlugs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func replaceInterests(ctx context.Context, tx pgx.Tx, userID string, slugs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(slugs) == 0 {
		return nil
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, slugs)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(slugs) {
		return ErrUnknownInterest
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in, age_min = EXCLUDED.age_min,
		    age_max = EXCLUDED.age_max, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon *float64) error {
	tag, err := r.db.Exec(ctx, `UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1`, userID, lat, lon)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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

func (r *PGRepository) InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosForUsers(ctx context.Context, userIDs []string) (map[string][]PhotoRef, error) {
	out := make(map[string][]PhotoRef, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1) ORDER BY user_id, position
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

func (r *PGRepository) CardProfiles(ctx context.Context, viewerID string, userIDs []string) ([]storedProfile, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+profileColumns+`
		FROM profiles p
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		WHERE p.user_id = ANY($2)
		  AND (
		    p.user_id = $1
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		           OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
		      )
		      AND EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = p.user_id)
		      AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		      AND (
		        p.discoverable
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.unmatched_at IS NULL
		            AND m.user_low = LEAST($1::uuid, p.user_id) AND m.user_high = GREATEST($1::uuid, p.user_id)
		        )
		      )
		    )
		  )
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) HasLocationAndComplete(ctx context.Context, userID string) (bool, bool, error) {
	var hasLoc, complete bool
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(p.latitude IS NOT NULL, FALSE),
		       EXISTS (SELECT 1 FROM preferences pr WHERE pr.user_id = p.user_id)
		       AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p WHERE p.user_id = $1
	`, userID).Scan(&hasLoc, &complete)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	return hasLoc, complete, err
}
