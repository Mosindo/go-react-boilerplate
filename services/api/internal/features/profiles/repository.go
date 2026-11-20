package profiles

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound        = errors.New("profile not found")
	ErrRepositoryUnknownInterest = errors.New("unknown interest")
)

type StoredProfile struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	Latitude         *float64
	Longitude        *float64
	Discoverable     bool
	IsDemo           bool
}

type StoredPhoto struct {
	ID       string
	UserID   string
	Position int
}

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetProfile(ctx context.Context, userID string) (StoredProfile, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertProfile(ctx context.Context, userID string, in UpsertProfileInput, birthdate time.Time) error
	UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lng *float64) error
	SetDiscoverable(ctx context.Context, userID string, discoverable bool) error
	ProfilesByIDs(ctx context.Context, userIDs []string) ([]StoredProfile, error)
	InterestsByUserIDs(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	PhotosByUserIDs(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT id, slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label, &i.Category)
		return i, err
	})
}

const profileColumns = `p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city,
	p.height_cm, p.relationship_goal, p.location_lat, p.location_lng, p.discoverable, u.is_demo`

func scanProfile(row pgx.Row) (StoredProfile, error) {
	var p StoredProfile
	var height *int16
	err := row.Scan(&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&height, &p.RelationshipGoal, &p.Latitude, &p.Longitude, &p.Discoverable, &p.IsDemo)
	if height != nil {
		h := int(*height)
		p.HeightCm = &h
	}
	return p, err
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (StoredProfile, error) {
	p, err := scanProfile(r.dbPool.QueryRow(ctx, `
		SELECT `+profileColumns+`
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredProfile{}, ErrRepositoryNotFound
	}
	return p, err
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var prefs Preferences
	var ageMin, ageMax, distance int16
	err := r.dbPool.QueryRow(ctx, `
		SELECT interested_in, age_min, age_max, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&prefs.InterestedIn, &ageMin, &ageMax, &distance)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	prefs.AgeMin, prefs.AgeMax, prefs.MaxDistanceKm = int(ageMin), int(ageMax), int(distance)
	return &prefs, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in UpsertProfileInput, birthdate time.Time) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The birthdate is only written on creation: it cannot be edited later.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			job_title = EXCLUDED.job_title,
			city = EXCLUDED.city,
			height_cm = EXCLUDED.height_cm,
			relationship_goal = EXCLUDED.relationship_goal,
			updated_at = NOW()
	`, userID, in.FirstName, birthdate, in.Gender, in.Bio, in.JobTitle, in.City, in.HeightCm, in.RelationshipGoal); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, unnest($2::smallint[])
		`, userID, in.InterestIDs); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return ErrRepositoryUnknownInterest
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, prefs Preferences) error {
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			age_min = EXCLUDED.age_min,
			age_max = EXCLUDED.age_max,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, prefs.InterestedIn, prefs.AgeMin, prefs.AgeMax, prefs.MaxDistanceKm)
	return err
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lng *float64) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE profiles
		SET location_lat = $2, location_lng = $3,
		    location_updated_at = CASE WHEN $2::double precision IS NULL THEN NULL ELSE NOW() END,
		    updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) SetDiscoverable(ctx context.Context, userID string, discoverable bool) error {
	tag, err := r.dbPool.Exec(ctx, `UPDATE profiles SET discoverable = $2, updated_at = NOW() WHERE user_id = $1`, userID, discoverable)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) ProfilesByIDs(ctx context.Context, userIDs []string) ([]StoredProfile, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+profileColumns+`
		FROM profiles p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredProfile, error) {
		return scanProfile(row)
	})
}

func (r *PGRepository) InterestsByUserIDs(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label, i.category
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]Interest)
	for rows.Next() {
		var userID string
		var i Interest
		if err := rows.Scan(&userID, &i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], i)
	}
	return out, rows.Err()
}

func (r *PGRepository) PhotosByUserIDs(ctx context.Context, userIDs []string) (map[string][]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, user_id, position FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]StoredPhoto)
	for rows.Next() {
		var p StoredPhoto
		var position int16
		if err := rows.Scan(&p.ID, &p.UserID, &position); err != nil {
			return nil, err
		}
		p.Position = int(position)
		out[p.UserID] = append(out[p.UserID], p)
	}
	return out, rows.Err()
}

// CanView applies the server-side visibility rules for a full profile:
// no block in either direction, account not suspended, and the target is
// either discoverable or already matched with the viewer.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.moderation_status <> 'suspended'
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				$1 = $2
				OR (p.discoverable AND u.moderation_status = 'active')
				OR EXISTS (
					SELECT 1 FROM matches m
					WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
				)
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
