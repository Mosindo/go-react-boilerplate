package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterests = errors.New("unknown interests")
)

type Repository interface {
	GetOwn(ctx context.Context, userID string) (storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	Upsert(ctx context.Context, userID string, in ProfileInput) error
	SavePreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotoIDsFor(ctx context.Context, userIDs []string) (map[string][]string, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	LoadCards(ctx context.Context, viewerID string, ids []string) (map[string]CardRow, error)
}

// CardRow is a raw row for Card assembly (distance still unbucketed).
type CardRow struct {
	ID         string
	FirstName  string
	BirthYear  int
	BirthMonth int
	BirthDay   int
	Gender     string
	Bio        string
	City       string
	DistanceKm *float64
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	var gender *string
	var lat, lng *float64
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.birth_date, p.first_name, p.gender, p.bio, p.city, p.latitude, p.longitude, p.is_discoverable, p.show_distance
		FROM users u JOIN profiles p ON p.user_id = u.id
		WHERE u.id = $1`, userID).Scan(&p.UserID, &p.BirthDate, &p.FirstName, &gender, &p.Bio, &p.City, &lat, &lng, &p.IsDiscoverable, &p.ShowDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedProfile{}, ErrNotFound
		}
		return storedProfile{}, err
	}
	if gender != nil {
		p.Gender = *gender
	}
	if lat != nil && lng != nil {
		p.Location = &Location{Latitude: *lat, Longitude: *lng}
	}
	return p, nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p := Preferences{InterestedIn: []string{}, AgeMin: 18, AgeMax: 99, MaxDistanceKm: 50}
	err := r.db.QueryRow(ctx, `SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.AgeMin, &p.AgeMax, &p.MaxDistanceKm)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, err
	}
	if p.InterestedIn == nil {
		p.InterestedIn = []string{}
	}
	return p, nil
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lat, lng any
	if in.Location != nil {
		lat, lng = in.Location.Latitude, in.Location.Longitude
	}
	var gender any
	if in.Gender != "" {
		gender = in.Gender
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, gender, bio, city, latitude, longitude, is_discoverable, show_distance)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city,
			latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
			is_discoverable = EXCLUDED.is_discoverable, show_distance = EXCLUDED.show_distance, updated_at = NOW()`,
		userID, in.FirstName, gender, in.Bio, in.City, lat, lng, in.IsDiscoverable, in.ShowDistance); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestSlugs) > 0 {
		tag, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)`, userID, in.InterestSlugs)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(in.InterestSlugs) {
			return ErrUnknownInterests
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()`,
		userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrNotFound
	}
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

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := map[string][]Interest{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, userIDs)
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

func (r *PGRepository) PhotoIDsFor(ctx context.Context, userIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `SELECT user_id, id FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, pid string
		if err := rows.Scan(&uid, &pid); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], pid)
	}
	return out, rows.Err()
}

// CanView reports whether viewer may see target's profile: never when either
// side blocked the other; otherwise only when target is discoverable, or the
// two are matched, or target already liked viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (
				p.is_discoverable
				OR EXISTS (SELECT 1 FROM matches m WHERE m.ended_at IS NULL AND m.user_low = LEAST($1::uuid, $2::uuid) AND m.user_high = GREATEST($1::uuid, $2::uuid))
				OR EXISTS (SELECT 1 FROM likes l WHERE l.swiper_id = $2 AND l.target_id = $1 AND l.kind = 'like')
			  )
		)`, viewerID, targetID).Scan(&ok)
	if isInvalidUUID(err) {
		return false, nil
	}
	return ok, err
}

// LoadCards returns raw card rows for ids with a distance computed from the
// viewer's stored (rounded) coordinates when both sides share a location and
// the target allows distance display.
func (r *PGRepository) LoadCards(ctx context.Context, viewerID string, ids []string) (map[string]CardRow, error) {
	out := map[string]CardRow{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, EXTRACT(YEAR FROM u.birth_date)::int, EXTRACT(MONTH FROM u.birth_date)::int,
		       EXTRACT(DAY FROM u.birth_date)::int, COALESCE(p.gender,''), p.bio, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND me.latitude IS NOT NULL THEN
		         2 * 6371 * asin(sqrt(
		           power(sin(radians(p.latitude - me.latitude) / 2), 2) +
		           cos(radians(me.latitude)) * cos(radians(p.latitude)) *
		           power(sin(radians(p.longitude - me.longitude) / 2), 2)))
		       END
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c CardRow
		if err := rows.Scan(&c.ID, &c.FirstName, &c.BirthYear, &c.BirthMonth, &c.BirthDay, &c.Gender, &c.Bio, &c.City, &c.DistanceKm); err != nil {
			return nil, err
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
