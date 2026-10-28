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
	ErrRepoNotFound        = errors.New("profile not found")
	ErrRepoUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	Upsert(ctx context.Context, userID string, birthDate time.Time, in ProfileInput) error // nil flags keep stored values
	GetBirthDate(ctx context.Context, userID string) (time.Time, error)
	SetLocation(ctx context.Context, userID string, lat, lon float64, city string) error
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label, category FROM interests ORDER BY category, label`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		err := row.Scan(&i.ID, &i.Slug, &i.Label, &i.Category)
		return i, err
	})
}

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	var sp storedProfile
	err := r.db.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, date_part('year', age(CURRENT_DATE, birth_date))::int,
		       gender, bio, city, latitude IS NOT NULL, show_distance, discoverable
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&sp.UserID, &sp.FirstName, &sp.BirthDate, &sp.Age, &sp.Gender, &sp.Bio, &sp.City, &sp.HasLocation, &sp.ShowDistance, &sp.Discoverable)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OwnProfile{}, ErrRepoNotFound
		}
		return OwnProfile{}, err
	}
	interests, photos, err := r.loadRelations(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	return OwnProfile{
		UserID: sp.UserID, FirstName: sp.FirstName, BirthDate: sp.BirthDate.Format("2006-01-02"), Age: sp.Age,
		Gender: sp.Gender, Bio: sp.Bio, City: sp.City, HasLocation: sp.HasLocation,
		ShowDistance: sp.ShowDistance, Discoverable: sp.Discoverable,
		Interests: orEmpty(interests[userID]), Photos: orEmptyPhotos(photos[userID]),
	}, nil
}

func (r *PGRepository) GetBirthDate(ctx context.Context, userID string) (time.Time, error) {
	var d time.Time
	err := r.db.QueryRow(ctx, `SELECT birth_date FROM profiles WHERE user_id = $1`, userID).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrRepoNotFound
	}
	return d, err
}

func (r *PGRepository) Upsert(ctx context.Context, userID string, birthDate time.Time, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, show_distance, discoverable)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, TRUE), COALESCE($8, TRUE))
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name, gender = EXCLUDED.gender, bio = EXCLUDED.bio, city = EXCLUDED.city,
			show_distance = COALESCE($7, profiles.show_distance), discoverable = COALESCE($8, profiles.discoverable), updated_at = NOW()
	`, userID, in.FirstName, birthDate, string(in.Gender), in.Bio, in.City, in.ShowDistance, in.Discoverable); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestIDs) > 0 {
		if _, err := tx.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, unnest($2::smallint[])`, userID, in.InterestIDs); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return ErrRepoUnknownInterest
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon float64, city string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, city = CASE WHEN $4 <> '' THEN $4 ELSE city END, updated_at = NOW()
		WHERE user_id = $1`, userID, lat, lon, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	var p Preferences
	var genders []string
	var minAge, maxAge, maxDist int16
	err := r.db.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&genders, &minAge, &maxAge, &maxDist)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	for _, g := range genders {
		p.InterestedIn = append(p.InterestedIn, Gender(g))
	}
	p.MinAge, p.MaxAge, p.MaxDistanceKm = int(minAge), int(maxAge), int(maxDist)
	return &p, nil
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	genders := make([]string, len(p.InterestedIn))
	for i, g := range p.InterestedIn {
		genders[i] = string(g)
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET interested_in = EXCLUDED.interested_in, min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age, max_distance_km = EXCLUDED.max_distance_km, updated_at = NOW()
	`, userID, genders, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	return err
}

// CanView is the single visibility rule for a profile and its photos: the owner always;
// otherwise the target must exist, not be blocked in either direction, and be discoverable or matched.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	if viewerID == targetID {
		var exists bool
		err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, viewerID).Scan(&exists)
		return exists, err
	}
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND NOT EXISTS (SELECT 1 FROM blocks b
			                  WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (p.discoverable OR EXISTS (SELECT 1 FROM matches m
			                                 WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// PublicByIDs loads profiles in the given order with interests and photos in two extra queries (no N+1).
// Distance is computed from the coarse stored coordinates and then rounded up to 5 km steps.
func (r *PGRepository) PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	if len(ids) == 0 {
		return []PublicProfile{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, p.first_name, date_part('year', age(CURRENT_DATE, p.birth_date))::int, p.gender, p.bio, p.city,
		       CASE WHEN v.show_distance AND p.show_distance AND v.latitude IS NOT NULL AND p.latitude IS NOT NULL
		            THEN haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) END
		FROM unnest($2::uuid[]) WITH ORDINALITY AS wanted(id, ord)
		JOIN profiles p ON p.user_id = wanted.id
		LEFT JOIN profiles v ON v.user_id = $1
		ORDER BY wanted.ord`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PublicProfile, 0, len(ids))
	for rows.Next() {
		var p PublicProfile
		var dist *float64
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &dist); err != nil {
			return nil, err
		}
		p.DistanceKm = ApproxDistance(dist)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	loaded := make([]string, len(out))
	for i := range out {
		loaded[i] = out[i].UserID
	}
	interests, photos, err := r.loadRelations(ctx, loaded)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Interests = orEmpty(interests[out[i].UserID])
		out[i].Photos = orEmptyPhotos(photos[out[i].UserID])
	}
	return out, nil
}

func (r *PGRepository) loadRelations(ctx context.Context, userIDs []string) (map[string][]Interest, map[string][]Photo, error) {
	interests := make(map[string][]Interest)
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label, i.category
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, userIDs)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var uid string
		var i Interest
		if err := rows.Scan(&uid, &i.ID, &i.Slug, &i.Label, &i.Category); err != nil {
			rows.Close()
			return nil, nil, err
		}
		interests[uid] = append(interests[uid], i)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	photos := make(map[string][]Photo)
	prows, err := r.db.Query(ctx, `
		SELECT user_id, id, position, width, height FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var uid string
		var p Photo
		if err := prows.Scan(&uid, &p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, nil, err
		}
		p.URL = "/photos/" + p.ID + "/file"
		photos[uid] = append(photos[uid], p)
	}
	return interests, photos, prows.Err()
}

func orEmpty(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}

func orEmptyPhotos(in []Photo) []Photo {
	if in == nil {
		return []Photo{}
	}
	return in
}
