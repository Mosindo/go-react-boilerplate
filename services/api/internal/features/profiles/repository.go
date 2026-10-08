package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoProfile = errors.New("profile not found")

type storedProfile struct {
	FirstName   string
	Birthdate   time.Time
	Gender      string
	Bio         string
	Occupation  string
	LookingFor  string
	City        string
	HasLocation bool
}

type rawCard struct {
	PublicProfile
	DistanceRaw *float64
	Birthdate   time.Time
}

type Repository interface {
	GetAccount(ctx context.Context, userID string) (AccountInfo, error)
	GetProfile(ctx context.Context, userID string) (*storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	GetInterests(ctx context.Context, userID string) ([]Interest, error)
	ListPhotos(ctx context.Context, userID string) ([]Photo, error)
	UpsertProfile(ctx context.Context, userID string, p validProfile) error
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lng float64, city string) error
	ClearLocation(ctx context.Context, userID string) error
	ListInterests(ctx context.Context) ([]Interest, error)
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetAccount(ctx context.Context, userID string) (AccountInfo, error) {
	var a AccountInfo
	err := r.db.QueryRow(ctx, `SELECT id, email, created_at FROM users WHERE id = $1`, userID).Scan(&a.ID, &a.Email, &a.CreatedAt)
	return a, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (*storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birthdate, gender, bio, occupation, looking_for, city, latitude IS NOT NULL
		FROM profiles WHERE user_id = $1`, userID,
	).Scan(&p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.Occupation, &p.LookingFor, &p.City, &p.HasLocation)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func defaultPreferences() Preferences {
	return Preferences{
		InterestedIn: []string{"man", "woman", "non_binary"}, MinAge: MinAge, MaxAge: MaxAge,
		IsVisible: true, ShowDistance: true,
	}
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p := defaultPreferences()
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, is_visible, show_distance
		FROM preferences WHERE user_id = $1`, userID,
	).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm, &p.IsVisible, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultPreferences(), nil
	}
	return p, err
}

func (r *PGRepository) GetInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label`, userID)
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

func photoOf(id string, position int) Photo {
	return Photo{ID: id, Position: position, URL: "/photos/" + id, ThumbURL: "/photos/" + id + "?size=thumb"}
}

func (r *PGRepository) ListPhotos(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position, created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var id string
		var pos int
		if err := rows.Scan(&id, &pos); err != nil {
			return nil, err
		}
		out = append(out, photoOf(id, pos))
	}
	return out, rows.Err()
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, p validProfile) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, occupation, looking_for, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name, birthdate = EXCLUDED.birthdate, gender = EXCLUDED.gender,
		  bio = EXCLUDED.bio, occupation = EXCLUDED.occupation, looking_for = EXCLUDED.looking_for,
		  city = CASE WHEN EXCLUDED.city = '' THEN profiles.city ELSE EXCLUDED.city END,
		  updated_at = NOW()`,
		userID, p.FirstName, p.Birthdate, p.Gender, p.Bio, p.Occupation, p.LookingFor, p.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(p.Interests) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)`, userID, p.Interests)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(p.Interests) {
			return ErrUnknownInterest
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, is_visible, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		  max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, is_visible = EXCLUDED.is_visible,
		  show_distance = EXCLUDED.show_distance, updated_at = NOW()`,
		userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.IsVisible, p.ShowDistance)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = CASE WHEN $4 = '' THEN city ELSE $4 END, updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoProfile
	}
	return nil
}

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE profiles SET latitude = NULL, longitude = NULL, updated_at = NOW() WHERE user_id = $1`, userID)
	return err
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

// Cards loads public profiles for ids in 3 queries total (profiles, photos, interests),
// regardless of how many ids are requested. Profiles that do not exist are absent from the map.
func (r *PGRepository) Cards(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error) {
	out := make(map[string]PublicProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.occupation, p.looking_for, p.city,
		  CASE WHEN COALESCE(pr.show_distance, TRUE) AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		       THEN distance_km(v.latitude, v.longitude, p.latitude, p.longitude) END,
		  (SELECT count(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
		    WHERE a.user_id = p.user_id AND b.user_id = $1::uuid)
		FROM profiles p
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		LEFT JOIN profiles v ON v.user_id = $1::uuid
		WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for rows.Next() {
		var c PublicProfile
		var bd time.Time
		var dist *float64
		if err := rows.Scan(&c.ID, &c.FirstName, &bd, &c.Gender, &c.Bio, &c.Occupation, &c.LookingFor, &c.City, &dist, &c.SharedInterests); err != nil {
			rows.Close()
			return nil, err
		}
		c.Age = AgeOn(bd, now)
		if dist != nil {
			d := DisplayDistance(*dist)
			c.DistanceKm = &d
		}
		c.Interests = []Interest{}
		c.Photos = []Photo{}
		out[c.ID] = c
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	prow, err := r.db.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position, created_at`, ids)
	if err != nil {
		return nil, err
	}
	for prow.Next() {
		var uid, pid string
		var pos int
		if err := prow.Scan(&uid, &pid, &pos); err != nil {
			prow.Close()
			return nil, err
		}
		if c, ok := out[uid]; ok {
			c.Photos = append(c.Photos, photoOf(pid, pos))
			out[uid] = c
		}
	}
	prow.Close()
	if err := prow.Err(); err != nil {
		return nil, err
	}

	irow, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, ids)
	if err != nil {
		return nil, err
	}
	defer irow.Close()
	for irow.Next() {
		var uid string
		var in Interest
		if err := irow.Scan(&uid, &in.Slug, &in.Label); err != nil {
			return nil, err
		}
		if c, ok := out[uid]; ok {
			c.Interests = append(c.Interests, in)
			out[uid] = c
		}
	}
	return out, irow.Err()
}

func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM users u JOIN profiles p ON p.user_id = u.id
		  WHERE u.id = $2::uuid AND u.status = 'active'
		    AND NOT EXISTS (SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1::uuid AND b.blocked_id = $2::uuid) OR (b.blocker_id = $2::uuid AND b.blocked_id = $1::uuid))
		    AND ( $1::uuid = $2::uuid
		       OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid))
		       OR (COALESCE((SELECT is_visible FROM preferences WHERE user_id = $2::uuid), TRUE)
		           AND EXISTS (SELECT 1 FROM photos WHERE user_id = $2::uuid)) )
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}
