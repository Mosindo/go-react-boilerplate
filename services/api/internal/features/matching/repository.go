package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("profile incomplete")
	ErrNotEligible       = errors.New("user not available")
	ErrMatchNotFound     = errors.New("match not found")
)

// eligibleQuery builds the single source of truth for "who may user $1 see/like".
// Discovery lists these rows; swiping verifies membership with the same rules so a
// crafted request cannot like someone the product would never have shown.
//
// Recommendation extension point: the score columns (liked_me, shared) and the ORDER BY
// in discoverSQL are the only place ranking is defined.
func eligibleQuery(selectList, extraWhere, tail string) string {
	return `
WITH me AS (
  SELECT p.user_id, p.gender, p.latitude AS lat, p.longitude AS lon,
         EXTRACT(YEAR FROM age(CURRENT_DATE, p.birthdate))::int AS age,
         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
  FROM profiles p
  JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
), cand AS (
  SELECT c.user_id, c.first_name, c.gender, c.bio, c.city, c.show_distance,
         EXTRACT(YEAR FROM age(CURRENT_DATE, c.birthdate))::int AS age,
         ` + geo.SQLDistanceKm("me.lat", "me.lon", "c.latitude", "c.longitude") + ` AS dist_km,
         cp.interested_in AS their_interested_in, cp.min_age AS their_min, cp.max_age AS their_max,
         cp.max_distance_km AS their_distance, me.age AS my_age, me.min_age AS my_min, me.max_age AS my_max,
         me.max_distance_km AS my_distance, me.gender AS my_gender
  FROM me
  JOIN profiles c ON c.user_id <> me.user_id
                 AND c.is_discoverable
                 AND c.gender = ANY (me.interested_in)
                 AND c.latitude BETWEEN me.lat - me.max_distance_km / 110.0 AND me.lat + me.max_distance_km / 110.0
  JOIN users cu ON cu.id = c.user_id AND cu.status = 'active'
  JOIN preferences cp ON cp.user_id = c.user_id
  WHERE EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
    AND NOT EXISTS (
          SELECT 1 FROM blocks b
          WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
             OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
)
SELECT ` + selectList + `
FROM cand
WHERE cand.my_gender = ANY (cand.their_interested_in)
  AND cand.age BETWEEN cand.my_min AND cand.my_max
  AND cand.my_age BETWEEN cand.their_min AND cand.their_max
  AND cand.dist_km <= LEAST(cand.my_distance, cand.their_distance)
  ` + extraWhere + `
` + tail
}

const discoverSelect = `cand.user_id, cand.first_name, cand.age, cand.gender, cand.bio, cand.city,
  CASE WHEN cand.show_distance THEN cand.dist_km END,
  EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = cand.user_id AND s.to_user_id = $1 AND s.action = 'like') AS liked_me,
  (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
    WHERE a.user_id = $1 AND b.user_id = cand.user_id) AS shared`

var discoverSQL = eligibleQuery(
	discoverSelect,
	`AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = $1 AND s.to_user_id = cand.user_id)`,
	`ORDER BY liked_me DESC, shared DESC, cand.dist_km ASC, cand.user_id
LIMIT $2`,
)

var eligibleOneSQL = `SELECT EXISTS (` + eligibleQuery(`1`, `AND cand.user_id = $2`, ``) + `)`

type Repository interface {
	Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error)
	ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]Match, error)
	GetMatch(ctx context.Context, userID, matchID string) (Match, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

// SwipeOutcome is the persistence result of a swipe.
type SwipeOutcome struct {
	Action        string
	AlreadySwiped bool
	MatchID       string // set when the pair is matched (new or pre-existing)
	NewMatch      bool
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	var hasProfile bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles p JOIN preferences pr USING (user_id)
		               WHERE p.user_id = $1 AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))
	`, userID).Scan(&hasProfile); err != nil {
		return nil, err
	}
	if !hasProfile {
		return nil, ErrProfileIncomplete
	}

	rows, err := r.pool.Query(ctx, discoverSQL, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]profiles.PublicProfile, 0, limit)
	ids := make([]string, 0, limit)
	for rows.Next() {
		var p profiles.PublicProfile
		var dist *float64
		var likedMe bool
		var shared int
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &dist, &likedMe, &shared); err != nil {
			return nil, err
		}
		if dist != nil {
			d := geo.DisplayDistance(*dist)
			p.DistanceKm = &d
		}
		out = append(out, p)
		ids = append(ids, p.UserID)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	rows.Close()

	// Two batched queries for the whole page: no N+1.
	interests, err := profiles.LoadInterests(ctx, r.pool, ids)
	if err != nil {
		return nil, err
	}
	photos, err := profiles.LoadPhotos(ctx, r.pool, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Interests = interests[out[i].UserID]
		out[i].Photos = photos[out[i].UserID]
	}
	return out, nil
}

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise swipes on the same pair. Without this, two users liking each other at
	// the same instant could each miss the other's uncommitted like and no match
	// would ever be created.
	a, b := userID, targetID
	if b < a {
		a, b = b, a
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, a+":"+b); err != nil {
		return SwipeOutcome{}, err
	}

	outcome := SwipeOutcome{Action: action}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		outcome.Action = existing
		outcome.AlreadySwiped = true
		matchID, err := findMatch(ctx, tx, a, b)
		if err != nil {
			return SwipeOutcome{}, err
		}
		outcome.MatchID = matchID
		return outcome, tx.Commit(ctx)
	case !errors.Is(err, pgx.ErrNoRows):
		return SwipeOutcome{}, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, eligibleOneSQL, userID, targetID).Scan(&eligible); err != nil {
		return SwipeOutcome{}, err
	}
	if !eligible {
		// Distinguish "my profile is not ready" from "target unavailable".
		var ready bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM profiles p JOIN preferences pr USING (user_id) WHERE p.user_id = $1)
		`, userID).Scan(&ready); err != nil {
			return SwipeOutcome{}, err
		}
		if !ready {
			return SwipeOutcome{}, ErrProfileIncomplete
		}
		return SwipeOutcome{}, ErrNotEligible
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
	`, userID, targetID, action); err != nil {
		return SwipeOutcome{}, err
	}

	if action == "like" {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')
		`, targetID, userID).Scan(&reciprocal); err != nil {
			return SwipeOutcome{}, err
		}
		if reciprocal {
			matchID, err := createMatch(ctx, tx, a, b, userID, targetID)
			if err != nil {
				return SwipeOutcome{}, err
			}
			outcome.MatchID = matchID
			outcome.NewMatch = true
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeOutcome{}, err
	}
	return outcome, nil
}

func findMatch(ctx context.Context, tx pgx.Tx, a, b string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_a_id = $1 AND user_b_id = $2`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func createMatch(ctx context.Context, tx pgx.Tx, a, b, swiper, target string) (string, error) {
	var matchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id) VALUES ($1, $2)
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = EXCLUDED.user_a_id
		RETURNING id
	`, a, b).Scan(&matchID); err != nil {
		return "", err
	}
	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id
	`, matchID).Scan(&conversationID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING
	`, conversationID, a, b); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_user_id, conversation_id)
		VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)
	`, swiper, target, conversationID); err != nil {
		return "", err
	}
	return matchID, nil
}

const matchSelect = `
	SELECT m.id, c.id, m.created_at, o.user_id, o.first_name,
	       EXTRACT(YEAR FROM age(CURRENT_DATE, o.birthdate))::int,
	       ph.id, ph.position, ph.width, ph.height, left(md5(ph.storage_key), 8),
	       EXISTS (SELECT 1 FROM messages msg WHERE msg.conversation_id = c.id)
	FROM matches m
	JOIN conversations c ON c.match_id = m.id
	JOIN profiles o ON o.user_id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
	LEFT JOIN photos ph ON ph.user_id = o.user_id AND ph.position = 0
	WHERE (m.user_a_id = $1 OR m.user_b_id = $1)`

func scanMatch(row pgx.Row) (Match, error) {
	var m Match
	var photoID *string
	var pos, w, h *int
	var version *string
	if err := row.Scan(&m.ID, &m.ConversationID, &m.CreatedAt, &m.User.UserID, &m.User.FirstName, &m.User.Age,
		&photoID, &pos, &w, &h, &version, &m.HasMessages); err != nil {
		return Match{}, err
	}
	if photoID != nil {
		m.User.Photo = &profiles.PhotoRef{ID: *photoID, Position: *pos, Width: *w, Height: *h, Version: *version}
	}
	return m, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]Match, error) {
	// Keyset pagination on created_at; id breaks no ties on purpose (timestamps are microsecond-precise).
	query := matchSelect + `
	  AND ($2::timestamptz IS NULL OR m.created_at < $2)
	ORDER BY m.created_at DESC
	LIMIT $3`
	rows, err := r.pool.Query(ctx, query, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Match, 0, limit)
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (Match, error) {
	m, err := scanMatch(r.pool.QueryRow(ctx, matchSelect+` AND m.id = $2`, userID, matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Match{}, ErrMatchNotFound
		}
		return Match{}, err
	}
	return m, nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.pool.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING user_a_id, user_b_id
	`, userID, matchID).Scan(&a, &b)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrMatchNotFound
		}
		return "", err
	}
	if a == userID {
		return b, nil
	}
	return a, nil
}
