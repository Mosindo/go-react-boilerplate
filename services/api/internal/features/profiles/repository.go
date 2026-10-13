package profiles

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("profile not found")
	ErrUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	ListInterests(ctx context.Context) ([]Interest, error)
	GetOwn(ctx context.Context, userID string) (OwnProfile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	SetLocation(ctx context.Context, userID string, lat, lon *float64) error
	GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error)
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error)
	CanView(ctx context.Context, viewerID, ownerID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// viewableCondition restricts a profiles row (alias p) joined to users (alias u)
// to what viewer $1 may see: an active, visible profile with at least one photo
// and no block in either direction. Matched users stay visible even when paused.
const viewableCondition = `
	u.status = 'active'
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = $1::uuid AND b.blocked_id = p.user_id)
		   OR (b.blocker_id = p.user_id AND b.blocked_id = $1::uuid)
	)
	AND (
		p.is_visible
		OR EXISTS (
			SELECT 1 FROM matches m
			WHERE m.unmatched_at IS NULL
			  AND m.user_a = LEAST($1::uuid, p.user_id)
			  AND m.user_b = GREATEST($1::uuid, p.user_id)
		)
	)`

const publicColumns = `
	p.user_id, p.first_name, date_part('year', age(p.birth_date))::int, p.gender, p.bio, p.city,
	CASE WHEN p.show_distance AND p.latitude IS NOT NULL AND v.latitude IS NOT NULL
	     THEN haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) END,
	COALESCE((
		SELECT json_agg(json_build_object('id', i.id, 'slug', i.slug, 'label', i.label) ORDER BY i.id)
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = p.user_id
	), '[]'::json),
	COALESCE((
		SELECT json_agg(json_build_object('id', ph.id, 'position', ph.position) ORDER BY ph.position)
		FROM photos ph WHERE ph.user_id = p.user_id
	), '[]'::json)`

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.db.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY id`)
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

func (r *PGRepository) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	p := OwnProfile{ID: userID, InterestIDs: []int{}, Photos: []PhotoRef{}}
	err := r.db.QueryRow(ctx, `
		SELECT first_name, to_char(birth_date, 'YYYY-MM-DD'), date_part('year', age(birth_date))::int,
		       gender, bio, city, latitude, longitude, is_visible, show_distance, updated_at
		FROM profiles WHERE user_id = $1
	`, userID).Scan(&p.FirstName, &p.BirthDate, &p.Age, &p.Gender, &p.Bio, &p.City, &p.Latitude, &p.Longitude,
		&p.IsVisible, &p.ShowDistance, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return OwnProfile{}, ErrNotFound
	}
	if err != nil {
		return OwnProfile{}, err
	}

	rows, err := r.db.Query(ctx, `SELECT interest_id::int FROM user_interests WHERE user_id = $1 ORDER BY interest_id`, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return OwnProfile{}, err
		}
		p.InterestIDs = append(p.InterestIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return OwnProfile{}, err
	}

	prows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	for prows.Next() {
		var ph PhotoRef
		if err := prows.Scan(&ph.ID, &ph.Position); err != nil {
			prows.Close()
			return OwnProfile{}, err
		}
		p.Photos = append(p.Photos, ph)
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		return OwnProfile{}, err
	}

	var prefs Preferences
	err = r.db.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&prefs.InterestedIn, &prefs.MinAge, &prefs.MaxAge, &prefs.MaxDistanceKm)
	switch {
	case err == nil:
		p.Preferences = &prefs
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return OwnProfile{}, err
	}

	p.IsComplete = p.Preferences != nil && len(p.Photos) > 0
	return p, nil
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// birth_date is deliberately not updated on conflict: it is set once.
	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city, is_visible, show_distance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			gender = EXCLUDED.gender,
			bio = EXCLUDED.bio,
			city = EXCLUDED.city,
			is_visible = EXCLUDED.is_visible,
			show_distance = EXCLUDED.show_distance,
			updated_at = NOW()
	`, userID, in.FirstName, in.BirthDate, in.Gender, in.Bio, in.City, in.IsVisible, in.ShowDistance); err != nil {
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
				return ErrUnknownInterest
			}
			return err
		}
	}
	return tx.Commit(ctx)
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

func (r *PGRepository) SetLocation(ctx context.Context, userID string, lat, lon *float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, lat, lon)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPublic(row pgx.Row) (PublicProfile, error) {
	var p PublicProfile
	var dist *float64
	if err := row.Scan(&p.ID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &dist, &p.Interests, &p.Photos); err != nil {
		return PublicProfile{}, err
	}
	if dist != nil {
		b := DistanceBucket(*dist)
		p.DistanceKm = &b
	}
	return p, nil
}

func (r *PGRepository) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	p, err := scanPublic(r.db.QueryRow(ctx, `
		SELECT `+publicColumns+`
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN profiles v ON v.user_id = $1::uuid
		WHERE p.user_id = $2::uuid AND `+viewableCondition, viewerID, targetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicProfile{}, ErrNotFound
	}
	return p, err
}

// PublicByIDs loads many profiles in one query, preserving the order of ids.
// Callers are responsible for having checked visibility.
func (r *PGRepository) PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	if len(ids) == 0 {
		return []PublicProfile{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+publicColumns+`
		FROM unnest($2::uuid[]) WITH ORDINALITY AS t(id, ord)
		JOIN profiles p ON p.user_id = t.id
		LEFT JOIN profiles v ON v.user_id = $1::uuid
		ORDER BY t.ord
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PublicProfile, 0, len(ids))
	for rows.Next() {
		p, err := scanPublic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) CanView(ctx context.Context, viewerID, ownerID string) (bool, error) {
	if viewerID == ownerID {
		return true, nil
	}
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2::uuid AND `+viewableCondition+`
		)`, viewerID, ownerID).Scan(&ok)
	return ok, err
}
