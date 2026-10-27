package matching

import (
	"context"
	"errors"
	"math"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoTargetUnavailable = errors.New("matching: target unavailable")
	ErrRepoAlreadySwiped     = errors.New("matching: already swiped")
	ErrRepoNotFound          = errors.New("matching: not found")
	ErrRepoProfileMissing    = errors.New("matching: profile missing")
)

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error)
	ListMatches(ctx context.Context, userID string, limit int, before *Cursor) ([]Match, *Cursor, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	GetPublicProfile(ctx context.Context, viewerID, targetID string) (profiles.PublicProfile, error)
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// haversineKM is the great-circle distance between viewer (me) and candidate (c),
// NULL when either side has no stored location.
const haversineKM = `
	CASE WHEN me.lat IS NULL OR c.latitude IS NULL THEN NULL ELSE
	  2 * 6371 * asin(least(1, sqrt(
	    power(sin(radians(c.latitude - me.lat) / 2), 2) +
	    cos(radians(me.lat)) * cos(radians(c.latitude)) *
	    power(sin(radians(c.longitude - me.lng) / 2), 2)
	  )))
	END`

// meCTE loads the viewer's own attributes once.
const meCTE = `
	WITH me AS (
	  SELECT p.user_id, p.gender, p.latitude AS lat, p.longitude AS lng,
	         EXTRACT(YEAR FROM age(p.birth_date))::int AS age,
	         pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
	  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	  WHERE p.user_id = $1
	)`

// Ranking is isolated here so the recommendation strategy can evolve without
// touching eligibility rules: people who already liked the viewer first (higher
// chance of a mutual match), then most recently active, then nearest.
const candidateOrder = `
	ORDER BY (EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like')) DESC,
	         u.last_active_at DESC,
	         dist ASC NULLS LAST,
	         c.user_id`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	rows, err := r.db.Query(ctx, meCTE+`
	SELECT c.user_id, c.first_name, EXTRACT(YEAR FROM age(c.birth_date))::int, c.gender, c.bio, c.city,
	       c.show_distance, `+haversineKM+` AS dist
	FROM me
	JOIN profiles c ON c.user_id <> me.user_id
	JOIN preferences cp ON cp.user_id = c.user_id
	JOIN users u ON u.id = c.user_id
	WHERE c.discoverable
	  AND c.gender = ANY (me.interested_in)
	  AND me.gender = ANY (cp.interested_in)
	  AND EXTRACT(YEAR FROM age(c.birth_date)) BETWEEN me.age_min AND me.age_max
	  AND me.age BETWEEN cp.age_min AND cp.age_max
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	                  WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	                     OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND (`+haversineKM+` IS NULL OR `+haversineKM+` <= least(me.max_distance_km, cp.max_distance_km))
	`+candidateOrder+`
	LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	out, err := scanPublic(rows)
	if err != nil {
		return nil, err
	}
	return r.decorate(ctx, out)
}

// scanPublic reads rows shaped as (id, name, age, gender, bio, city, show_distance, dist).
func scanPublic(rows pgx.Rows) ([]profiles.PublicProfile, error) {
	defer rows.Close()
	out := []profiles.PublicProfile{}
	for rows.Next() {
		var p profiles.PublicProfile
		var show bool
		var dist *float64
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.Gender, &p.Bio, &p.City, &show, &dist); err != nil {
			return nil, err
		}
		if show && dist != nil {
			p.DistanceKM = approxDistance(*dist)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// approxDistance rounds up to the next 5 km so exact positions can never be
// recovered by trilateration across repeated requests.
func approxDistance(km float64) *int {
	d := int(math.Ceil(km/5)) * 5
	if d < 5 {
		d = 5
	}
	return &d
}

// decorate attaches interests and photos for the whole page with two queries.
func (r *PGRepository) decorate(ctx context.Context, list []profiles.PublicProfile) ([]profiles.PublicProfile, error) {
	if len(list) == 0 {
		return list, nil
	}
	ids := make([]string, len(list))
	for i, p := range list {
		ids[i] = p.UserID
	}
	interests, err := profiles.LoadInterests(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}
	photos, err := profiles.LoadPhotos(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Interests = interests[list[i].UserID]
		list[i].Photos = photos[list[i].UserID]
	}
	return list, nil
}

func (r *PGRepository) GetPublicProfile(ctx context.Context, viewerID, targetID string) (profiles.PublicProfile, error) {
	// A profile is viewable when it is a discoverable, unblocked profile or a match.
	rows, err := r.db.Query(ctx, meCTE+`
	SELECT c.user_id, c.first_name, EXTRACT(YEAR FROM age(c.birth_date))::int, c.gender, c.bio, c.city,
	       c.show_distance, `+haversineKM+` AS dist
	FROM me
	JOIN profiles c ON c.user_id = $2 AND c.user_id <> me.user_id
	WHERE NOT EXISTS (SELECT 1 FROM blocks b
	                  WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	                     OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND (c.discoverable OR EXISTS (
	        SELECT 1 FROM matches m
	        WHERE m.user_a = least(me.user_id, c.user_id) AND m.user_b = greatest(me.user_id, c.user_id)))`,
		viewerID, targetID)
	if err != nil {
		return profiles.PublicProfile{}, err
	}
	list, err := scanPublic(rows)
	if err != nil {
		return profiles.PublicProfile{}, err
	}
	if len(list) == 0 {
		return profiles.PublicProfile{}, ErrRepoNotFound
	}
	list, err = r.decorate(ctx, list)
	if err != nil {
		return profiles.PublicProfile{}, err
	}
	return list[0], nil
}

func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Serialize the two directions of a pair: without it, two simultaneous likes
	// could each miss the other's uncommitted row and no match would be created.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(least($1::text, $2::text) || greatest($1::text, $2::text), 0))`, swiperID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var swiperHasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = $1)`, swiperID).Scan(&swiperHasProfile); err != nil {
		return SwipeResult{}, err
	}
	if !swiperHasProfile {
		return SwipeResult{}, ErrRepoProfileMissing
	}

	var targetOK bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2 AND p.user_id <> $1
		    AND NOT EXISTS (SELECT 1 FROM blocks b
		                    WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		                       OR (b.blocker_id = p.user_id AND b.blocked_id = $1)))`,
		swiperID, targetID).Scan(&targetOK); err != nil {
		return SwipeResult{}, err
	}
	if !targetOK {
		return SwipeResult{}, ErrRepoTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING`, swiperID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrRepoAlreadySwiped
	}

	result := SwipeResult{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')`,
			swiperID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			if err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b)
				VALUES (least($1::text, $2::text)::uuid, greatest($1::text, $2::text)::uuid)
				ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = matches.user_a
				RETURNING id`, swiperID, targetID).Scan(&result.MatchID); err != nil {
				return SwipeResult{}, err
			}
			if err := tx.QueryRow(ctx, `
				INSERT INTO conversations (match_id) VALUES ($1)
				ON CONFLICT (match_id) DO UPDATE SET match_id = conversations.match_id
				RETURNING id`, result.MatchID).Scan(&result.ConversationID); err != nil {
				return SwipeResult{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO conversation_participants (conversation_id, user_id)
				VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, result.ConversationID, swiperID, targetID); err != nil {
				return SwipeResult{}, err
			}
			result.Matched = true
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return result, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit int, before *Cursor) ([]Match, *Cursor, error) {
	var beforeTime *time.Time
	var beforeID *string
	if before != nil {
		beforeTime, beforeID = &before.CreatedAt, &before.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.created_at, cv.id,
		       CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS other_id
		FROM matches m
		JOIN conversations cv ON cv.match_id = m.id
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($3::timestamptz IS NULL OR (m.created_at, m.id) < ($3, $4::uuid))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2`, userID, limit+1, beforeTime, beforeID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var matches []Match
	var otherIDs []string
	for rows.Next() {
		var m Match
		var other string
		if err := rows.Scan(&m.ID, &m.CreatedAt, &m.ConversationID, &other); err != nil {
			return nil, nil, err
		}
		m.User.UserID = other
		matches = append(matches, m)
		otherIDs = append(otherIDs, other)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	rows.Close()

	var next *Cursor
	if len(matches) > limit {
		matches = matches[:limit]
		otherIDs = otherIDs[:limit]
		last := matches[len(matches)-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if len(matches) == 0 {
		return []Match{}, nil, nil
	}

	cards, err := r.cards(ctx, userID, otherIDs)
	if err != nil {
		return nil, nil, err
	}
	for i := range matches {
		matches[i].User = cards[matches[i].User.UserID]
	}
	return matches, next, nil
}

// cards loads public profiles for matched users (batched, no N+1).
func (r *PGRepository) cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.PublicProfile, error) {
	rows, err := r.db.Query(ctx, meCTE+`
	SELECT c.user_id, c.first_name, EXTRACT(YEAR FROM age(c.birth_date))::int, c.gender, c.bio, c.city,
	       c.show_distance, `+haversineKM+` AS dist
	FROM me JOIN profiles c ON c.user_id = ANY ($2::uuid[])`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	list, err := scanPublic(rows)
	if err != nil {
		return nil, err
	}
	list, err = r.decorate(ctx, list)
	if err != nil {
		return nil, err
	}
	out := make(map[string]profiles.PublicProfile, len(list))
	for _, p := range list {
		out[p.UserID] = p
	}
	return out, nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $2 AND (user_a = $1 OR user_b = $1)
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END`, userID, matchID).Scan(&other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepoNotFound
		}
		return "", err
	}
	return other, nil
}
