package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("profile incomplete")
	ErrAlreadySwiped     = errors.New("already swiped")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	ProfileComplete(ctx context.Context, userID string) (bool, error)
	Candidates(ctx context.Context, viewerID string, limit int) ([]string, error)
	Swipe(ctx context.Context, actorID, targetID, action string) (*swipeResult, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) ProfileComplete(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, userID).Scan(&ok)
	return ok, err
}

// Candidates returns ranked profile ids for the viewer's discovery deck.
//
// Hard filters (always mutual): gender preferences, age ranges, distance
// (only when both members shared a location), visibility, suspension,
// blocks in either direction, and profiles the viewer already processed.
//
// Ranking (scoreCandidate) is intentionally a single SQL expression so the
// algorithm can evolve in one place: people who already liked the viewer,
// shared interests, recent activity and proximity score higher.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.latitude, p.longitude, age_years(p.birthdate) AS age,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		),
		candidates AS (
			SELECT c.user_id,
			       distance_km(me.latitude, me.longitude, c.latitude, c.longitude) AS dist,
			       u.last_active_at,
			       EXISTS (
			         SELECT 1 FROM swipes s
			         WHERE s.actor_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like'
			       ) AS liked_me,
			       (
			         SELECT COUNT(*) FROM user_interests a
			         JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = me.user_id
			         WHERE a.user_id = c.user_id
			       ) AS shared
			FROM me
			JOIN profiles c ON c.user_id <> me.user_id
			JOIN preferences cp ON cp.user_id = c.user_id
			JOIN users u ON u.id = c.user_id
			WHERE c.discoverable
			  AND u.suspended_at IS NULL
			  AND c.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cp.interested_in)
			  AND age_years(c.birthdate) BETWEEN me.min_age AND me.max_age
			  AND me.age BETWEEN cp.min_age AND cp.max_age
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = me.user_id AND s.target_id = c.user_id)
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
			       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id)
			  )
			  AND (
			    me.latitude IS NULL OR c.latitude IS NULL
			    OR (
			      c.latitude BETWEEN me.latitude - (me.max_distance_km / 111.0) AND me.latitude + (me.max_distance_km / 111.0)
			      AND distance_km(me.latitude, me.longitude, c.latitude, c.longitude) <= LEAST(me.max_distance_km, cp.max_distance_km)
			    )
			  )
		)
		SELECT user_id
		FROM candidates
		ORDER BY
		  (CASE WHEN liked_me THEN 3 ELSE 0 END)
		  + LEAST(shared, 5) * 0.8
		  + (CASE WHEN last_active_at > NOW() - INTERVAL '3 days' THEN 1.5
		          WHEN last_active_at > NOW() - INTERVAL '14 days' THEN 0.5 ELSE 0 END)
		  - COALESCE(dist, 50) / 100.0 DESC,
		  user_id
		LIMIT $2
	`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// Swipe records a decision and, for a reciprocal like, creates the match.
// A per-pair advisory lock makes simultaneous mutual likes produce exactly
// one match; the swipes primary key rejects duplicates.
func (r *PGRepository) Swipe(ctx context.Context, actorID, targetID, action string) (*swipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::TEXT, $2::TEXT) || ':' || GREATEST($1::TEXT, $2::TEXT), 0))
	`, actorID, targetID); err != nil {
		return nil, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.suspended_at IS NULL
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
		)
	`, actorID, targetID).Scan(&available); err != nil {
		return nil, err
	}
	if !available {
		return nil, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (actor_id, target_id) DO NOTHING
	`, actorID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadySwiped
	}

	result := &swipeResult{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $2 AND target_id = $1 AND action = 'like')
		`, actorID, targetID).Scan(&reciprocal); err != nil {
			return nil, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a_id, user_b_id)
				VALUES (LEAST($1::UUID, $2::UUID), GREATEST($1::UUID, $2::UUID))
				ON CONFLICT (user_a_id, user_b_id) DO NOTHING
				RETURNING id, created_at
			`, actorID, targetID).Scan(&result.MatchID, &result.MatchCreatedAt)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, err
			}
			if result.MatchID != "" {
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (match_id, user_id) VALUES ($1, $2), ($1, $3)
				`, result.MatchID, actorID, targetID); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// Unmatch deletes the match and its conversation for both members. Swipes
// are kept so the two profiles do not reappear in each other's deck.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
		RETURNING CASE WHEN user_a_id = $2 THEN user_b_id ELSE user_a_id END
	`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}
