package matching

import (
	"context"
	"errors"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var (
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrSelfSwipe         = errors.New("cannot swipe yourself")
	ErrNoProfile         = errors.New("complete your profile first")
	ErrMatchNotFound     = errors.New("match not found")
)

// Recommender produces the next candidates for a user. The SQL implementation
// below is the default; a smarter ranking (or an external service) only has to
// satisfy this interface.
type Recommender interface {
	Candidates(ctx context.Context, userID string, limit int) ([]Candidate, error)
}

type Repository interface {
	Recommender
	Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error)
	ResetPasses(ctx context.Context, userID string) (int64, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Candidates ranks eligible profiles. Eligibility is mutual: each side's gender,
// age and distance preferences must accept the other. Score favours shared
// interests, proximity, recent activity and people who already liked the viewer.
// Everything runs in one query; photos/interests are batch-loaded afterwards.
func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.lat, p.lng,
			       date_part('year', age(p.birth_date))::int AS age,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
			FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		), cand AS (
			SELECT p.user_id, p.first_name, p.gender, p.bio, p.city, p.show_distance, u.last_active_at,
			       date_part('year', age(p.birth_date))::int AS age,
			       2 * 6371 * asin(least(1, sqrt(
			         power(sin(radians(p.lat - me.lat) / 2), 2) +
			         cos(radians(me.lat)) * cos(radians(p.lat)) * power(sin(radians(p.lng - me.lng) / 2), 2)))) AS dist,
			       LEAST(me.max_distance_km, pr.max_distance_km) AS radius
			FROM me
			JOIN profiles p ON p.user_id <> me.user_id AND p.is_discoverable
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.gender = ANY(me.interested_in)
			  AND me.gender = ANY(pr.interested_in)
			  AND date_part('year', age(p.birth_date)) BETWEEN me.min_age AND me.max_age
			  AND me.age BETWEEN pr.min_age AND pr.max_age
			  AND p.lat BETWEEN me.lat - LEAST(me.max_distance_km, pr.max_distance_km) / 111.0
			                AND me.lat + LEAST(me.max_distance_km, pr.max_distance_km) / 111.0
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = me.user_id AND s.target_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM blocks b
			        WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
			           OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id))
		)
		SELECT c.user_id, c.first_name, c.age, c.gender, c.bio, c.city, c.dist, c.show_distance
		FROM cand c
		WHERE c.dist <= c.radius
		ORDER BY (
		    2.0 * (SELECT count(*) FROM user_interests a
		           JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = c.user_id
		           WHERE a.user_id = $1)
		  - c.dist / 25.0
		  + CASE WHEN c.last_active_at > NOW() - INTERVAL '3 days' THEN 1.0 ELSE 0 END
		  + CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = c.user_id AND s.target_id = $1 AND s.action = 'like')
		         THEN 1.5 ELSE 0 END
		) DESC, c.user_id
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Candidate, 0, limit)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.ID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.DistanceKm, &c.ShowDistance); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Swipe records a decision atomically. A pair-level advisory lock serialises two
// people liking each other at the same moment, so exactly one match is created.
func (r *PGRepository) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error) {
	if actorID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	if !uuidPattern.MatchString(targetID) {
		return SwipeResult{}, ErrTargetUnavailable
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(least($1::text, $2::text) || ':' || greatest($1::text, $2::text), 0))`, actorID, targetID); err != nil {
		if isInvalidUUID(err) {
			return SwipeResult{}, ErrTargetUnavailable
		}
		return SwipeResult{}, err
	}

	var res SwipeResult
	var haveActor, targetOK bool
	err = tx.QueryRow(ctx, `
		SELECT
		  COALESCE((SELECT first_name FROM profiles WHERE user_id = $1), ''),
		  COALESCE((SELECT first_name FROM profiles WHERE user_id = $2), ''),
		  EXISTS (SELECT 1 FROM profiles WHERE user_id = $1),
		  EXISTS (SELECT 1 FROM profiles WHERE user_id = $2)
		    AND NOT EXISTS (SELECT 1 FROM blocks b
		          WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
	`, actorID, targetID).Scan(&res.ActorName, &res.TargetName, &haveActor, &targetOK)
	if err != nil {
		return SwipeResult{}, err
	}
	if !haveActor {
		return SwipeResult{}, ErrNoProfile
	}
	if !targetOK {
		return SwipeResult{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (actor_id, target_id) DO NOTHING
	`, actorID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	res.Duplicate = tag.RowsAffected() == 0

	if action == ActionLike || res.Duplicate {
		// Report the current match state (also for duplicate likes, idempotently).
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $1 AND target_id = $2 AND action = 'like')
			   AND EXISTS (SELECT 1 FROM swipes WHERE actor_id = $2 AND target_id = $1 AND action = 'like')
		`, actorID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			created, err := r.ensureMatch(ctx, tx, actorID, targetID, &res)
			if err != nil {
				return SwipeResult{}, err
			}
			res.Matched = true
			// Only announce a match the first time it is created.
			if !created {
				res.Duplicate = true
			}
		}
	}
	return res, tx.Commit(ctx)
}

func (r *PGRepository) ensureMatch(ctx context.Context, tx pgx.Tx, actorID, targetID string, res *SwipeResult) (bool, error) {
	var created bool
	err := tx.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO matches (user_a, user_b)
			VALUES (least($1::uuid, $2::uuid), greatest($1::uuid, $2::uuid))
			ON CONFLICT (user_a, user_b) DO NOTHING
			RETURNING id
		)
		SELECT COALESCE((SELECT id FROM ins), (SELECT id FROM matches WHERE user_a = least($1::uuid, $2::uuid) AND user_b = greatest($1::uuid, $2::uuid))),
		       EXISTS (SELECT 1 FROM ins)
	`, actorID, targetID).Scan(&res.MatchID, &created)
	if err != nil {
		return false, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id
	`, res.MatchID).Scan(&res.ConversationID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING
	`, res.ConversationID, actorID, targetID); err != nil {
		return false, err
	}
	return created, nil
}

func (r *PGRepository) ResetPasses(ctx context.Context, userID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM swipes WHERE actor_id = $1 AND action = 'pass'`, userID)
	return tag.RowsAffected(), err
}

// Unmatch dissolves a match (and, by cascade, its conversation and messages).
// The swipe rows stay, so the two people never reappear in each other's feed.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2) RETURNING user_a, user_b
	`, matchID, userID).Scan(&a, &b)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", ErrMatchNotFound
		}
		return "", err
	}
	if a == userID {
		return b, nil
	}
	return a, nil
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
