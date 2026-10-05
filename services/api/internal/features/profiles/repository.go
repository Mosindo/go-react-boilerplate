package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/signing"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profile not found")
	ErrUnknownInterest = errors.New("unknown interest")
)

type StoredProfile struct {
	UserID         string
	FirstName      string
	BirthDate      time.Time
	Age            int
	Gender         string
	Bio            string
	Occupation     string
	City           string
	HasLocation    bool
	IsDiscoverable bool
	ShowDistance   bool
	CreatedAt      time.Time
}

type ProfileInput struct {
	FirstName  string
	BirthDate  time.Time
	Gender     string
	Bio        string
	Occupation string
	Interests  []string
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdateLocation(ctx context.Context, userID string, lat, lng float64, city string) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CountPhotos(ctx context.Context, userID string) (int, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	db     *pgxpool.Pool
	signer *signing.Signer
}

func NewPGRepository(db *pgxpool.Pool, signer *signing.Signer) *PGRepository {
	return &PGRepository{db: db, signer: signer}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, date_part('year', age(CURRENT_DATE, birth_date))::int,
		       gender, bio, occupation, city, latitude IS NOT NULL, is_discoverable, show_distance, created_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Age, &p.Gender, &p.Bio, &p.Occupation, &p.City,
		&p.HasLocation, &p.IsDiscoverable, &p.ShowDistance, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrNotFound
	}
	return p, err
}

// UpsertProfile writes the profile and replaces its interests atomically.
func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// birth_date is deliberately not updated on conflict: the age gate cannot be changed afterwards.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, occupation)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name, gender = EXCLUDED.gender, bio = EXCLUDED.bio,
		    occupation = EXCLUDED.occupation, updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.Occupation); err != nil {
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

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
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
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		    max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.db.Exec(ctx, `
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

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, discoverable, showDistance)
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

func (r *PGRepository) CountPhotos(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

// Cards loads the public cards of many users with three queries (profile, interests, photos), whatever the
// number of users, so list endpoints never run N+1 queries. Distance is relative to the viewer and rounded.
func (r *PGRepository) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	cards := make(map[string]Card, len(userIDs))
	if len(userIDs) == 0 {
		return cards, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, date_part('year', age(CURRENT_DATE, p.birth_date))::int, p.gender, p.bio,
		       p.occupation, p.city, p.show_distance, p.user_id = $1,
		       CASE WHEN me.latitude IS NOT NULL AND p.latitude IS NOT NULL
		            THEN haversine_km(me.latitude, me.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2)
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Card
		var showDistance, isSelf bool
		var km *float64
		if err := rows.Scan(&c.ID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.Occupation, &c.City, &showDistance, &isSelf, &km); err != nil {
			rows.Close()
			return nil, err
		}
		if !showDistance && !isSelf {
			c.City = ""
			km = nil
		}
		if km != nil && !isSelf {
			d := RoundDistanceKm(*km)
			c.DistanceKm = &d
		}
		c.Interests = []Interest{}
		c.Photos = []Photo{}
		cards[c.ID] = c
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.attachInterests(ctx, cards, userIDs); err != nil {
		return nil, err
	}
	if err := r.attachPhotos(ctx, cards, userIDs); err != nil {
		return nil, err
	}
	return cards, nil
}

func (r *PGRepository) attachInterests(ctx context.Context, cards map[string]Card, userIDs []string) error {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			return err
		}
		if c, ok := cards[uid]; ok {
			c.Interests = append(c.Interests, i)
			cards[uid] = c
		}
	}
	return rows.Err()
}

func (r *PGRepository) attachPhotos(ctx context.Context, cards map[string]Card, userIDs []string) error {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var ph Photo
		if err := rows.Scan(&uid, &ph.ID, &ph.Position); err != nil {
			return err
		}
		ph.URL = r.signer.PhotoPath(ph.ID)
		if c, ok := cards[uid]; ok {
			c.Photos = append(c.Photos, ph)
			cards[uid] = c
		}
	}
	return rows.Err()
}

// CanView reports whether viewer may open target's profile: no block in either direction, and the target is
// either discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND NOT EXISTS (
			      SELECT 1 FROM blocks b
			      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (p.is_discoverable OR EXISTS (
			      SELECT 1 FROM matches m
			      WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
