package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTargetUnavailable = errors.New("this profile is no longer available")
	ErrAlreadySwiped     = errors.New("you already responded to this profile")
	ErrProfileIncomplete = errors.New("complete your profile before liking someone")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error)
	Unmatch(ctx context.Context, userID, otherID string) (string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func orderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Swipe records a like/pass and creates the match when the like is mutual.
//
// Concurrency: a per-pair advisory transaction lock serializes the two
// members' swipes, so two simultaneous likes always produce exactly one match
// (the second transaction sees the first one's like). The primary key on
// swipes makes repeated likes impossible and the (low, high) unique key on
// matches makes duplicate matches impossible.
func (r *PGRepository) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := orderedPair(actorID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))`, low, high); err != nil {
		return SwipeResult{}, err
	}

	var actorReady bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $1 AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		)
	`, actorID).Scan(&actorReady); err != nil {
		return SwipeResult{}, err
	}
	if !actorReady {
		return SwipeResult{}, ErrProfileIncomplete
	}

	// The server re-checks that the target is a legitimate candidate: it must
	// exist in the same tenant, have a complete, visible profile, and no block
	// may exist in either direction.
	var targetOK bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles t
			JOIN users tu ON tu.id = t.user_id
			JOIN users au ON au.id = $1
			WHERE t.user_id = $2
			  AND tu.organization_id = au.organization_id
			  AND t.is_visible AND NOT t.is_suspended
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
		)
	`, actorID, targetID).Scan(&targetOK); err != nil {
		return SwipeResult{}, err
	}
	if !targetOK {
		return SwipeResult{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_user_id, target_user_id, action)
		VALUES ($1, $2, $3)
		ON CONFLICT (actor_user_id, target_user_id) DO NOTHING
	`, actorID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrAlreadySwiped
	}

	if action != ActionLike {
		return SwipeResult{}, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM swipes WHERE actor_user_id = $2 AND target_user_id = $1 AND action = 'like'
		)
	`, actorID, targetID).Scan(&reciprocal); err != nil {
		return SwipeResult{}, err
	}
	if !reciprocal {
		return SwipeResult{}, tx.Commit(ctx)
	}

	var match Match
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_low_id, user_high_id)
		VALUES ($1, $2)
		ON CONFLICT (user_low_id, user_high_id) DO NOTHING
		RETURNING id, created_at
	`, low, high).Scan(&match.ID, &match.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already matched (cannot normally happen thanks to the lock).
		return SwipeResult{}, tx.Commit(ctx)
	}
	if err != nil {
		return SwipeResult{}, err
	}

	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, created_by_user_id, match_id)
		VALUES ('direct', $1 || ':' || $2, $3, $4)
		ON CONFLICT (direct_key) DO UPDATE SET match_id = EXCLUDED.match_id, updated_at = NOW()
		RETURNING id
	`, low, high, actorID, match.ID).Scan(&conversationID); err != nil {
		return SwipeResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, low, high); err != nil {
		return SwipeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return SwipeResult{}, ErrAlreadySwiped
		}
		return SwipeResult{}, err
	}
	match.UserID = actorID
	match.OtherID = targetID
	return SwipeResult{Matched: true, Match: &match}, nil
}

// Unmatch deletes the match; the conversation and its messages are removed by
// cascade. Swipes are kept so the two members never see each other again.
func (r *PGRepository) Unmatch(ctx context.Context, userID, otherID string) (string, error) {
	low, high := orderedPair(userID, otherID)
	var matchID string
	err := r.dbPool.QueryRow(ctx, `
		DELETE FROM matches WHERE user_low_id = $1 AND user_high_id = $2 RETURNING id
	`, low, high).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return matchID, err
}
