package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("profile not found")
	ErrUnknownInterests = errors.New("unknown interests")
)

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       Gender
	Bio          string
	City         string
	HasLocation  bool
	Discoverable bool
	ShowDistance bool
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func PhotoURLs(id string, position int) Photo {
	return Photo{ID: id, Position: position, IsPrimary: position == 0, URL: "/photos/" + id + "/image", ThumbURL: "/photos/" + id + "/image?size=thumb"}
}

func (r *Repository) getProfile(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, city, latitude IS NOT NULL, discoverable, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.HasLocation, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrNotFound
	}
	return p, err
}

func (r *Repository) getPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var interested []string
	var prefs Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&interested, &prefs.MinAge, &prefs.MaxAge, &prefs.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	prefs.InterestedIn = make([]Gender, len(interested))
	for i, g := range interested {
		prefs.InterestedIn[i] = Gender(g)
	}
	return &prefs, nil
}

func (r *Repository) upsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Resolve interests first so unknown slugs fail before any write.
	var interestIDs []int16
	if len(in.InterestSlug) > 0 {
		rows, err := tx.Query(ctx, `SELECT id FROM interests WHERE slug = ANY($1)`, in.InterestSlug)
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int16
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			interestIDs = append(interestIDs, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(interestIDs) != len(in.InterestSlug) {
			return ErrUnknownInterests
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date, gender = EXCLUDED.gender,
			bio = EXCLUDED.bio, discoverable = EXCLUDED.discoverable, show_distance = EXCLUDED.show_distance,
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, string(in.Gender), in.Bio, in.Discoverable, in.ShowDistance); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(interestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id) SELECT $1, UNNEST($2::smallint[])
		`, userID, interestIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) upsertPreferences(ctx context.Context, userID string, p Preferences) error {
	interested := make([]string, len(p.InterestedIn))
	for i, g := range p.InterestedIn {
		interested[i] = string(g)
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age, max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, interested, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *Repository) setLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = $4, updated_at = NOW() WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) clearLocation(ctx context.Context, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = NULL, longitude = NULL, city = '', updated_at = NOW() WHERE user_id = $1
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) listInterests(ctx context.Context) ([]Interest, error) {
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

// InterestsByUsers loads interests for many users in one query.
func (r *Repository) InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label
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

// PhotosByUsers loads photo metadata (never the bytes) for many users in one query.
func (r *Repository) PhotosByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	out := make(map[string][]Photo, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, id string
		var pos int
		if err := rows.Scan(&uid, &id, &pos); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], PhotoURLs(id, pos))
	}
	return out, rows.Err()
}

type publicRow struct {
	UserID     string
	FirstName  string
	Age        int
	Bio        string
	City       string
	DistanceKm *int
}

func (r *Repository) publicRows(ctx context.Context, viewerID string, ids []string) ([]publicRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, `+db.Age("p.birth_date")+`, p.bio, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		            THEN `+db.ApproxDistance(db.DistanceKm("v.latitude", "v.longitude", "p.latitude", "p.longitude"))+` END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []publicRow
	for rows.Next() {
		var row publicRow
		if err := rows.Scan(&row.UserID, &row.FirstName, &row.Age, &row.Bio, &row.City, &row.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// canView encodes who may see a profile: complete, not blocked either way, and discoverable or matched
// (or the viewer themself).
func (r *Repository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2::uuid
			  AND `+db.ProfileComplete("p.user_id")+`
			  AND `+db.NotBlocked("$1::uuid", "$2::uuid")+`
			  AND (p.discoverable OR $1::uuid = $2::uuid OR EXISTS (
			        SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
