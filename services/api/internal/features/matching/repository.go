package matching

import (
	"context"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound     = errors.New("not found")
	ErrRepositoryAlreadyTaken = errors.New("already swiped")
	ErrRepositoryNotEligible  = errors.New("target not eligible")
	ErrRepositoryNoProfile    = errors.New("actor has no profile")
)

type Repository interface {
	Swipe(ctx context.Context, actorID, targetID, decision string) (swipeResult, error)
	Summary(ctx context.Context, viewerID, otherID string) (storedSummary, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error)
	Unmatch(ctx context.Context, userID, otherID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// Swipe records a decision and creates the match when the like is mutual.
func (r *PGRepository) Swipe(ctx context.Context, actorID, targetID, decision string) (swipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return swipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := db.LockPair(ctx, tx, actorID, targetID); err != nil {
		return swipeResult{}, err
	}

	var hasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, actorID).Scan(&hasProfile); err != nil {
		return swipeResult{}, err
	}
	if !hasProfile {
		return swipeResult{}, ErrRepositoryNoProfile
	}

	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles t
			JOIN users tu ON tu.id = t.user_id
			JOIN users au ON au.id = $1
			WHERE t.user_id = $2
			  AND t.user_id <> au.id
			  AND tu.organization_id = au.organization_id
			  AND t.is_discoverable
			  AND NOT t.moderation_hidden
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
			       OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
		)
	`, actorID, targetID).Scan(&eligible); err != nil {
		return swipeResult{}, err
	}
	if !eligible {
		return swipeResult{}, ErrRepositoryNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_user_id, target_user_id, decision)
		VALUES ($1, $2, $3)
		ON CONFLICT (actor_user_id, target_user_id) DO NOTHING
	`, actorID, targetID, decision)
	if err != nil {
		return swipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return swipeResult{}, ErrRepositoryAlreadyTaken
	}

	result := swipeResult{}
	if decision == DecisionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM swipes WHERE actor_user_id = $2 AND target_user_id = $1 AND decision = 'like'
			)
		`, actorID, targetID).Scan(&reciprocal); err != nil {
			return swipeResult{}, err
		}
		if reciprocal {
			var conversationID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO conversations (kind, direct_key, created_by_user_id)
				VALUES ('direct', $1, $2)
				ON CONFLICT (direct_key) DO UPDATE SET updated_at = NOW()
				RETURNING id
			`, db.PairKey(actorID, targetID), actorID).Scan(&conversationID); err != nil {
				return swipeResult{}, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO conversation_participants (conversation_id, user_id)
				VALUES ($1, $2), ($1, $3)
				ON CONFLICT (conversation_id, user_id) DO NOTHING
			`, conversationID, actorID, targetID); err != nil {
				return swipeResult{}, err
			}
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_low_id, user_high_id, conversation_id)
				VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid), $3)
				ON CONFLICT (user_low_id, user_high_id) DO NOTHING
				RETURNING id, created_at
			`, actorID, targetID, conversationID).Scan(&result.MatchID, &result.MatchedAt)
			switch {
			case err == nil:
				result.Matched = true
			case errors.Is(err, pgx.ErrNoRows):
				// Already matched: nothing new to announce.
			default:
				return swipeResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return swipeResult{}, err
	}
	return result, nil
}

const summaryColumns = `
	p.user_id, p.first_name, p.birthdate,
	(SELECT ph.id FROM photos ph WHERE ph.user_id = p.user_id ORDER BY ph.position LIMIT 1),
	m.id, m.created_at,
	EXISTS (SELECT 1 FROM messages msg WHERE msg.conversation_id = m.conversation_id)`

func scanSummary(row pgx.Row) (storedSummary, error) {
	var s storedSummary
	err := row.Scan(&s.UserID, &s.FirstName, &s.Birthdate, &s.PhotoID, &s.MatchID, &s.MatchedAt, &s.HasMessages)
	return s, err
}

func (r *PGRepository) Summary(ctx context.Context, viewerID, otherID string) (storedSummary, error) {
	s, err := scanSummary(r.dbPool.QueryRow(ctx, `
		SELECT `+summaryColumns+`
		FROM matches m
		JOIN profiles p ON p.user_id = $2
		WHERE m.user_low_id = LEAST($1::uuid, $2::uuid) AND m.user_high_id = GREATEST($1::uuid, $2::uuid)
	`, viewerID, otherID))
	if errors.Is(err, pgx.ErrNoRows) {
		return storedSummary{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedSummary, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+summaryColumns+`
		FROM matches m
		JOIN profiles p ON p.user_id = CASE WHEN m.user_low_id = $1 THEN m.user_high_id ELSE m.user_low_id END
		WHERE m.user_low_id = $1 OR m.user_high_id = $1
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedSummary, 0)
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Unmatch removes the match and its conversation for both members, and turns
// the requester's like into a pass so the pair cannot rematch implicitly.
func (r *PGRepository) Unmatch(ctx context.Context, userID, otherID string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := db.LockPair(ctx, tx, userID, otherID); err != nil {
		return err
	}
	var conversationID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
		RETURNING conversation_id
	`, userID, otherID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, conversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE swipes SET decision = 'pass', updated_at = NOW()
		WHERE actor_user_id = $1 AND target_user_id = $2
	`, userID, otherID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
