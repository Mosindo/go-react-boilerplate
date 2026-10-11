package users

import (
	"context"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepoNotFound = errors.New("users: not found")

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Interests(ctx context.Context) ([]Interest, error) {
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

func (r *Repository) GetStored(ctx context.Context, userID string) (storedProfile, error) {
	var p storedProfile
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, first_name, birth_date, EXTRACT(YEAR FROM age(birth_date))::int, gender, bio, city,
		       discoverable, show_distance, latitude IS NOT NULL
		FROM profiles WHERE user_id = $1`, userID).
		Scan(&p.UserID, &p.FirstName, &p.BirthDate, &p.Age, &p.Gender, &p.Bio, &p.City, &p.Discoverable, &p.ShowDistance, &p.HasLocation)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, errRepoNotFound
	}
	return p, err
}

// Upsert creates or updates the profile, replaces interests and ensures default preferences exist.
func (r *Repository) Upsert(ctx context.Context, userID string, in ProfileInput, birthDate string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, birth_date, gender, bio, city)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE
		SET first_name = EXCLUDED.first_name, gender = EXCLUDED.gender, bio = EXCLUDED.bio,
		    city = EXCLUDED.city, updated_at = NOW()`,
		userID, in.FirstName, birthDate, in.Gender, in.Bio, in.City); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_interests WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if len(in.InterestIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, i FROM unnest($2::int[]) AS i`, userID, in.InterestIDs); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO preferences (user_id, interested_in, min_age, max_age, max_distance_km)
		VALUES ($1, ARRAY['man','woman','non_binary'], 18, 99, 50)
		ON CONFLICT (user_id) DO NOTHING`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	var p Preferences
	var minAge, maxAge int16
	err := r.pool.QueryRow(ctx, `SELECT interested_in, min_age, max_age, max_distance_km FROM preferences WHERE user_id = $1`, userID).
		Scan(&p.InterestedIn, &minAge, &maxAge, &p.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, errRepoNotFound
	}
	p.MinAge, p.MaxAge = int(minAge), int(maxAge)
	return p, err
}

func (r *Repository) SetPreferences(ctx context.Context, userID string, p Preferences) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE preferences SET interested_in = $2, min_age = $3, max_age = $4, max_distance_km = $5, updated_at = NOW()
		WHERE user_id = $1`, userID, p.InterestedIn, p.MinAge, p.MaxAge, p.MaxDistanceKm)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return nil
}

func (r *Repository) SetPrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE profiles SET discoverable = $2, show_distance = $3, updated_at = NOW() WHERE user_id = $1`,
		userID, discoverable, showDistance)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return nil
}

func (r *Repository) SetLocation(ctx context.Context, userID string, lat, lng *float64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE profiles SET latitude = $2, longitude = $3, updated_at = NOW() WHERE user_id = $1`, userID, lat, lng)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errRepoNotFound
	}
	return nil
}

func (r *Repository) loadInterests(ctx context.Context, ids []string) (map[string][]Interest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Interest{}
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

type photoRef struct {
	ID       string
	Position int
}

func (r *Repository) loadPhotos(ctx context.Context, ids []string) (map[string][]photoRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]photoRef{}
	for rows.Next() {
		var uid string
		var p photoRef
		var pos int16
		if err := rows.Scan(&uid, &p.ID, &pos); err != nil {
			return nil, err
		}
		p.Position = int(pos)
		out[uid] = append(out[uid], p)
	}
	return out, rows.Err()
}

type publicRow struct {
	PublicProfile
	distance *float64
}

// PublicRows returns profiles for ids (same order, missing ids skipped) with distance from the viewer.
func (r *Repository) PublicRows(ctx context.Context, viewerID string, ids []string) ([]publicRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	dist := db.HaversineKM("me.latitude", "me.longitude", "p.latitude", "p.longitude")
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, EXTRACT(YEAR FROM age(p.birth_date))::int, p.gender, p.bio, p.city,
		       CASE WHEN p.show_distance AND me.latitude IS NOT NULL AND p.latitude IS NOT NULL THEN `+dist+` END,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = p.user_id AND s.to_user_id = $1 AND s.action = 'like')
		FROM profiles p
		LEFT JOIN profiles me ON me.user_id = $1
		WHERE p.user_id = ANY($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := make(map[string]publicRow, len(ids))
	for rows.Next() {
		var row publicRow
		if err := rows.Scan(&row.UserID, &row.FirstName, &row.Age, &row.Gender, &row.Bio, &row.City, &row.distance, &row.LikedYou); err != nil {
			return nil, err
		}
		byID[row.UserID] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]publicRow, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

// CanView reports whether viewer may see target's profile: self, or complete + not blocked +
// (discoverable, already matched, or has liked the viewer).
func (r *Repository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT $1::uuid = $2::uuid OR EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
			  AND (p.discoverable
			       OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))
			       OR EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = $2 AND s.to_user_id = $1 AND s.action = 'like'))
		)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

func (r *Repository) InterestMaps(ctx context.Context, ids []string) (map[string][]Interest, map[string][]photoRef, error) {
	in, err := r.loadInterests(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	ph, err := r.loadPhotos(ctx, ids)
	return in, ph, err
}

// fkConstraint returns the violated foreign-key constraint name, or "" for any other error.
func fkConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return pgErr.ConstraintName
	}
	return ""
}
