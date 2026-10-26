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
	ErrRepositoryNotFound        = errors.New("profile not found")
	ErrRepositoryUnknownInterest = errors.New("unknown interest")
)

// Querier is satisfied by *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	JobTitle     string
	City         string
	HasLocation  bool
	IsVisible    bool
	ShowDistance bool
	CreatedAt    time.Time
}

type Repository interface {
	GetOwn(ctx context.Context, userID string) (storedProfile, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	Create(ctx context.Context, userID string, in profileInput, birthDate time.Time) error
	Update(ctx context.Context, userID string, in profileInput) error
	SetInterests(ctx context.Context, userID string, slugs []string) error
	SavePreferences(ctx context.Context, userID string, p Preferences) error
	SaveLocation(ctx context.Context, userID string, lat, lng float64, city string) error
	SavePrivacy(ctx context.Context, userID string, visible, showDistance bool) error
	CountPhotos(ctx context.Context, userID string) (int, error)
	ListInterests(ctx context.Context) ([]Interest, error)
}

type profileInput struct {
	FirstName string
	Gender    string
	Bio       string
	JobTitle  string
	Interests []string
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, job_title, city,
		       latitude IS NOT NULL, is_visible, show_distance, created_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&p.HasLocation, &p.IsVisible, &p.ShowDistance, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM user_preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PGRepository) Create(ctx context.Context, userID string, in profileInput, birthDate time.Time) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, job_title)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, in.FirstName, birthDate, in.Gender, in.Bio, in.JobTitle); err != nil {
			return err
		}
		return setInterests(ctx, tx, userID, in.Interests)
	})
}

func (r *PGRepository) Update(ctx context.Context, userID string, in profileInput) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE profiles SET first_name = $2, gender = $3, bio = $4, job_title = $5, updated_at = NOW()
			WHERE user_id = $1
		`, userID, in.FirstName, in.Gender, in.Bio, in.JobTitle)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRepositoryNotFound
		}
		return setInterests(ctx, tx, userID, in.Interests)
	})
}

func (r *PGRepository) SetInterests(ctx context.Context, userID string, slugs []string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error { return setInterests(ctx, tx, userID, slugs) })
}

func setInterests(ctx context.Context, tx pgx.Tx, userID string, slugs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(slugs) == 0 {
		return nil
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO user_interests (user_id, interest_id)
		SELECT $1, id FROM interests WHERE slug = ANY($2::text[])
	`, userID, slugs)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(slugs) {
		return ErrRepositoryUnknownInterest
	}
	return nil
}

func (r *PGRepository) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		    max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) SaveLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = $4, updated_at = NOW() WHERE user_id = $1
	`, userID, lat, lng, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) SavePrivacy(ctx context.Context, userID string, visible, showDistance bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET is_visible = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, visible, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) CountPhotos(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, label FROM interests ORDER BY label`)
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

func (r *PGRepository) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Loader builds PublicProfile views in batch (3 queries whatever the number of
// ids), so list endpoints never run N+1 queries.
type Loader struct {
	signer *signing.Signer
}

func NewLoader(signer *signing.Signer) *Loader { return &Loader{signer: signer} }

// Load returns the public profiles of ids as seen by viewerID, keyed by user id.
// Distance is approximated (rounded up to 5 km steps) and omitted when the
// profile owner hides it or either side has no location.
func (l *Loader) Load(ctx context.Context, q Querier, viewerID string, ids []string) (map[string]PublicProfile, error) {
	out := make(map[string]PublicProfile, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := q.Query(ctx, `
		SELECT p.user_id, p.first_name, date_part('year', age(p.birth_date))::int, p.gender, p.bio, p.job_title, p.city,
		       CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
		            THEN haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM profiles p
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p PublicProfile
		var dist *float64
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.JobTitle, &p.City, &dist); err != nil {
			rows.Close()
			return nil, err
		}
		p.DistanceKm = ApproxDistance(dist)
		p.Interests = []Interest{}
		p.Photos = []Photo{}
		out[p.UserID] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	irows, err := q.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label
	`, ids)
	if err != nil {
		return nil, err
	}
	for irows.Next() {
		var uid string
		var in Interest
		if err := irows.Scan(&uid, &in.Slug, &in.Label); err != nil {
			irows.Close()
			return nil, err
		}
		if p, ok := out[uid]; ok {
			p.Interests = append(p.Interests, in)
			out[uid] = p
		}
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return nil, err
	}

	prows, err := q.Query(ctx, `SELECT user_id, id FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var uid, pid string
		if err := prows.Scan(&uid, &pid); err != nil {
			return nil, err
		}
		if p, ok := out[uid]; ok {
			p.Photos = append(p.Photos, Photo{ID: pid, URL: l.signer.PhotoURL(pid)})
			out[uid] = p
		}
	}
	return out, prows.Err()
}

// ApproxDistance rounds a distance up to 5 km steps (minimum 5) so exact
// positions cannot be recovered by trilateration.
func ApproxDistance(km *float64) *int {
	if km == nil {
		return nil
	}
	steps := int(*km/5) + 1
	v := steps * 5
	return &v
}
