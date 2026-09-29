package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// pairKey is the advisory-lock key shared with the safety feature so swipes and blocks on the
// same pair are serialised. IDs are lower-cased UUID strings and ordered lexicographically.
func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return "pair:" + a + ":" + b
}

// Swipe records the swipe and, on a reciprocal like, the match, its conversation and participants,
// all in one transaction. The per-pair advisory lock serialises concurrent swipes of the same
// pair, so the second transaction always observes the first one's committed like and exactly one
// match is created.
func (r *Repository) Swipe(ctx context.Context, me, target, action string) (swipeOutcome, error) {
	var out swipeOutcome
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, pairKey(me, target)); err != nil {
		return out, err
	}

	var complete bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $1 AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))`, me).Scan(&complete); err != nil {
		return out, err
	}
	if !complete {
		return out, ErrIncompleteProfile
	}

	var discoverable, likesMe bool
	err = tx.QueryRow(ctx, `
		SELECT p.discoverable,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = p.user_id AND s.target_id = $1 AND s.action = 'like')
		FROM profiles p
		WHERE p.user_id = $2
		  AND p.latitude IS NOT NULL AND p.longitude IS NOT NULL
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))`,
		me, target).Scan(&discoverable, &likesMe)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotEligible
	}
	if err != nil {
		return out, err
	}
	if !discoverable && !likesMe {
		return out, ErrNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING`, me, target, action)
	if err != nil {
		return out, err
	}
	if tag.RowsAffected() == 0 {
		return out, ErrDuplicateSwipe
	}

	if action == ActionLike && likesMe {
		err = tx.QueryRow(ctx, `
			INSERT INTO matches (user_a, user_b)
			VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			ON CONFLICT (user_a, user_b) DO NOTHING
			RETURNING id, created_at`, me, target).Scan(&out.MatchID, &out.MatchedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		if err == nil {
			out.MatchedAt = out.MatchedAt.UTC()
			if err := tx.QueryRow(ctx, `
				INSERT INTO conversations (kind, direct_key, match_id, created_at, updated_at)
				VALUES ('match', NULL, $1, $2, $2) RETURNING id`, out.MatchID, out.MatchedAt).Scan(&out.ConversationID); err != nil {
				return out, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO conversation_participants (conversation_id, user_id, joined_at)
				VALUES ($1, $2, $4), ($1, $3, $4)`, out.ConversationID, me, target, out.MatchedAt); err != nil {
				return out, err
			}
			out.Matched = true
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return swipeOutcome{}, err
	}
	return out, nil
}

// Parties loads first name, displayed age and main photo id for the given users.
func (r *Repository) Parties(ctx context.Context, ids []string) (map[string]party, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, p.first_name,
		       CASE WHEN p.show_age AND u.birth_date IS NOT NULL THEN date_part('year', age(u.birth_date))::int END,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = u.id ORDER BY ph.position LIMIT 1)
		FROM users u JOIN profiles p ON p.user_id = u.id
		WHERE u.id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]party{}
	for rows.Next() {
		var p party
		if err := rows.Scan(&p.UserID, &p.FirstName, &p.Age, &p.PhotoID); err != nil {
			return nil, err
		}
		out[p.UserID] = p
	}
	return out, rows.Err()
}

type removedMatch struct {
	UserA, UserB   string
	ConversationID string
}

// Unmatch deletes the match if userID is a participant. The conversation, participants and
// messages go with it through ON DELETE CASCADE. Swipes stay so the pair never reappears.
func (r *Repository) Unmatch(ctx context.Context, userID, matchID string) (removedMatch, error) {
	var rm removedMatch
	err := r.pool.QueryRow(ctx, `
		DELETE FROM matches m
		WHERE m.id = $1 AND (m.user_a = $2 OR m.user_b = $2)
		RETURNING m.user_a, m.user_b, COALESCE((SELECT c.id FROM conversations c WHERE c.match_id = m.id), '00000000-0000-0000-0000-000000000000')`,
		matchID, userID).Scan(&rm.UserA, &rm.UserB, &rm.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return rm, ErrMatchNotFound
	}
	return rm, err
}
