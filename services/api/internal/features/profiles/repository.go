package profiles

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterests = errors.New("unknown interests")
)

type Repository interface {
	GetOwn(ctx context.Context, userID string) (*OwnProfile, error)
	Save(ctx context.Context, in ProfileInput) error
	SetLocation(ctx context.Context, userID string, lat, lng float64) error
	ClearLocation(ctx context.Context, userID string) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	SavePreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetVisible(ctx context.Context, viewerID, targetID string) (*PublicProfile, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (*OwnProfile, error) {
	var p OwnProfile
	var birth time.Time
	var hasLoc bool
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.display_name, p.birth_date, p.gender, p.bio, p.city,
		       p.show_distance, p.discoverable, p.latitude IS NOT NULL, p.created_at, p.updated_at,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&p.UserID, &p.DisplayName, &birth, &p.Gender, &p.Bio, &p.City,
		&p.ShowDistance, &p.Discoverable, &hasLoc, &p.CreatedAt, &p.UpdatedAt,
		&p.Preferences.InterestedIn, &p.Preferences.MinAge, &p.Preferences.MaxAge, &p.Preferences.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.BirthDate = birth.Format("2006-01-02")
	p.Age = AgeAt(birth, time.Now().UTC())
	p.HasLocation = hasLoc

	interests, err := r.interestsFor(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	photos, err := r.photosFor(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	p.Interests = orEmptyInterests(interests[userID])
	p.Photos = orEmptyPhotos(photos[userID])
	p.Completed = len(p.Photos) > 0
	return &p, nil
}

func (r *PGRepository) Save(ctx context.Context, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// birth_date is deliberately not updated: the age gate cannot be bypassed by editing it later.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, display_name, birth_date, gender, bio, city, show_distance, discoverable)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			show_distance = EXCLUDED.show_distance,
			discoverable = EXCLUDED.discoverable,
			updated_at = NOW()
	`, in.UserID, in.DisplayName, in.BirthDate, in.Gender, in.Bio, in.City, in.ShowDistance, in.Discoverable); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in) VALUES ($1, ARRAY['man','woman','nonbinary'])
		ON CONFLICT (user_id) DO NOTHING
	`, in.UserID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, in.UserID); err != nil {
		return err
	}
	if len(in.InterestSlug) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, in.UserID, in.InterestSlug)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.InterestSlug) {
			return ErrUnknownInterests
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) ClearLocation(ctx context.Context, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = NULL, longitude = NULL, location_updated_at = NULL, updated_at = NOW()
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
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

// publicColumns builds the viewer-relative projection shared by GetVisible and discovery.
// $1 is always the viewer id. distance is the raw haversine distance in km (NULL when unknown or hidden).
const publicColumns = `
	SELECT t.user_id, t.display_name, t.birth_date, t.gender, t.bio, t.city,
	       CASE WHEN me.latitude IS NULL OR t.latitude IS NULL OR NOT t.show_distance OR NOT me.show_distance
	            THEN NULL ELSE ` + DistanceSQL + ` END AS distance_km,
	       (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
	         WHERE a.user_id = me.user_id AND b.user_id = t.user_id) AS common`

// DistanceSQL is the haversine formula between profiles aliased me and t, in kilometres.
const DistanceSQL = `(6371 * 2 * asin(sqrt(least(1, power(sin(radians(t.latitude - me.latitude) / 2), 2)
	+ cos(radians(me.latitude)) * cos(radians(t.latitude)) * power(sin(radians(t.longitude - me.longitude) / 2), 2)))))`

// VisibilitySQL restricts target profile t to what viewer me may see: never blocked either way,
// and either matched or publicly discoverable. Callers add their own conditions.
const VisibilitySQL = `
	t.user_id <> me.user_id
	AND EXISTS (SELECT 1 FROM users u WHERE u.id = t.user_id AND u.status = 'active')
	AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = t.user_id)
	                                          OR (b.blocker_id = t.user_id AND b.blocked_id = me.user_id))
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
	AND (t.discoverable OR EXISTS (
	      SELECT 1 FROM matches m WHERE m.unmatched_at IS NULL
	        AND m.user_a = LEAST(me.user_id, t.user_id) AND m.user_b = GREATEST(me.user_id, t.user_id)))`

func (r *PGRepository) GetVisible(ctx context.Context, viewerID, targetID string) (*PublicProfile, error) {
	profiles, err := r.PublicByQuery(ctx, publicColumns+`
		FROM profiles me, profiles t
		WHERE me.user_id = $1 AND t.user_id = $2 AND `+VisibilitySQL, viewerID, targetID)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, ErrNotFound
	}
	return &profiles[0], nil
}

// PublicByQuery runs a query shaped like PublicColumns and hydrates interests and photos in two
// extra batched queries (no N+1). Result order follows the query's own ORDER BY.
func (r *PGRepository) PublicByQuery(ctx context.Context, query string, args ...any) ([]PublicProfile, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	var out []PublicProfile
	var ids []string
	for rows.Next() {
		var p PublicProfile
		var birth time.Time
		var dist *float64
		if err := rows.Scan(&p.UserID, &p.DisplayName, &birth, &p.Gender, &p.Bio, &p.City, &dist, &p.CommonInterests); err != nil {
			return nil, err
		}
		p.Age = AgeAt(birth, now)
		p.DistanceKm = ApproxDistance(dist)
		out = append(out, p)
		ids = append(ids, p.UserID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []PublicProfile{}, nil
	}
	interests, err := r.interestsFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := r.photosFor(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Interests = orEmptyInterests(interests[out[i].UserID])
		out[i].Photos = orEmptyPhotos(photos[out[i].UserID])
	}
	return out, nil
}

// PublicColumns is the viewer-relative projection (aliases me = viewer, t = target) that discovery
// completes with its own FROM/JOIN/WHERE/ORDER BY clauses.
const PublicColumns = publicColumns

func (r *PGRepository) interestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Interest{}
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

func (r *PGRepository) photosFor(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Photo{}
	for rows.Next() {
		var uid, id string
		var pos int
		if err := rows.Scan(&uid, &id, &pos); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], NewPhoto(id, pos))
	}
	return out, rows.Err()
}

// AgeAt returns the age in full years on the given day.
func AgeAt(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

// ApproxDistance rounds up to 5 km steps (minimum 5) so exact positions are never exposed.
func ApproxDistance(km *float64) *int {
	if km == nil {
		return nil
	}
	v := int(math.Ceil(*km/5.0)) * 5
	if v < 5 {
		v = 5
	}
	return &v
}

func orEmptyInterests(v []Interest) []Interest {
	if v == nil {
		return []Interest{}
	}
	return v
}

func orEmptyPhotos(v []Photo) []Photo {
	if v == nil {
		return []Photo{}
	}
	return v
}
