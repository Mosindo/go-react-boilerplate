package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("profile not found")

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type profileRow struct {
	FirstName   string
	BirthDate   time.Time
	Gender      string
	Bio         string
	City        string
	HasLocation bool
	Privacy     Privacy
}

func (r *Repository) ListInterests(ctx context.Context) ([]Interest, error) {
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

func (r *Repository) Upsert(ctx context.Context, userID string, p UpsertProfileRequest, birth time.Time, interests []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city, updated_at = NOW()
	`, userID, p.FirstName, birth, p.Gender, p.Bio, p.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(interests) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = ANY($2)
		`, userID, interests); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CountKnownInterests returns how many of the slugs exist.
func (r *Repository) CountKnownInterests(ctx context.Context, slugs []string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM interests WHERE slug = ANY($1)`, slugs).Scan(&n)
	return n, err
}

func (r *Repository) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	tag, err := r.db.Exec(ctx, `UPDATE profiles SET lat = $2, lng = $3, updated_at = NOW() WHERE user_id = $1`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ClearLocation(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE profiles SET lat = NULL, lng = NULL, updated_at = NOW() WHERE user_id = $1`, userID)
	return err
}

func (r *Repository) SetPrivacy(ctx context.Context, userID string, showDistance, discoverable *bool) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET
			show_distance = COALESCE($2, show_distance),
			discoverable = COALESCE($3, discoverable),
			updated_at = NOW()
		WHERE user_id = $1`, userID, showDistance, discoverable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) SetPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in, age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.AgeMin, p.AgeMax, p.MaxDistanceKm)
	return err
}

func (r *Repository) GetMine(ctx context.Context, userID string) (MyProfile, error) {
	var (
		row profileRow
		me  MyProfile
	)
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, lat IS NOT NULL, show_distance, discoverable,
			date_part('year', age(birth_date))::int
		FROM profiles WHERE user_id = $1`, userID,
	).Scan(&row.FirstName, &row.BirthDate, &row.Gender, &row.Bio, &row.City, &row.HasLocation,
		&row.Privacy.ShowDistance, &row.Privacy.Discoverable, &me.Age)
	if errors.Is(err, pgx.ErrNoRows) {
		return MyProfile{}, ErrNotFound
	}
	if err != nil {
		return MyProfile{}, err
	}
	me.UserID, me.FirstName, me.BirthDate = userID, row.FirstName, row.BirthDate.Format("2006-01-02")
	me.Gender, me.Bio, me.City, me.HasLocation, me.Privacy = row.Gender, row.Bio, row.City, row.HasLocation, row.Privacy

	me.Interests = []Interest{}
	irows, err := r.db.Query(ctx, `
		SELECT i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1 ORDER BY i.label`, userID)
	if err != nil {
		return MyProfile{}, err
	}
	for irows.Next() {
		var i Interest
		if err := irows.Scan(&i.Slug, &i.Label); err != nil {
			irows.Close()
			return MyProfile{}, err
		}
		me.Interests = append(me.Interests, i)
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return MyProfile{}, err
	}

	me.Photos = []PhotoRef{}
	prows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return MyProfile{}, err
	}
	for prows.Next() {
		var p PhotoRef
		if err := prows.Scan(&p.ID, &p.Position); err != nil {
			prows.Close()
			return MyProfile{}, err
		}
		p.URL = PhotoURL(p.ID)
		me.Photos = append(me.Photos, p)
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		return MyProfile{}, err
	}

	var prefs Preferences
	err = r.db.QueryRow(ctx, `SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&prefs.InterestedIn, &prefs.AgeMin, &prefs.AgeMax, &prefs.MaxDistanceKm)
	switch {
	case err == nil:
		me.Preferences = &prefs
	case !errors.Is(err, pgx.ErrNoRows):
		return MyProfile{}, err
	}
	return me, nil
}

// GetCard returns another user's public card if the viewer may see it.
func (r *Repository) GetCard(ctx context.Context, viewerID, targetID string) (Card, error) {
	card, err := ScanCard(r.db.QueryRow(ctx, `
		SELECT `+CardColumns+`
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = $2 AND p.user_id <> $1 AND `+VisibleToViewer, viewerID, targetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Card{}, ErrNotFound
	}
	return card, err
}
