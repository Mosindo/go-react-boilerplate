package profiles

import (
	"context"
	"encoding/json"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotFound        = errors.New("profile not found")
	ErrRepoUnknownInterest = errors.New("unknown interest")
)

type Repository interface {
	GetProfile(ctx context.Context, userID string) (Profile, error)
	UpsertProfile(ctx context.Context, userID string, in ProfileInput) error
	UpdateLocation(ctx context.Context, userID string, lat, lon float64, city *string) error
	GetPreferences(ctx context.Context, userID string) (Preferences, error)
	UpsertPreferences(ctx context.Context, userID string, p Preferences) error
	UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error
	ListPhotos(ctx context.Context, userID string) ([]Photo, error)
	ListUserInterests(ctx context.Context, userID string) ([]Interest, error)
	ListInterests(ctx context.Context) ([]Interest, error)
	TouchActivity(ctx context.Context, userID string) error
	GetCards(ctx context.Context, userIDs []string) ([]StoredCard, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

// StoredCard is a profile row with its photos and interests aggregated by
// the database in the same query (no N+1).
type StoredCard struct {
	Profile   Profile
	Photos    []Photo
	Interests []Interest
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

const profileColumns = `
	p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.job_title, p.city,
	p.height_cm, p.relationship_goal, p.latitude, p.longitude,
	p.is_discoverable, p.show_distance, p.moderation_status`

func scanProfile(row pgx.Row, extra ...any) (Profile, error) {
	var p Profile
	var height *int16
	dest := []any{
		&p.UserID, &p.FirstName, &p.Birthdate, &p.Gender, &p.Bio, &p.JobTitle, &p.City,
		&height, &p.RelationshipGoal, &p.Latitude, &p.Longitude,
		&p.IsDiscoverable, &p.ShowDistance, &p.ModerationStatus,
	}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return Profile{}, err
	}
	if height != nil {
		h := int(*height)
		p.HeightCm = &h
	}
	return p, nil
}

func (r *PGRepository) GetProfile(ctx context.Context, userID string) (Profile, error) {
	p, err := scanProfile(r.pool.QueryRow(ctx, `SELECT `+profileColumns+` FROM profiles p WHERE p.user_id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrRepoNotFound
	}
	return p, err
}

func (r *PGRepository) UpsertProfile(ctx context.Context, userID string, in ProfileInput) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
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
		`, userID, in.FirstName, in.Birthdate, in.Gender, in.Bio, in.JobTitle, in.City, in.HeightCm, in.RelationshipGoal)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if len(in.InterestIDs) > 0 {
			_, err := tx.Exec(ctx, `
				INSERT INTO user_interests (user_id, interest_id)
				SELECT $1, unnest($2::smallint[])
				ON CONFLICT DO NOTHING
			`, userID, in.InterestIDs)
			if db.IsForeignKeyViolation(err) {
				return ErrRepoUnknownInterest
			}
			return err
		}
		return nil
	})
}

func (r *PGRepository) UpdateLocation(ctx context.Context, userID string, lat, lon float64, city *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles
		SET latitude = $2, longitude = $3, city = COALESCE($4, city), location_updated_at = NOW(), updated_at = NOW()
		WHERE user_id = $1
	`, userID, lat, lon, city)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	var minAge, maxAge, dist int16
	err := r.pool.QueryRow(ctx, `
		SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1
	`, userID).Scan(&p.InterestedIn, &minAge, &maxAge, &dist)
	if errors.Is(err, pgx.ErrNoRows) {
		return Preferences{}, ErrRepoNotFound
	}
	p.MinAge, p.MaxAge, p.MaxDistanceKm = int(minAge), int(maxAge), int(dist)
	return p, err
}

func (r *PGRepository) UpsertPreferences(ctx context.Context, userID string, p Preferences) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			interested_in = EXCLUDED.interested_in,
			min_age = EXCLUDED.min_age,
			max_age = EXCLUDED.max_age,
			max_distance_km = EXCLUDED.max_distance_km,
			updated_at = NOW()
	`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if db.IsForeignKeyViolation(err) {
		return ErrRepoNotFound
	}
	return err
}

func (r *PGRepository) UpdatePrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE profiles SET is_discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1
	`, userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) ListPhotos(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, position, storage_key, width, height FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) {
		var p Photo
		var pos int16
		err := row.Scan(&p.ID, &pos, &p.StorageKey, &p.Width, &p.Height)
		p.Position = int(pos)
		return p, err
	})
}

func (r *PGRepository) ListUserInterests(ctx context.Context, userID string) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = $1
		ORDER BY i.label
	`, userID)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func (r *PGRepository) ListInterests(ctx context.Context) ([]Interest, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, label FROM interests ORDER BY label`)
	if err != nil {
		return nil, err
	}
	return collectInterests(rows)
}

func collectInterests(rows pgx.Rows) ([]Interest, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Interest, error) {
		var i Interest
		var id int16
		err := row.Scan(&id, &i.Slug, &i.Label)
		i.ID = int(id)
		return i, err
	})
}

// TouchActivity is throttled in SQL to at most one write per 5 minutes.
func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

type jsonPhoto struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Width    int    `json:"w"`
	Height   int    `json:"h"`
	Position int    `json:"pos"`
}

func (r *PGRepository) GetCards(ctx context.Context, userIDs []string) ([]StoredCard, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+profileColumns+`,
			COALESCE((
				SELECT json_agg(json_build_object('id', ph.id, 'key', ph.storage_key, 'w', ph.width, 'h', ph.height, 'pos', ph.position) ORDER BY ph.position)
				FROM photos ph WHERE ph.user_id = p.user_id
			), '[]'::json),
			COALESCE((
				SELECT json_agg(json_build_object('id', i.id, 'slug', i.slug, 'label', i.label) ORDER BY i.label)
				FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
				WHERE ui.user_id = p.user_id
			), '[]'::json)
		FROM profiles p
		WHERE p.user_id = ANY($1::uuid[])
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []StoredCard
	for rows.Next() {
		var photosJSON, interestsJSON []byte
		p, err := scanProfile(rows, &photosJSON, &interestsJSON)
		if err != nil {
			return nil, err
		}
		var jp []jsonPhoto
		if err := json.Unmarshal(photosJSON, &jp); err != nil {
			return nil, err
		}
		card := StoredCard{Profile: p}
		for _, ph := range jp {
			card.Photos = append(card.Photos, Photo{ID: ph.ID, StorageKey: ph.Key, Width: ph.Width, Height: ph.Height, Position: ph.Position})
		}
		if err := json.Unmarshal(interestsJSON, &card.Interests); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

// CanView: a profile is visible when nobody blocked anybody and either the two
// people are matched, or the target is an active, discoverable profile.
func (r *PGRepository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles t
			WHERE t.user_id = $2
			  AND t.moderation_status <> 'suspended'
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND (
				EXISTS (SELECT 1 FROM matches m WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid))
				OR (t.is_discoverable AND t.moderation_status = 'active')
			  )
		)
	`, viewerID, targetID).Scan(&ok)
	return ok, err
}
