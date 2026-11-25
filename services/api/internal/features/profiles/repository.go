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
	errRepoNotFound        = errors.New("profile not found")
	errRepoUnknownInterest = errors.New("unknown interest")
)

type profileInput struct {
	FirstName   string
	Birthdate   time.Time
	Gender      string
	Bio         string
	JobTitle    *string
	City        *string
	HeightCm    *int
	LookingFor  *string
	InterestIDs []int
}

type Repository interface {
	GetProfile(ctx context.Context, userID string) (storedProfile, error)
	UpsertProfile(ctx context.Context, userID string, in profileInput) error
	GetPreferences(ctx context.Context, userID string) (Preferences, bool, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, p Privacy) error
	UpdateLocation(ctx context.Context, userID string, lat, lon float64) error
	ListInterests(ctx context.Context) ([]Interest, error)
	InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error)
	VisibleCard(ctx context.Context, viewerID, targetID string) (CardRow, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birthdate, gender, bio, job_title, city, height_cm, looking_for,
		       latitude, longitude, location_updated_at, discoverable, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City, &p.HeightCm, &p.LookingFor,
		&p.Latitude, &p.Longitude, &p.LocationUpdatedAt, &p.Discoverable, &p.ShowDistance)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedProfile{}, errRepoNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in profileInput) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles (user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, looking_for)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (user_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				birthdate = EXCLUDED.birthdate,
				gender = EXCLUDED.gender,
				bio = EXCLUDED.bio,
				job_title = EXCLUDED.job_title,
				city = EXCLUDED.city,
				height_cm = EXCLUDED.height_cm,
				looking_for = EXCLUDED.looking_for,
				updated_at = NOW()
		`, userID, in.FirstName, in.Birthdate, in.Gender, in.Bio, in.JobTitle, in.City, in.HeightCm, in.LookingFor); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO preferences (user_id, interested_in) VALUES ($1, '{}')
			ON CONFLICT (user_id) DO NOTHING
		`, userID); err != nil {
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
					return errRepoUnknownInterest
				}
				return err
			}
		}
		return nil
	})
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, bool, error) {
	var p Preferences
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{InterestedIn: []string{}, MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: DefaultDistance}, false, nil
	}
	return p, err == nil, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, p Privacy) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, p.Discoverable, p.ShowDistance)
	if err == nil && tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return err
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lon float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lon)
	if err == nil && tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return err
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label)
		return i, err
	})
}

func (r *PGRepository) InterestsForUsers(ctx context.Context, userIDs []string) (map[string][]Interest, error) {
	out := make(map[string][]Interest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[])
		ORDER BY i.label
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var i Interest
		if err := rows.Scan(&userID, &i.ID, &i.Slug, &i.Label); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], i)
	}
	return out, rows.Err()
}

// CardColumns selects a CardRow from profile alias `p`, computing the
// distance to the viewer profile alias `me`.
const CardColumns = `
	p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city, p.height_cm, p.looking_for,
	p.show_distance,
	CASE WHEN me.latitude IS NOT NULL AND p.latitude IS NOT NULL
	     THEN distance_km(me.latitude, me.longitude, p.latitude, p.longitude) END AS distance,
	(SELECT u.is_demo FROM users u WHERE u.id = p.user_id) AS is_demo_profile`

func ScanCard(row pgx.CollectableRow) (CardRow, error) {
	var c CardRow
	err := row.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &c.City, &c.HeightCm,
		&c.LookingFor, &c.ShowDistance, &c.DistanceKm, &c.IsDemo)
	return c, err
}

// VisibleCard returns target's card only if viewer may see it: no block in
// either direction and the target is discoverable or matched with viewer.
func (r *PGRepository) VisibleCard(ctx context.Context, viewerID, targetID string) (CardRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+CardColumns+`
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = $2
		  AND (
		    p.user_id = $1
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
		      )
		      AND (
		        p.discoverable
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.ended_at IS NULL
		            AND m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
		        )
		      )
		    )
		  )
	`, viewerID, targetID)
	if err != nil {
		return CardRow{}, err
	}
	card, err := pgx.CollectExactlyOneRow(rows, ScanCard)
	if errors.Is(err, pgx.ErrNoRows) {
		return CardRow{}, errRepoNotFound
	}
	return card, err
}
