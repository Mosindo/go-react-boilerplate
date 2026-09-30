package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryTargetUnavailable = errors.New("target unavailable")
	ErrRepositorySwiperIncomplete  = errors.New("swiper profile incomplete")
	ErrRepositoryAlreadySwiped     = errors.New("already swiped")
	ErrRepositoryNotFound          = errors.New("match not found")
)

type Repository interface {
	Swipe(ctx context.Context, swiperID, targetID, action string) (*swipeResult, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// lockPair serialises concurrent swipes between the same two users so that
// simultaneous reciprocal likes always produce exactly one match.
func lockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	low, high := a, b
	if high < low {
		low, high = high, low
	}
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "pair:"+low+":"+high)
	return err
}

func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (*swipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockPair(ctx, tx, swiperID, targetID); err != nil {
		return nil, err
	}

	var swiperComplete bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles s WHERE s.user_id = $1 AND `+profiles.CompleteSQL("s")+`)
	`, swiperID).Scan(&swiperComplete); err != nil {
		return nil, err
	}
	if !swiperComplete {
		return nil, ErrRepositorySwiperIncomplete
	}

	var targetAvailable bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles t
			WHERE t.user_id = $2 AND t.user_id <> $1
			  AND `+profiles.EligibleSQL("t")+`
			  AND `+profiles.NotBlockedSQL("$1::uuid", "t.user_id")+`
		)
	`, swiperID, targetID).Scan(&targetAvailable); err != nil {
		return nil, err
	}
	if !targetAvailable {
		return nil, ErrRepositoryTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action)
		VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRepositoryAlreadySwiped
	}

	if action != ActionLike {
		return nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
	`, swiperID, targetID).Scan(&reciprocal); err != nil {
		return nil, err
	}
	if !reciprocal {
		return nil, tx.Commit(ctx)
	}

	result := &swipeResult{}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_low_id, user_high_id)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_low_id, user_high_id) DO NOTHING
		RETURNING id, created_at
	`, swiperID, targetID).Scan(&result.MatchID, &result.MatchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already matched (cannot normally happen thanks to the pair lock).
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1) RETURNING id
	`, result.MatchID).Scan(&result.ConversationID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
	`, result.ConversationID, swiperID, targetID); err != nil {
		return nil, err
	}

	return result, tx.Commit(ctx)
}

// Unmatch deletes the match (cascading to its conversation and messages) and
// turns the caller's like into a pass so the profile never comes back.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var otherID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
		RETURNING CASE WHEN user_low_id = $2 THEN user_high_id ELSE user_low_id END
	`, matchID, userID).Scan(&otherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE swipes SET action = 'pass' WHERE swiper_id = $1 AND target_id = $2
	`, userID, otherID); err != nil {
		return "", err
	}
	return otherID, tx.Commit(ctx)
}
