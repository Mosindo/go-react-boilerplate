package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("profile not found")

// HaversineKm is the SQL expression for the great-circle distance in km between two lat/lng pairs.
func HaversineKm(lat1, lng1, lat2, lng2 string) string {
	return "(6371 * 2 * asin(least(1, sqrt(power(sin(radians((" + lat2 + ") - (" + lat1 + ")) / 2), 2) + " +
		"cos(radians(" + lat1 + ")) * cos(radians(" + lat2 + ")) * power(sin(radians((" + lng2 + ") - (" + lng1 + ")) / 2), 2)))))"
}

type CardRow struct {
	UserID     string
	FirstName  string
	Age        int
	Gender     string
	Bio        string
	City       string
	DistanceKm *float64 // nil when the target hides it
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	UpsertProfile(ctx context.Context, p StoredProfile, interestSlugs []string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	CountKnownInterests(ctx context.Context, slugs []string) (int, error)
	InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	CardRows(ctx context.Context, viewerID string, ids []string) ([]CardRow, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude, longitude, discoverable, show_distance,
		       EXTRACT(YEAR FROM age(birth_date))::int
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude,
			&p.Discoverable, &p.ShowDistance, &p.Age)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrRepoNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, p StoredProfile, slugs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
		  first_name = EXCLUDED.first_name, gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city,
		  latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude,
		  discoverable = EXCLUDED.discoverable, show_distance = EXCLUDED.show_distance, updated_at = NOW()`,
		p.UserID, p.FirstName, p.BirthDate, p.Gender, p.Bio, p.City, p.Latitude, p.Longitude, p.Discoverable, p.ShowDistance)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, p.UserID); err != nil {
		return err
	}
	if len(slugs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)`, p.UserID, slugs); err != nil {
			return err
		}
	}
	// Sensible defaults so the account is usable before the preference screen is saved.
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, ARRAY['woman','man','nonbinary'], 18, 99, 50) ON CONFLICT (user_id) DO NOTHING`, p.UserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepoNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		return i, row.Scan(&i.Slug, &i.Label)
	})
}

func (r *PGRepository) CountKnownInterests(ctx context.Context, slugs []string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE slug = ANY($1)`, slugs).Scan(&n)
	return n, err
}

func (r *PGRepository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id::text, i.slug, i.label FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest)
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

// CardRows loads the public columns of many profiles at once, with the distance from the viewer.
func (r *PGRepository) CardRows(ctx context.Context, viewerID string, ids []string) ([]CardRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id::text, p.first_name, EXTRACT(YEAR FROM age(p.birth_date))::int, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND me.latitude IS NOT NULL
		            THEN `+HaversineKm("me.latitude", "me.longitude", "p.latitude", "p.longitude")+` END
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (CardRow, error) {
		var c CardRow
		return c, row.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.DistanceKm)
	})
}
