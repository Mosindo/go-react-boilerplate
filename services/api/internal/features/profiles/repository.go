package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profiles: not found")

// storedPhoto carries storage ids; URLs are signed by the service.
type storedPhoto struct {
	ID       string
	UserID   string
	Position int
}

type ownRow struct {
	OwnProfile
	BirthDate time.Time
}

// publicRow is a profile row before photos/interests/distance are attached.
type publicRow struct {
	PublicProfile
	BirthDate    time.Time
	ShowDistance bool
	Lat, Lng     *float64
}

type Repository interface {
	GetOwn(ctx context.Context, userID string) (ownRow, error)
	Upsert(ctx context.Context, userID string, in ProfileInput, lat, lng *float64) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, p Preferences) error
	PublicRows(ctx context.Context, ids []string) ([]publicRow, error)
	PhotosFor(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	ViewerLocation(ctx context.Context, userID string) (lat, lng *float64, err error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (ownRow, error) {
	var o ownRow
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, u.birth_date, p.gender, p.bio, p.city, p.latitude, p.longitude,
		       p.occupation, p.height_cm, p.looking_for,
		       COALESCE((SELECT array_agg(i.slug ORDER BY i.slug) FROM user_interests ui JOIN interests i ON i.id = ui.interest_id WHERE ui.user_id = p.user_id), '{}')
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&o.UserID, &o.FirstName, &o.BirthDate, &o.Gender, &o.Bio, &o.City, &o.Latitude, &o.Longitude,
		&o.Occupation, &o.HeightCM, &o.LookingFor, &o.Interests)
	if errors.Is(err, pgx.ErrNoRows) {
		return ownRow{}, ErrNotFound
	}
	return o, err
}

// Upsert writes the profile, creates default preferences on first save and replaces interests, atomically.
func (r *PGRepository) Upsert(ctx context.Context, userID string, in ProfileInput, lat, lng *float64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, gender, bio, city, latitude, longitude, occupation, height_cm, looking_for)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			occupation = EXCLUDED.occupation,
			height_cm = EXCLUDED.height_cm,
			looking_for = EXCLUDED.looking_for,
			updated_at = NOW()
	`, userID, in.FirstName, in.Gender, in.Bio, in.City, lat, lng, in.Occupation, in.HeightCM, in.LookingFor)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.Interests) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, in.Interests); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
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

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	var minAge, maxAge int16
	var maxDist *int16
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &minAge, &maxAge, &maxDist, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrNotFound
	}
	if err != nil {
		return Preferences{}, err
	}
	p.MinAge, p.MaxAge = int(minAge), int(maxAge)
	if maxDist != nil {
		v := int(*maxDist)
		p.MaxDistanceKm = &v
	}
	return p, nil
}

func (r *PGRepository) UpdatePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences
		SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5,
		    discoverable = $6, show_distance = $7, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.Discoverable, p.ShowDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PublicRows loads profile rows for the given ids in one query. Callers must have authorised the ids.
func (r *PGRepository) PublicRows(ctx context.Context, ids []string) ([]publicRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, u.birth_date, p.gender, p.bio, p.city, p.occupation, p.height_cm,
		       p.looking_for, pr.show_distance, p.latitude, p.longitude
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = ANY($1::uuid[])
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []publicRow
	for rows.Next() {
		var row publicRow
		if err := rows.Scan(&row.UserID, &row.FirstName, &row.BirthDate, &row.Gender, &row.Bio, &row.City,
			&row.Occupation, &row.HeightCM, &row.LookingFor, &row.ShowDistance, &row.Lat, &row.Lng); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosFor(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var in Interest
		if err := rows.Scan(&uid, &in.Slug, &in.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], in)
	}
	return out, rows.Err()
}

func (r *PGRepository) ViewerLocation(ctx context.Context, userID string) (*float64, *float64, error) {
	var lat, lng *float64
	err := r.db.QueryRow(ctx, `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, userID).Scan(&lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return lat, lng, err
}

// CanView applies the visibility rules: active account, no block in either direction, and either a
// discoverable profile or an existing match with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.status = 'active'
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $2
			  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (pr.discoverable
			       OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
			       OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
