package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("profile not visible")

// Recommender returns ranked candidates for a viewer. The PostgreSQL
// implementation below is the default; swap it to plug in a smarter ranker.
type Recommender interface {
	Candidates(ctx context.Context, viewerID string, limit int) ([]Candidate, error)
}

type Repository interface {
	Recommender
	Onboarded(ctx context.Context, viewerID string) (bool, error)
	TouchActive(ctx context.Context, userID string) error
	Visible(ctx context.Context, viewerID, targetID string) (Candidate, error)
	InterestsFor(ctx context.Context, viewerID string, userIDs []string) (map[string][]CardInterest, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Onboarded(ctx context.Context, viewerID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, viewerID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1`, userID)
	return err
}

// Haversine distance in km between the viewer (me.lat/lng) and candidate c.
// NULL when either side has no coarse location.
const distanceSQL = `
	CASE WHEN me.lat IS NULL OR c.latitude IS NULL THEN NULL
	ELSE 6371.0 * 2 * asin(least(1.0, sqrt(
		power(sin(radians(c.latitude - me.lat) / 2), 2) +
		cos(radians(me.lat)) * cos(radians(c.latitude)) *
		power(sin(radians(c.longitude - me.lng) / 2), 2)
	))) END`

const meCTE = `
	WITH me AS (
		SELECT p.gender, p.latitude AS lat, p.longitude AS lng,
		       date_part('year', age(p.birth_date))::int AS age,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	)`

// Candidates applies, in SQL and in one query: mutual preferences (gender, age,
// distance), active/discoverable/onboarded accounts, no previous swipe, no block
// in either direction. Ranking: shared interests and people who already liked
// the viewer first, then recent activity and proximity.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, meCTE+`
		SELECT c.user_id, c.first_name, c.birth_date, c.gender, c.bio, c.city, d.km, cp.show_distance
		FROM profiles c
		CROSS JOIN me
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users u ON u.id = c.user_id AND u.status = 'active'
		CROSS JOIN LATERAL (SELECT date_part('year', age(c.birth_date))::int AS age) ca
		CROSS JOIN LATERAL (SELECT `+distanceSQL+` AS km) d
		WHERE c.user_id <> $1
		  AND cp.discoverable
		  AND c.gender = ANY (me.interested_in)
		  AND ca.age BETWEEN me.min_age AND me.max_age
		  AND me.gender = ANY (cp.interested_in)
		  AND me.age BETWEEN cp.min_age AND cp.max_age
		  AND (d.km IS NULL OR (d.km <= me.max_distance_km AND d.km <= cp.max_distance_km))
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = c.user_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = c.user_id)
		       OR (b.blocker_id = c.user_id AND b.blocked_id = $1)
		  )
		ORDER BY (
		    3 * (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
		         WHERE a.user_id = $1 AND b.user_id = c.user_id)
		  + CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = $1 AND s.action = 'like')
		         THEN 5 ELSE 0 END
		  - LEAST(EXTRACT(EPOCH FROM (NOW() - u.last_active_at)) / 86400.0, 30) / 3
		  - COALESCE(d.km, 0) / 50.0
		) DESC, c.user_id
		LIMIT $2
	`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidates(rows)
}

// Visible loads one profile if the viewer may see it: not blocked either way,
// active, onboarded, and either discoverable, matched, or already interested in the viewer.
func (r *PGRepository) Visible(ctx context.Context, viewerID, targetID string) (Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT latitude AS lat, longitude AS lng FROM profiles WHERE user_id = $1
			UNION ALL SELECT NULL, NULL WHERE NOT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
			LIMIT 1
		)
		SELECT c.user_id, c.first_name, c.birth_date, c.gender, c.bio, c.city, d.km, cp.show_distance
		FROM profiles c
		CROSS JOIN me
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users u ON u.id = c.user_id AND u.status = 'active'
		CROSS JOIN LATERAL (SELECT `+distanceSQL+` AS km) d
		WHERE c.user_id = $2 AND c.user_id <> $1
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = c.user_id)
		       OR (b.blocker_id = c.user_id AND b.blocked_id = $1)
		  )
		  AND (
		    cp.discoverable
		    OR EXISTS (SELECT 1 FROM matches m WHERE m.unmatched_at IS NULL
		               AND m.user_a = LEAST($1::uuid, c.user_id) AND m.user_b = GREATEST($1::uuid, c.user_id))
		    OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = $1 AND s.action = 'like')
		  )
	`, viewerID, targetID)
	if err != nil {
		return Candidate{}, err
	}
	defer rows.Close()
	list, err := scanCandidates(rows)
	if err != nil {
		return Candidate{}, err
	}
	if len(list) == 0 {
		return Candidate{}, ErrRepositoryNotFound
	}
	return list[0], nil
}

func scanCandidates(rows pgx.Rows) ([]Candidate, error) {
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.BirthDate, &c.Gender, &c.Bio, &c.City, &c.DistanceKm, &c.ShowDistance); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// InterestsFor loads the interests of many users at once, flagging those the viewer shares.
func (r *PGRepository) InterestsFor(ctx context.Context, viewerID string, userIDs []string) (map[string][]CardInterest, error) {
	out := make(map[string][]CardInterest, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label,
		       i.id IN (SELECT interest_id FROM user_interests WHERE user_id = $1) AS shared
		FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($2::uuid[])
		ORDER BY shared DESC, i.label
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var ci CardInterest
		if err := rows.Scan(&uid, &ci.Slug, &ci.Label, &ci.Shared); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], ci)
	}
	return out, rows.Err()
}
