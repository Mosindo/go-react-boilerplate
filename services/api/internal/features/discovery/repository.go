package discovery

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotEligible = errors.New("profile not available")
	ErrIncomplete  = errors.New("own profile incomplete")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// candidateSQL is the single place that defines who may be shown to whom, and
// in which order. It is used both to build the discovery deck ($3 NULL) and to
// authorise a like/pass on one given profile ($3 = target). To evolve the
// recommendation algorithm, change the ORDER BY (or the scoring columns) here.
//
// Hard filters (all must hold, symmetric where it makes sense):
//   - candidate is discoverable, has a photo and a location
//   - gender / age / distance preferences match in both directions
//   - no block in either direction
//   - the viewer has not already liked or passed the candidate
//
// Ranking: people who already liked the viewer, then shared interests, then
// proximity, then recent activity.
const candidateSQL = `
WITH me AS (
	SELECT p.user_id, p.gender, p.birth_date, p.latitude AS lat, p.longitude AS lng,
	       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1 AND p.latitude IS NOT NULL
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
)
SELECT c.user_id, c.first_name, c.birth_date, c.gender, c.bio, c.city, c.show_distance, d.km,
       (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = c.user_id
         WHERE a.user_id = me.user_id) AS shared,
       EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = c.user_id AND s.to_user_id = me.user_id AND s.action = 'like') AS liked_me
FROM me
JOIN profiles c ON c.user_id <> me.user_id AND c.discoverable AND c.latitude IS NOT NULL
JOIN preferences cp ON cp.user_id = c.user_id
JOIN users u ON u.id = c.user_id
CROSS JOIN LATERAL (
	SELECT 6371 * 2 * asin(sqrt(least(1, power(sin(radians(c.latitude - me.lat) / 2), 2) +
	       cos(radians(me.lat)) * cos(radians(c.latitude)) * power(sin(radians(c.longitude - me.lng) / 2), 2)))) AS km
) d
WHERE ($3::uuid IS NULL OR c.user_id = $3)
  AND c.latitude BETWEEN me.lat - me.max_distance_km / 111.0 AND me.lat + me.max_distance_km / 111.0
  AND c.gender = ANY(me.interested_in)
  AND me.gender = ANY(cp.interested_in)
  AND date_part('year', age(c.birth_date)) BETWEEN me.min_age AND me.max_age
  AND date_part('year', age(me.birth_date)) BETWEEN cp.min_age AND cp.max_age
  AND d.km <= LEAST(me.max_distance_km, cp.max_distance_km)
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
                                            OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
ORDER BY liked_me DESC, shared DESC, d.km ASC, u.last_active_at DESC, c.user_id
LIMIT $2`

type candidateRow struct {
	card     Card
	showDist bool
	km       float64
}

func (r *Repository) candidates(ctx context.Context, q queryer, viewerID string, limit int, target *string) ([]Card, error) {
	rows, err := q.Query(ctx, candidateSQL, viewerID, limit, target)
	if err != nil {
		return nil, err
	}
	var list []candidateRow
	now := time.Now()
	for rows.Next() {
		var cr candidateRow
		var birth time.Time
		var liked bool
		if err := rows.Scan(&cr.card.UserID, &cr.card.FirstName, &birth, &cr.card.Gender, &cr.card.Bio, &cr.card.City,
			&cr.showDist, &cr.km, &cr.card.SharedInterests, &liked); err != nil {
			rows.Close()
			return nil, err
		}
		cr.card.Age = profiles.AgeOn(birth, now)
		if cr.showDist {
			b := DistanceBucket(cr.km)
			cr.card.DistanceKm = &b
		}
		list = append(list, cr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	cards := make([]Card, len(list))
	ids := make([]string, len(list))
	for i, cr := range list {
		cards[i] = cr.card
		ids[i] = cr.card.UserID
	}
	if err := r.attach(ctx, q, cards, ids); err != nil {
		return nil, err
	}
	return cards, nil
}

// attach loads interests and photos for all cards with two queries (no N+1).
func (r *Repository) attach(ctx context.Context, q queryer, cards []Card, ids []string) error {
	if len(cards) == 0 {
		return nil
	}
	idx := map[string]*Card{}
	for i := range cards {
		cards[i].Interests = []profiles.Interest{}
		cards[i].Photos = []profiles.PhotoRef{}
		idx[cards[i].UserID] = &cards[i]
	}
	rows, err := q.Query(ctx, `
		SELECT ui.user_id, i.id, i.slug, i.label FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.label`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var uid string
		var in profiles.Interest
		if err := rows.Scan(&uid, &in.ID, &in.Slug, &in.Label); err != nil {
			rows.Close()
			return err
		}
		idx[uid].Interests = append(idx[uid].Interests, in)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	rows, err = q.Query(ctx, `SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY position`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid string
		var p profiles.PhotoRef
		if err := rows.Scan(&uid, &p.ID, &p.Position); err != nil {
			return err
		}
		p.URL = "/photos/" + p.ID
		idx[uid].Photos = append(idx[uid].Photos, p)
	}
	return rows.Err()
}

func (r *Repository) Deck(ctx context.Context, viewerID string, limit int) ([]Card, error) {
	return r.candidates(ctx, r.pool, viewerID, limit, nil)
}

// Swipe records a decision atomically. It is idempotent: repeating the same
// action returns the original outcome; contradicting a past decision fails.
func (r *Repository) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise both directions of a pair so that two simultaneous likes
	// cannot each miss the other and leave a mutual like without a match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))`, viewerID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, viewerID, targetID).Scan(&existing)
	if err == nil {
		if existing != action {
			return SwipeResult{}, ErrAlreadyDecided
		}
		res := SwipeResult{Action: action}
		if action == "like" {
			res.MatchID, res.Matched = r.findMatch(ctx, tx, viewerID, targetID)
		}
		return res, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeResult{}, err
	}

	var haveSelf bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles p JOIN preferences pr USING (user_id)
		  WHERE p.user_id = $1 AND p.latitude IS NOT NULL AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1))`, viewerID).Scan(&haveSelf); err != nil {
		return SwipeResult{}, err
	}
	if !haveSelf {
		return SwipeResult{}, ErrIncomplete
	}
	elig, err := r.candidates(ctx, tx, viewerID, 1, &targetID)
	if err != nil {
		return SwipeResult{}, err
	}
	if len(elig) == 0 {
		return SwipeResult{}, ErrNotEligible
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, viewerID, targetID, action); err != nil {
		return SwipeResult{}, err
	}
	res := SwipeResult{Action: action}
	if action == "like" {
		var reverse bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, targetID, viewerID).Scan(&reverse); err != nil {
			return SwipeResult{}, err
		}
		if reverse {
			err = tx.QueryRow(ctx, `
				INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = EXCLUDED.user_a_id
				RETURNING id`, viewerID, targetID).Scan(&res.MatchID)
			if err != nil {
				return SwipeResult{}, err
			}
			res.Matched = true
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return res, nil
}

var ErrAlreadyDecided = errors.New("already decided differently")

func (r *Repository) findMatch(ctx context.Context, q queryer, a, b string) (string, bool) {
	var id string
	err := q.QueryRow(ctx, `SELECT id FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)`, a, b).Scan(&id)
	return id, err == nil
}

// Profile returns the public card of one user if the viewer may see it: not
// blocked in either direction and either matched or currently discoverable.
func (r *Repository) Profile(ctx context.Context, viewerID, targetID string) (Card, error) {
	var card Card
	var birth time.Time
	var showDist bool
	var lat, lng, vlat, vlng *float64
	err := r.pool.QueryRow(ctx, `
		SELECT p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city, p.show_distance, p.latitude, p.longitude,
		       (SELECT latitude FROM profiles WHERE user_id = $1), (SELECT longitude FROM profiles WHERE user_id = $1)
		FROM profiles p
		WHERE p.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (p.discoverable OR EXISTS (SELECT 1 FROM matches m
		        WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)))`, viewerID, targetID).
		Scan(&card.UserID, &card.FirstName, &birth, &card.Gender, &card.Bio, &card.City, &showDist, &lat, &lng, &vlat, &vlng)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Card{}, ErrNotEligible
		}
		return Card{}, err
	}
	card.Age = profiles.AgeOn(birth, time.Now())
	if showDist && lat != nil && vlat != nil {
		b := DistanceBucket(haversine(*vlat, *vlng, *lat, *lng))
		card.DistanceKm = &b
	}
	cards := []Card{card}
	if err := r.attach(ctx, r.pool, cards, []string{targetID}); err != nil {
		return Card{}, err
	}
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = $2 WHERE a.user_id = $1`,
		viewerID, targetID).Scan(&cards[0].SharedInterests)
	return cards[0], err
}
