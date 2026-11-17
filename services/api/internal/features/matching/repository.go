package matching

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadySwiped   = errors.New("already swiped")
	ErrTargetNotFound  = errors.New("target not available")
	ErrMatchNotFound   = errors.New("match not found")
	ErrNothingToUndo   = errors.New("nothing to undo")
	ErrInvalidAction   = errors.New("invalid action")
	ErrCannotSwipeSelf = errors.New("cannot swipe self")
)

type Repository interface {
	TargetSwipeable(ctx context.Context, swiperID, targetID string) (bool, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error)
	UndoLastPass(ctx context.Context, swiperID string, window time.Duration) (string, error)
	EndMatch(ctx context.Context, userID, matchID string) (otherUserID, conversationID string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// TargetSwipeable checks the target exists with a complete, discoverable profile and
// that no block exists between both users.
func (r *PGRepository) TargetSwipeable(ctx context.Context, swiperID, targetID string) (bool, error) {
	var ok bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2 AND p.discoverable
			  AND p.first_name <> '' AND p.birthdate IS NOT NULL AND p.gender IS NOT NULL AND p.latitude IS NOT NULL
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
			      SELECT 1 FROM blocks b
			      WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		)
	`, swiperID, targetID).Scan(&ok)
	return ok, err
}

// Swipe records a like or pass and creates the match (and its conversation) when the
// like is reciprocal. A transaction-scoped advisory lock on the pair serialises two
// simultaneous likes so the mutual like is always detected exactly once.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
	`, swiperID, targetID); err != nil {
		return SwipeOutcome{}, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return SwipeOutcome{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeOutcome{}, ErrAlreadySwiped
	}

	if action != ActionLike {
		return SwipeOutcome{}, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
	`, swiperID, targetID).Scan(&reciprocal); err != nil {
		return SwipeOutcome{}, err
	}
	if !reciprocal {
		return SwipeOutcome{}, tx.Commit(ctx)
	}

	out := SwipeOutcome{Matched: true}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING
		RETURNING id, created_at
	`, swiperID, targetID).Scan(&out.MatchID, &out.MatchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// A match row already exists for this pair (e.g. ended earlier): no new match.
		return SwipeOutcome{}, tx.Commit(ctx)
	}
	if err != nil {
		return SwipeOutcome{}, err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1) RETURNING id
	`, out.MatchID).Scan(&out.ConversationID); err != nil {
		return SwipeOutcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, out.ConversationID, swiperID, targetID); err != nil {
		return SwipeOutcome{}, err
	}

	return out, tx.Commit(ctx)
}

// UndoLastPass removes the most recent swipe if it was a pass made within window.
// Likes cannot be undone (they may already have produced a match).
func (r *PGRepository) UndoLastPass(ctx context.Context, swiperID string, window time.Duration) (string, error) {
	var targetID string
	err := r.dbPool.QueryRow(ctx, `
		WITH last AS (
			SELECT swiper_id, target_id, action, created_at
			FROM swipes WHERE swiper_id = $1
			ORDER BY created_at DESC LIMIT 1
		)
		DELETE FROM swipes s
		USING last
		WHERE s.swiper_id = last.swiper_id AND s.target_id = last.target_id
		  AND last.action = 'pass' AND last.created_at > NOW() - make_interval(secs => $2::double precision)
		RETURNING s.target_id
	`, swiperID, window.Seconds()).Scan(&targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNothingToUndo
	}
	return targetID, err
}

// EndMatch unmatches; the conversation disappears for both members but messages are
// retained for moderation of reports.
func (r *PGRepository) EndMatch(ctx context.Context, userID, matchID string) (string, string, error) {
	var otherID, conversationID string
	err := r.dbPool.QueryRow(ctx, `
		UPDATE matches m SET ended_at = NOW(), ended_by = $2
		FROM conversations c
		WHERE m.id = $1 AND c.match_id = m.id
		  AND (m.user_a_id = $2 OR m.user_b_id = $2)
		  AND m.ended_at IS NULL
		RETURNING CASE WHEN m.user_a_id = $2 THEN m.user_b_id ELSE m.user_a_id END, c.id
	`, matchID, userID).Scan(&otherID, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrMatchNotFound
	}
	return otherID, conversationID, err
}
