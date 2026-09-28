package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profiles: not found")
	ErrNoProfile       = errors.New("profiles: profile does not exist")
	ErrBirthDateChange = errors.New("profiles: birth date is immutable")
	ErrUnknownInterest = errors.New("profiles: unknown interest")
)

// DefaultPreferences are the settings of a user who never customised them.
func DefaultPreferences() Preferences {
	return Preferences{InterestedIn: []string{GenderMan, GenderWoman, GenderNonBinary}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 50}
}

// Querier is satisfied by *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type UserRecord struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type Repository interface {
	GetUser(ctx context.Context, userID string) (UserRecord, error)
	GetProfile(ctx context.Context, userID string) (*ProfileRecord, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	SetLocation(ctx context.Context, userID string, lat, lon float64, label *string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosFor(ctx context.Context, userIDs []string) (map[string][]Photo, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) GetUser(ctx context.Context, userID string) (UserRecord, error) {
	var u UserRecord
	err := r.pool.QueryRow(ctx, `SELECT id, email, created_at FROM users WHERE id = $1`, userID).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UserRecord{}, ErrNotFound
	}
	u.CreatedAt = u.CreatedAt.UTC()
	return u, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (*ProfileRecord, error) {
	var p ProfileRecord
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, gender, bio, location_label,
		       latitude IS NOT NULL, show_distance, is_discoverable
		FROM profiles WHERE user_id = $1`, userID).Scan(
		&p.UserID, &p.FirstName, &p.BirthDate, &p.Gender, &p.Bio, &p.LocationLabel,
		&p.HasLocation, &p.ShowDistance, &p.IsDiscoverable)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpsertProfile creates or updates the profile and replaces the interest set in
// one transaction. The birth date can be set once and never changed.
func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if len(in.InterestIDs) > 0 {
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM interests WHERE id = ANY($1::smallint[])`, in.InterestIDs).Scan(&n); err != nil {
			return err
		}
		if n != len(in.InterestIDs) {
			return ErrUnknownInterest
		}
	}

	var existing time.Time
	err = tx.QueryRow(ctx, `SELECT birth_date FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&existing)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_, err = tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, show_distance, is_discoverable)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6, TRUE), COALESCE($7, TRUE))`,
			userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.ShowDistance, in.IsDiscoverable)
		if err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if !sameDate(existing, in.BirthDate) {
			return ErrBirthDateChange
		}
		_, err = tx.Exec(ctx, `
			UPDATE profiles SET first_name = $2, gender = $3, bio = $4,
			       show_distance = COALESCE($5, show_distance),
			       is_discoverable = COALESCE($6, is_discoverable),
			       updated_at = NOW()
			WHERE user_id = $1`,
			userID, in.FirstName, in.Gender, in.Bio, in.ShowDistance, in.IsDiscoverable)
		if err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])`, userID, in.InterestIDs); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO preferences (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon float64, label *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3,
		    location_label = COALESCE($4, location_label),
		    updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lon, label)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoProfile
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	var minAge, maxAge int16
	var dist int32
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &minAge, &maxAge, &dist)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultPreferences(), nil
	}
	if err != nil {
		return Preferences{}, err
	}
	p.MinAge, p.MaxAge, p.MaxDistanceKm = int(minAge), int(maxAge), int(dist)
	return p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
		    max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km,
		    updated_at = NOW()`,
		userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Interest{}
	for rows.Next() {
		var i Interest
		if err := rows.Scan(&i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *PGRepository) InterestsFor(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	return InterestsFor(ctx, r.pool, userIDs)
}

func (r *PGRepository) PhotosFor(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	return PhotosFor(ctx, r.pool, userIDs)
}

// InterestsFor loads the interests of many users with one query (no N+1).
func InterestsFor(ctx context.Context, q Querier, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY ui.user_id, i.label`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], i)
	}
	return out, rows.Err()
}

// PhotosFor loads photo metadata (never the image bytes) for many users.
func PhotosFor(ctx context.Context, q Querier, userIDs []string) (map[string][]Photo, error) {
	out := make(map[string][]Photo, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT user_id, id, position
		FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var p Photo
		var pos int16
		if err := rows.Scan(&uid, &p.ID, &pos); err != nil {
			return nil, err
		}
		p.Position = int(pos)
		p.URL = PhotoURL(p.ID)
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}
