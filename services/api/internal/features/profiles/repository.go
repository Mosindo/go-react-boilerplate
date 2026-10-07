package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Get(ctx context.Context, userID string) (StoredProfile, error) {
	var p StoredProfile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude IS NOT NULL
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.HasLocation)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrNotFound
	}
	return p, err
}

// Upsert writes the profile, makes sure a preferences row exists and replaces the interests.
// interestSlugs must already be validated against the catalog by the caller; the
// returned count lets the caller detect an unknown slug race.
func (r *Repository) Upsert(ctx context.Context, userID, firstName string, birthDate time.Time, gender, bio, city string, interestSlugs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
		    bio = EXCLUDED.bio, city = EXCLUDED.city, updated_at = NOW()
	`, userID, firstName, birthDate, gender, bio, city); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(interestSlugs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, interestSlugs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.pool.Exec(ctx, `
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

func (r *Repository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.Slug, &i.Label); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *Repository) CountKnownInterests(ctx context.Context, slugs []string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM interests WHERE slug = ANY($1)`, slugs).Scan(&n)
	return n, err
}

func (r *Repository) GetPreferences(ctx context.Context, userID string) (Preferences, bool, error) {
	var p Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km, discoverable, show_distance
		FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, false, nil
	}
	return p, err == nil, err
}

func (r *Repository) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age, max_age = EXCLUDED.max_age,
		    max_distance_km = EXCLUDED.max_distance_km, discoverable = EXCLUDED.discoverable,
		    show_distance = EXCLUDED.show_distance, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm, p.Discoverable, p.ShowDistance)
	return err
}

// --- batch readers shared with other features (single query per kind: no N+1) ---

type PhotoRow struct {
	ID       string
	Position int
}

func (r *Repository) PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]PhotoRow, error) {
	out := make(map[string][]PhotoRow, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var row PhotoRow
		if err := rows.Scan(&uid, &row.ID, &row.Position); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], row)
	}
	return out, rows.Err()
}

func (r *Repository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1) ORDER BY ui.user_id, i.label
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

type summaryRow struct {
	UserID    string
	FirstName string
	Age       int
}

func (r *Repository) SummaryRows(ctx context.Context, userIDs []string) ([]summaryRow, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, first_name, date_part('year', age(birth_date))::int
		FROM profiles WHERE user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []summaryRow
	for rows.Next() {
		var s summaryRow
		if err := rows.Scan(&s.UserID, &s.FirstName, &s.Age); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
