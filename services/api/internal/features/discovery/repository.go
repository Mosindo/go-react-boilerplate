package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrNotCompleted      = errors.New("profile not completed")
)

type Repository interface {
	Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error)
}

type PGRepository struct {
	db       *pgxpool.Pool
	profiles *profiles.PGRepository
}

func NewPGRepository(db *pgxpool.Pool, profileRepo *profiles.PGRepository) *PGRepository {
	return &PGRepository{db: db, profiles: profileRepo}
}

// candidateQuery selects who to show: strictly the Recommender seam of the product.
//
// Hard filters (never relaxed): active, discoverable, mutual gender interest, mutual age range,
// mutual max distance (when both sides share a location), has a photo, never swiped by the viewer,
// not blocked either way, not already matched.
//
// Ranking (rankScore): the only part to change when improving the recommendation algorithm.
// Today: people who already liked the viewer first, then shared interests, then proximity, then recent activity.
const candidateFrom = `
	FROM profiles me
	JOIN preferences mp ON mp.user_id = me.user_id
	CROSS JOIN profiles t
	JOIN preferences tp ON tp.user_id = t.user_id
	JOIN users tu ON tu.id = t.user_id`

// compatibility holds the hard filters, shared by the deck and by swipe authorization so the server
// never accepts a like for someone the viewer could not have been shown.
const compatibility = `
	AND t.discoverable
	AND t.gender = ANY(mp.interested_in)
	AND me.gender = ANY(tp.interested_in)
	AND date_part('year', age(CURRENT_DATE, t.birth_date)) BETWEEN mp.min_age AND mp.max_age
	AND date_part('year', age(CURRENT_DATE, me.birth_date)) BETWEEN tp.min_age AND tp.max_age
	AND (me.latitude IS NULL OR t.latitude IS NULL OR ` + profiles.DistanceSQL + ` <= LEAST(mp.max_distance_km, tp.max_distance_km))`

// candidateQuery selects who to show: strictly the Recommender seam of the product.
//
// Hard filters (never relaxed): active, discoverable, mutual gender interest, mutual age range,
// mutual max distance (when both sides share a location), has a photo, never swiped by the viewer,
// not blocked either way.
//
// Ranking (rankScore): the only part to change when improving the recommendation algorithm.
// Today: people who already liked the viewer first, then shared interests, then proximity, then recent activity.
const candidateQuery = profiles.PublicColumns + candidateFrom + `
	WHERE me.user_id = $1
	  AND ` + profiles.VisibilitySQL + compatibility + `
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = me.user_id AND s.target_id = t.user_id)
	ORDER BY ` + rankScore + ` DESC, tu.last_active_at DESC, t.user_id
	LIMIT $2`

const rankScore = `(
	  CASE WHEN EXISTS (SELECT 1 FROM swipes l WHERE l.actor_id = t.user_id AND l.target_id = me.user_id AND l.action = 'like') THEN 100 ELSE 0 END
	+ 10 * (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
	         WHERE a.user_id = me.user_id AND b.user_id = t.user_id)
	+ CASE WHEN me.latitude IS NULL OR t.latitude IS NULL THEN 0
	       ELSE GREATEST(0, 50 - ` + profiles.DistanceSQL + `) / 5 END
)`

func (r *PGRepository) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	var completed bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&completed); err != nil {
		return nil, err
	}
	if !completed {
		return nil, ErrNotCompleted
	}
	return r.profiles.PublicByQuery(ctx, candidateQuery, userID, limit)
}

// Swipe records the viewer's decision and, on a reciprocal like, creates the match, its conversation,
// both participants and both notifications in ONE transaction.
//
// Concurrency: a transaction-scoped advisory lock keyed on the (unordered) pair serialises two users
// liking each other at the same moment, so exactly one of them creates the match.
// Idempotency: a repeated swipe returns the stored decision and never changes anything.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
	`, userID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var completed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&completed); err != nil {
		return SwipeResult{}, err
	}
	if !completed {
		return SwipeResult{}, ErrNotCompleted
	}

	// A prior decision wins: report it (and the active match, if any) without writing.
	var stored string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE actor_id = $1 AND target_id = $2`, userID, targetID).Scan(&stored)
	if err == nil {
		matchID, err := activeMatch(ctx, tx, userID, targetID)
		if err != nil {
			return SwipeResult{}, err
		}
		return SwipeResult{Action: stored, MatchID: matchID}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeResult{}, err
	}

	// The target must be someone the viewer could have been shown (same rules as the deck).
	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 `+candidateFrom+`
		WHERE me.user_id = $1 AND t.user_id = $2 AND `+profiles.VisibilitySQL+compatibility+`)`,
		userID, targetID).Scan(&eligible); err != nil {
		return SwipeResult{}, err
	}
	if !eligible {
		return SwipeResult{}, ErrTargetUnavailable
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return SwipeResult{}, err
	}
	result := SwipeResult{Action: action}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, userID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			if err := createMatch(ctx, tx, userID, targetID, &result); err != nil {
				return SwipeResult{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return result, nil
}

func activeMatch(ctx context.Context, tx pgx.Tx, a, b string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id FROM matches
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL
	`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func createMatch(ctx context.Context, tx pgx.Tx, userID, targetID string, out *SwipeResult) error {
	var matchID string
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id
	`, userID, targetID).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Pair already matched before (e.g. unmatched later): never resurrect or duplicate it.
		return nil
	}
	if err != nil {
		return err
	}
	var conversationID string
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, matchID).Scan(&conversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, conversationID, userID, targetID); err != nil {
		return err
	}

	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		var name string
		if err := tx.QueryRow(ctx, `SELECT display_name FROM profiles WHERE user_id = $1`, pair[1]).Scan(&name); err != nil {
			return err
		}
		n, err := notifications.CreateTx(ctx, tx, notifications.New{
			UserID: pair[0],
			Type:   notifications.TypeMatch,
			Title:  "It's a match!",
			Body:   "You and " + name + " liked each other. Say hello!",
			Data:   map[string]string{"matchId": matchID, "conversationId": conversationID, "userId": pair[1]},
		})
		if err != nil {
			return err
		}
		out.Notifications = append(out.Notifications, NotificationOut{UserID: pair[0], Value: n})
	}
	out.MatchID = matchID
	out.NewMatch = true
	return nil
}
