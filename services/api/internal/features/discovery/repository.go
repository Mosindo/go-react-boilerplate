package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// ViewerReady reports whether the user has a profile with at least one photo.
func (r *Repository) ViewerReady(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1) AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&ok)
	return ok, err
}

func (r *Repository) TouchActivity(ctx context.Context, userID string) {
	_, _ = r.pool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'`, userID)
}

// recommendation ranking lives here only: swap this query to evolve the algorithm.
// Current order: people who already liked you, then shared interests, then recent activity.
var candidatesSQL = `
WITH me AS (
	SELECT p.user_id, p.gender, p.latitude, p.longitude, EXTRACT(YEAR FROM age(p.birth_date))::int AS age,
	       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1
), c AS (
	SELECT p.user_id, u.last_active_at, pr.max_distance_km AS their_max,
	       EXTRACT(YEAR FROM age(p.birth_date))::int AS age, pr.min_age, pr.max_age,
	       CASE WHEN me.latitude IS NOT NULL AND p.latitude IS NOT NULL
	            THEN ` + db.HaversineKM("me.latitude", "me.longitude", "p.latitude", "p.longitude") + ` END AS dist
	FROM me, profiles p
	JOIN preferences pr ON pr.user_id = p.user_id
	JOIN users u ON u.id = p.user_id
	WHERE p.user_id <> me.user_id
	  AND p.discoverable
	  AND ($3::uuid IS NULL OR p.user_id = $3::uuid)
	  AND p.gender = ANY (me.interested_in)
	  AND me.gender = ANY (pr.interested_in)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
	                                             OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id))
	  AND (me.max_distance_km IS NULL OR me.latitude IS NULL
	       OR p.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0)
)
SELECT c.user_id
FROM c, me
WHERE c.age BETWEEN me.min_age AND me.max_age
  AND me.age BETWEEN c.min_age AND c.max_age
  AND (me.max_distance_km IS NULL OR me.latitude IS NULL OR (c.dist IS NOT NULL AND c.dist <= me.max_distance_km))
  AND (c.their_max IS NULL OR c.dist IS NULL OR c.dist <= c.their_max)
ORDER BY EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = c.user_id AND s.to_user_id = me.user_id AND s.action = 'like') DESC,
         (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = me.user_id
          WHERE a.user_id = c.user_id) DESC,
         c.last_active_at DESC,
         c.user_id
LIMIT $2`

func (r *Repository) scanIDs(ctx context.Context, sql string, args ...any) ([]string, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) CandidateIDs(ctx context.Context, viewerID string, limit int) ([]string, error) {
	return r.scanIDs(ctx, candidatesSQL, viewerID, limit, (*string)(nil))
}

// CanSwipe is true when target is a compatible, not-yet-swiped candidate of the viewer, or when the
// two already interacted (a repeat/upgrade of the viewer's swipe, or the target liked the viewer first).
func (r *Repository) CanSwipe(ctx context.Context, viewerID, targetID string) (bool, error) {
	var interacted bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2)
		    OR EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $2 AND to_user_id = $1 AND action = 'like')`, viewerID, targetID).Scan(&interacted); err != nil {
		return false, err
	}
	if interacted {
		return true, nil
	}
	ids, err := r.scanIDs(ctx, candidatesSQL, viewerID, 1, targetID)
	return len(ids) == 1, err
}

func (r *Repository) LikesReceivedIDs(ctx context.Context, viewerID string, limit int) ([]string, error) {
	return r.scanIDs(ctx, `
		SELECT s.from_user_id FROM swipes s
		WHERE s.to_user_id = $1 AND s.action = 'like'
		  AND NOT EXISTS (SELECT 1 FROM swipes r WHERE r.from_user_id = $1 AND r.to_user_id = s.from_user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = s.from_user_id)
		                                             OR (b.blocker_id = s.from_user_id AND b.blocked_id = $1))
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = s.from_user_id)
		  AND EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = s.from_user_id)
		ORDER BY s.created_at DESC LIMIT $2`, viewerID, limit)
}

// Outcome describes what a swipe did, so the service can emit realtime events after commit.
type Outcome struct {
	Action         string
	Matched        bool
	NewMatch       bool
	ConversationID string
	Notifications  []Notified
}

type Notified struct {
	UserID       string
	Notification notifications.Notification
}

func pairLock(ctx context.Context, tx pgx.Tx, a, b string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 1))`, a, b)
	return err
}

// Swipe records the decision atomically. The pair lock serialises concurrent opposite likes so that
// exactly one of them observes the other and creates the (single) match.
func (r *Repository) Swipe(ctx context.Context, from, to, action string) (Outcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Outcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := pairLock(ctx, tx, from, to); err != nil {
		return Outcome{}, err
	}

	out := Outcome{Action: action}
	var stored string
	err = tx.QueryRow(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO UPDATE
		SET action = EXCLUDED.action, updated_at = NOW()
		WHERE swipes.action = 'pass' AND EXCLUDED.action = 'like'
		RETURNING action`, from, to, action).Scan(&stored)
	fresh := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, from, to).Scan(&stored); err != nil {
			return Outcome{}, err
		}
	} else if err != nil {
		return Outcome{}, err
	}
	out.Action = stored

	if stored == ActionLike {
		if err := r.resolveLike(ctx, tx, from, to, fresh, &out); err != nil {
			return Outcome{}, err
		}
	}
	return out, tx.Commit(ctx)
}

func (r *Repository) resolveLike(ctx context.Context, tx pgx.Tx, from, to string, fresh bool, out *Outcome) error {
	if !fresh {
		// Idempotent repeat: report the existing match, create nothing.
		err := tx.QueryRow(ctx, `
			SELECT c.id FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)`, from, to).Scan(&out.ConversationID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		out.Matched = err == nil
		return err
	}

	names := map[string]string{}
	rows, err := tx.Query(ctx, `SELECT user_id, first_name FROM profiles WHERE user_id IN ($1, $2)`, from, to)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		names[id] = name
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, to, from).Scan(&reciprocal); err != nil {
		return err
	}
	if !reciprocal {
		n, err := notifications.Insert(ctx, tx, notifications.NewNotification{
			UserID: to, Type: notifications.TypeNewLike, Title: "Nouveau like",
			Body: names[from] + " vous a liké.", Data: map[string]string{"userId": from},
		})
		if err != nil {
			return err
		}
		out.Notifications = append(out.Notifications, Notified{UserID: to, Notification: n})
		return nil
	}

	var matchID string
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id`, from, to).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	convID, err := chat.CreateConversation(ctx, tx, matchID, from, to)
	if err != nil {
		return err
	}
	out.Matched, out.NewMatch, out.ConversationID = true, true, convID
	for _, pair := range [][2]string{{from, to}, {to, from}} {
		n, err := notifications.Insert(ctx, tx, notifications.NewNotification{
			UserID: pair[0], Type: notifications.TypeNewMatch, Title: "C'est un match !",
			Body: "Vous avez un match avec " + names[pair[1]] + ".", Data: map[string]string{"userId": pair[1], "conversationId": convID},
		})
		if err != nil {
			return err
		}
		out.Notifications = append(out.Notifications, Notified{UserID: pair[0], Notification: n})
	}
	return nil
}

// RemovedMatch identifies what was deleted when a match is dissolved.
type RemovedMatch struct {
	ConversationID string
	UserA, UserB   string
}

// DeleteMatchBetween dissolves the match of a pair (conversation and messages cascade).
func DeleteMatchBetween(ctx context.Context, q db.Querier, a, b string) (*RemovedMatch, error) {
	var rm RemovedMatch
	err := q.QueryRow(ctx, `
		WITH m AS (
			DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
			RETURNING id, user_a, user_b
		)
		SELECT COALESCE(c.id::text, ''), m.user_a, m.user_b FROM m LEFT JOIN conversations c ON c.match_id = m.id`, a, b).
		Scan(&rm.ConversationID, &rm.UserA, &rm.UserB)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rm, nil
}

// DeleteMatch dissolves a match by id if userID is one of its members.
func (r *Repository) DeleteMatch(ctx context.Context, matchID, userID string) (*RemovedMatch, error) {
	var rm RemovedMatch
	err := r.pool.QueryRow(ctx, `
		WITH m AS (
			DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2) RETURNING id, user_a, user_b
		)
		SELECT COALESCE(c.id::text, ''), m.user_a, m.user_b FROM m LEFT JOIN conversations c ON c.match_id = m.id`, matchID, userID).
		Scan(&rm.ConversationID, &rm.UserA, &rm.UserB)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rm, nil
}
