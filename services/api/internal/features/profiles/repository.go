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
	ErrNotFound        = errors.New("profiles: not found")
	ErrUnknownInterest = errors.New("profiles: unknown interest")
	// ErrViewerIncomplete means the viewer has no profile yet, so no distance can be computed.
	ErrViewerIncomplete = errors.New("profiles: viewer profile incomplete")
)

type Repository interface {
	GetMe(ctx context.Context, userID string) (MeResponse, error)
	GetProfile(ctx context.Context, userID string) (MyProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	GetPreferences(ctx context.Context, userID string) (Preferences, bool, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	ListInterests(ctx context.Context) ([]Interest, error)
	GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) GetMe(ctx context.Context, userID string) (MeResponse, error) {
	var me MeResponse
	err := r.db.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.created_at,
		       EXISTS (SELECT 1 FROM profiles WHERE user_id = u.id),
		       EXISTS (SELECT 1 FROM photos WHERE user_id = u.id),
		       EXISTS (SELECT 1 FROM preferences WHERE user_id = u.id)
		FROM users u WHERE u.id = $1
	`, userID).Scan(&me.ID, &me.Email, &me.CreatedAt, &me.Onboarding.Profile, &me.Onboarding.Photos, &me.Onboarding.Preferences)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MeResponse{}, ErrNotFound
		}
		return MeResponse{}, err
	}
	me.Onboarding.Complete = me.Onboarding.Profile && me.Onboarding.Photos && me.Onboarding.Preferences
	return me, nil
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (MyProfile, error) {
	var p MyProfile
	var birth time.Time
	var gender string
	err := r.db.QueryRow(ctx, `
		SELECT first_name, birth_date, gender, bio, city, latitude, longitude, discoverable, show_distance
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &birth, &gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude, &p.Discoverable, &p.ShowDistance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyProfile{}, ErrNotFound
		}
		return MyProfile{}, err
	}
	p.Gender = Gender(gender)
	p.BirthDate = birth.Format("2006-01-02")
	p.Age = AgeOn(birth, time.Now())
	p.Interests = []Interest{}
	p.Photos = []Photo{}

	pub := &PublicProfile{ID: userID, Interests: []Interest{}, Photos: []Photo{}}
	if err := Hydrate(ctx, r.db, []*PublicProfile{pub}); err != nil {
		return MyProfile{}, err
	}
	p.Interests = pub.Interests
	p.Photos = pub.Photos
	return p, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var interestIDs []int16
	if len(in.InterestSlugs) > 0 {
		rows, err := tx.Query(ctx, `SELECT id FROM interests WHERE slug = ANY($1)`, in.InterestSlugs)
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
		if len(interestIDs) != len(in.InterestSlugs) {
			return ErrUnknownInterest
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, latitude, longitude, discoverable, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, TRUE), COALESCE($10, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			birth_date = EXCLUDED.birth_date,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			discoverable = COALESCE($9, profiles.discoverable),
			show_distance = COALESCE($10, profiles.show_distance),
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, string(in.Gender), in.Bio, in.City, in.Latitude, in.Longitude, in.Discoverable, in.ShowDistance); err != nil {
		if isForeignKeyViolation(err) {
			return ErrNotFound
		}
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1 AND NOT (interest_id = ANY($2::smallint[]))`, userID, interestIDs); err != nil {
		return err
	}
	if len(interestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, UNNEST($2::smallint[])
			ON CONFLICT DO NOTHING
		`, userID, interestIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, bool, error) {
	var p Preferences
	var interested []string
	err := r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&interested, &p.MinAge, &p.MaxAge, &p.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Preferences{}, false, nil
		}
		return Preferences{}, false, err
	}
	p.InterestedIn = make([]Gender, len(interested))
	for i, g := range interested {
		p.InterestedIn[i] = Gender(g)
	}
	return p, true, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	genders := make([]string, len(p.InterestedIn))
	for i, g := range p.InterestedIn {
		genders[i] = string(g)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, genders, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil && isForeignKeyViolation(err) {
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
		var in Interest
		if err := rows.Scan(&in.Slug, &in.Label); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

// GetPublicProfile returns another user's profile if the viewer may see it:
// never when either side blocked the other, and only when the target is
// discoverable, matched with the viewer, or has already liked the viewer.
func (r *PGRepository) GetPublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	var hasProfile bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, viewerID).Scan(&hasProfile); err != nil {
		return PublicProfile{}, err
	}
	if !hasProfile {
		return PublicProfile{}, ErrViewerIncomplete
	}

	var p PublicProfile
	err := ScanPublic(r.db.QueryRow(ctx, `
		SELECT `+PublicSelect+`
		FROM profiles p
		JOIN profiles v ON v.user_id = $1
		WHERE p.user_id = $2
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (p.user_id = $1 OR p.discoverable
		    OR EXISTS (SELECT 1 FROM matches m
		               WHERE m.unmatched_at IS NULL AND m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
		    OR EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = $2 AND s.target_id = $1 AND s.action = 'like'))
	`, viewerID, targetID), &p)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return PublicProfile{}, ErrNotFound
		}
		return PublicProfile{}, err
	}
	if err := Hydrate(ctx, r.db, []*PublicProfile{&p}); err != nil {
		return PublicProfile{}, err
	}
	return p, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
