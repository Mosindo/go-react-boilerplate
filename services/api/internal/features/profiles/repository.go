package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound        = errors.New("profile not found")
	ErrRepositoryUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetOwn(ctx context.Context, userID string) (StoredProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput, interestSlugs []string) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lon float64, city *string) error
	SetPrivacy(ctx context.Context, userID string, isVisible, showDistance *bool) error
	// Cards loads public profiles in a fixed number of queries (no N+1).
	// Missing/hidden profiles are simply absent from the result.
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]*CardData, error)
	InterestsOf(ctx context.Context, userID string) ([]Interest, error)
	PhotoIDs(ctx context.Context, userIDs []string) (map[string][]string, error)
	CountPhotos(ctx context.Context, userID string) (int, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

// CardData is a card before photo ids are turned into signed URLs.
type CardData struct {
	Profile   StoredProfile
	Interests []Interest
	PhotoIDs  []string
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

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

const profileColumns = `p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.occupation, p.city,
	(p.latitude IS NOT NULL), p.is_visible, p.show_distance, date_part('year', age(p.birth_date))::int`

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.db.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles p WHERE p.user_id = $1`, userID).Scan(
		&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.Occupation, &p.City,
		&p.HasLocation, &p.IsVisible, &p.ShowDistance, &p.Age)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput, interestSlugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, occupation, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			occupation = EXCLUDED.occupation,
			city = EXCLUDED.city,
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.Occupation, in.City); err != nil {
		return err
	}

	var found int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE slug = ANY($1)`, interestSlugs).Scan(&found); err != nil {
		return err
	}
	if found != len(interestSlugs) {
		return ErrRepositoryUnknownInterest
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2)
	`, userID, interestSlugs); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon float64, city *string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, location_updated_at = NOW(),
		    city = COALESCE($4, city), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lon, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) SetPrivacy(ctx context.Context, userID string, isVisible, showDistance *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles
		SET is_visible = COALESCE($2, is_visible),
		    show_distance = COALESCE($3, show_distance),
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, isVisible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]*CardData, error) {
	out := make(map[string]*CardData, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+profileColumns+`,
		  CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		       THEN haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2)
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p StoredProfile
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.Occupation, &p.City,
			&p.HasLocation, &p.IsVisible, &p.ShowDistance, &p.Age, &p.DistanceKm); err != nil {
			rows.Close()
			return nil, err
		}
		out[p.UserID] = &CardData{Profile: p, Interests: []Interest{}, PhotoIDs: []string{}}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	irows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1)
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	for irows.Next() {
		var uid string
		var i Interest
		if err := irows.Scan(&uid, &i.Slug, &i.Label); err != nil {
			irows.Close()
			return nil, err
		}
		if c, ok := out[uid]; ok {
			c.Interests = append(c.Interests, i)
		}
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return nil, err
	}

	photos, err := r.PhotoIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for uid, ids := range photos {
		if c, ok := out[uid]; ok {
			c.PhotoIDs = ids
		}
	}
	return out, nil
}

func (r *PGRepository) InterestsOf(ctx context.Context, userID string) ([]Interest, error) {
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

func (r *PGRepository) PhotoIDs(ctx context.Context, userIDs []string) (map[string][]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id FROM photos WHERE user_id = ANY($1) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var uid, id string
		if err := rows.Scan(&uid, &id); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], id)
	}
	return out, rows.Err()
}

func (r *PGRepository) CountPhotos(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

// CanView: the target is enabled, nobody blocked the other, and the target is
// either discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2::uuid AND u.disabled_at IS NULL
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = $1::uuid AND b.blocked_id = $2::uuid)
			       OR (b.blocker_id = $2::uuid AND b.blocked_id = $1::uuid))
			  AND (p.is_visible OR EXISTS (
			    SELECT 1 FROM matches m
			    WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}
