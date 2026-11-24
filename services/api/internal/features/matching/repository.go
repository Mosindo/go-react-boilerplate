package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryAlreadySwiped = errors.New("already swiped")
	ErrRepositoryNotFound      = errors.New("not found")
	ErrRepositoryIncomplete    = errors.New("viewer profile incomplete")
)

type ViewerContext struct {
	MaxDistanceKm int
}

type Repository interface {
	Viewer(ctx context.Context, userID string) (ViewerContext, error)
	CandidatePool(ctx context.Context, userID string, includeDemo bool, poolSize int) ([]Candidate, error)
	Swipe(ctx context.Context, userID, targetID, decision string, includeDemo bool) (*Match, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	TouchUser(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// Viewer loads the viewer's discovery settings and checks the profile is
// complete (profile, location and at least one photo).
func (r *PGRepository) Viewer(ctx context.Context, userID string) (ViewerContext, error) {
	var v ViewerContext
	err := r.db.QueryRow(ctx, `
		SELECT pr.max_distance_km
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
		  AND p.latitude IS NOT NULL
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	`, userID).Scan(&v.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return ViewerContext{}, ErrRepositoryIncomplete
	}
	return v, err
}

// CandidatePool returns discoverable members matching BOTH sides'
// preferences (gender, age, distance), excluding anyone already swiped,
// blocked in either direction, or under moderation review. A bounding box on
// the indexed coordinates prefilters before the exact distance check.
func (r *PGRepository) CandidatePool(ctx context.Context, userID string, includeDemo bool, poolSize int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude AS lat, p.longitude AS lng,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
			       pr.max_distance_km / 111.0 AS dlat,
			       pr.max_distance_km / (111.0 * GREATEST(cos(radians(p.latitude)), 0.01)) AS dlng
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1 AND p.latitude IS NOT NULL
		),
		pool AS (
			SELECT c.user_id,
			       distance_km(me.lat, me.lng, c.latitude, c.longitude) AS dist,
			       cp.max_distance_km AS their_max_distance,
			       u.last_active_at
			FROM me
			JOIN profiles c
			  ON c.is_discoverable
			 AND c.latitude IS NOT NULL
			 AND c.user_id <> me.user_id
			 AND c.latitude BETWEEN me.lat - me.dlat AND me.lat + me.dlat
			 AND c.longitude BETWEEN me.lng - me.dlng AND me.lng + me.dlng
			JOIN preferences cp ON cp.user_id = c.user_id
			JOIN users u ON u.id = c.user_id
			WHERE c.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cp.interested_in)
			  AND c.birthdate <= (CURRENT_DATE - make_interval(years => me.min_age::int))
			  AND c.birthdate > (CURRENT_DATE - make_interval(years => me.max_age::int + 1))
			  AND me.birthdate <= (CURRENT_DATE - make_interval(years => cp.min_age::int))
			  AND me.birthdate > (CURRENT_DATE - make_interval(years => cp.max_age::int + 1))
			  AND ($2 OR NOT u.is_demo)
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
				   OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id)
			  )
			  AND (SELECT COUNT(DISTINCT r.reporter_id) FROM reports r
			       JOIN users ru ON ru.id = r.reporter_id
			       WHERE r.reported_user_id = c.user_id AND r.status = 'open'
			         AND ru.created_at < NOW() - INTERVAL '1 day') < 3
		)
		SELECT pool.user_id, pool.dist,
		       (SELECT COUNT(*) FROM user_interests a
		        JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = pool.user_id
		        WHERE a.user_id = $1) AS shared,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = pool.user_id AND s.target_id = $1 AND s.decision = 'like') AS liked_me,
		       pool.last_active_at
		FROM pool, me
		WHERE GREATEST(pool.dist, 5) <= me.max_distance_km AND GREATEST(pool.dist, 5) <= pool.their_max_distance
		ORDER BY pool.last_active_at DESC, pool.user_id
		LIMIT $3
	`, userID, includeDemo, poolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Candidate, 0, poolSize)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.SharedCount, &c.LikedViewer, &c.LastActiveAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Swipe records a decision and atomically creates the match (plus its
// conversation) when the like is mutual. A per-pair advisory lock removes
// the race where two simultaneous likes would each miss the other.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, decision string, includeDemo bool) (*Match, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := OrderedPair(userID, targetID)
	if err := LockPair(ctx, tx, userID, targetID); err != nil {
		return nil, err
	}

	// Eligibility is checked under the pair lock, so a concurrent block
	// (which takes the same lock) can never be bypassed.
	var eligible bool
	if err := tx.QueryRow(ctx, swipeEligibilitySQL, userID, targetID, includeDemo).Scan(&eligible); err != nil {
		return nil, err
	}
	if !eligible {
		return nil, ErrRepositoryNotFound
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, decision) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, userID, targetID, decision)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRepositoryAlreadySwiped
	}

	var match *Match
	if decision == DecisionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND decision = 'like')
		`, targetID, userID).Scan(&reciprocal); err != nil {
			return nil, err
		}
		if reciprocal {
			m := Match{}
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_low_id, user_high_id) VALUES ($1, $2)
				ON CONFLICT (user_low_id, user_high_id) DO NOTHING
				RETURNING id, created_at
			`, low, high).Scan(&m.ID, &m.CreatedAt)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			if err == nil {
				if err := tx.QueryRow(ctx, `
					INSERT INTO conversations (match_id) VALUES ($1) RETURNING id
				`, m.ID).Scan(&m.ConversationID); err != nil {
					return nil, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
				`, m.ConversationID, low, high); err != nil {
					return nil, err
				}
				match = &m
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return match, nil
}

// Unmatch deletes the match; its conversation and messages cascade.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var low, high string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
		RETURNING user_low_id, user_high_id
	`, matchID, userID).Scan(&low, &high)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if low == userID {
		return high, nil
	}
	return low, nil
}

func (r *PGRepository) TouchUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'`, userID)
	return err
}

// LockPair serialises every write touching the relationship between two
// members (swipes, matches, blocks) for the current transaction.
func LockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	low, high := OrderedPair(a, b)
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('pair:' || $1::text || ':' || $2::text, 0))`, low, high)
	return err
}

// swipeEligibilitySQL mirrors the discovery rules (except distance, which
// legitimately changes when members move): visible and complete target,
// mutually compatible gender and age, no block either way, not on moderation
// hold, demo accounts hidden in production.
const swipeEligibilitySQL = `
	SELECT EXISTS (
		SELECT 1
		FROM profiles me
		JOIN preferences mp ON mp.user_id = me.user_id
		JOIN profiles t ON t.user_id = $2
		JOIN preferences tp ON tp.user_id = t.user_id
		JOIN users tu ON tu.id = t.user_id
		WHERE me.user_id = $1
		  AND t.is_discoverable
		  AND t.latitude IS NOT NULL
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
		  AND ($3 OR NOT tu.is_demo)
		  AND t.gender = ANY(mp.interested_in)
		  AND me.gender = ANY(tp.interested_in)
		  AND t.birthdate <= (CURRENT_DATE - make_interval(years => mp.min_age::int))
		  AND t.birthdate > (CURRENT_DATE - make_interval(years => mp.max_age::int + 1))
		  AND me.birthdate <= (CURRENT_DATE - make_interval(years => tp.min_age::int))
		  AND me.birthdate > (CURRENT_DATE - make_interval(years => tp.max_age::int + 1))
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
		  )
		  AND ` + moderationHoldClear + `
	)`

// moderationHoldClear excludes members reported by 3+ distinct accounts that
// are at least a day old, so throwaway accounts cannot shadow-ban someone.
const moderationHoldClear = `(SELECT COUNT(DISTINCT r.reporter_id) FROM reports r
	JOIN users ru ON ru.id = r.reporter_id
	WHERE r.reported_user_id = $2 AND r.status = 'open' AND ru.created_at < NOW() - INTERVAL '1 day') < 3`
