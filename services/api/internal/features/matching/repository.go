package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoProfile         = errors.New("complete your profile first")
	ErrTargetUnavailable = errors.New("profile not available")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error)
	Matches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, int, error)
	Unmatch(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// Swipe records the decision and, on a mutual like, creates exactly one match.
//
// A per-pair advisory lock serialises two people liking each other at the same
// instant: without it both transactions could miss the other's uncommitted like and
// no match would ever form. The canonical (user_a < user_b) pair plus the unique
// constraint make duplicates impossible even without the lock.
func (r *PGRepository) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::uuid, $2::uuid)::text || GREATEST($1::uuid, $2::uuid)::text, 11))
	`, actorID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var actorReady, targetOK bool
	if err := tx.QueryRow(ctx, `
		SELECT
		  EXISTS (SELECT 1 FROM profiles WHERE user_id = $1) AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1),
		  EXISTS (
		    SELECT 1 FROM profiles t
		    WHERE t.user_id = $2
		      AND EXISTS (SELECT 1 FROM photos WHERE user_id = t.user_id)
		      AND NOT EXISTS (SELECT 1 FROM blocks b
		                      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		      AND (t.discoverable
		           OR EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		  )
	`, actorID, targetID).Scan(&actorReady, &targetOK); err != nil {
		return SwipeResult{}, err
	}
	if !actorReady {
		return SwipeResult{}, ErrNoProfile
	}
	if !targetOK {
		return SwipeResult{}, ErrTargetUnavailable
	}

	userA, userB := canonical(actorID, targetID)

	// Already matched: the swipe is idempotent and never changes the outcome.
	var matchID string
	err = tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_a = $1 AND user_b = $2`, userA, userB).Scan(&matchID)
	if err == nil {
		return SwipeResult{Matched: true, MatchID: matchID, UserA: userA, UserB: userB}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeResult{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (actor_id, target_id) DO UPDATE SET action = EXCLUDED.action, updated_at = NOW()
		WHERE swipes.action <> EXCLUDED.action
	`, actorID, targetID, action); err != nil {
		return SwipeResult{}, err
	}

	result := SwipeResult{UserA: userA, UserB: userB}
	if action == ActionLike {
		var reverse bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, actorID).Scan(&reverse); err != nil {
			return SwipeResult{}, err
		}
		if reverse {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id
			`, userA, userB).Scan(&result.MatchID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return SwipeResult{}, err
			}
			result.Matched = result.MatchID != ""
			result.Created = result.Matched
		}
	}
	return result, tx.Commit(ctx)
}

func canonical(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func (r *PGRepository) Matches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, CASE WHEN user_a = $1 THEN user_b ELSE user_a END, created_at
		FROM matches
		WHERE user_a = $1 OR user_b = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredMatch
	for rows.Next() {
		var m StoredMatch
		if err := rows.Scan(&m.ID, &m.OtherID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LikesReceived lists people who liked userID and are still waiting for an answer.
func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, int, error) {
	const filter = `
		FROM swipes s
		JOIN profiles p ON p.user_id = s.actor_id AND p.discoverable
		WHERE s.target_id = $1 AND s.action = 'like'
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = s.actor_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.actor_id = $1 AND mine.target_id = s.actor_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = s.actor_id) OR (b.blocker_id = s.actor_id AND b.blocked_id = $1))`

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+filter, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT s.actor_id::text `+filter+` ORDER BY s.created_at DESC, s.actor_id LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

// Unmatch removes the match (and with it the conversation) for both people. The
// caller's swipe flips to "pass" so the pair cannot silently re-match later.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var other string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)
		RETURNING CASE WHEN user_a = $2 THEN user_b ELSE user_a END
	`, matchID, userID).Scan(&other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMatchNotFound
		}
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE swipes SET action = 'pass', updated_at = NOW() WHERE actor_id = $1 AND target_id = $2
	`, userID, other); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
