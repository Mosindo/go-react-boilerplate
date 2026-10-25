package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// candidateFrom is the single definition of who may be shown to user $1: both sides' preferences
// (gender, age, distance) must be satisfied, nobody blocked anybody, and the candidate is discoverable
// with at least one photo. Shared by the feed and by swipe validation so the rules cannot diverge.
var candidateFrom = `
	FROM profiles p
	JOIN preferences pr ON pr.user_id = p.user_id
	JOIN users u ON u.id = p.user_id
	JOIN profiles mp ON mp.user_id = $1::uuid
	JOIN preferences mpr ON mpr.user_id = $1::uuid
	CROSS JOIN LATERAL (SELECT ` + db.DistanceKm("mp.latitude", "mp.longitude", "p.latitude", "p.longitude") + ` AS km) d
	WHERE p.user_id <> $1::uuid
	  AND p.discoverable
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	  AND p.gender = ANY(mpr.interested_in)
	  AND mp.gender = ANY(pr.interested_in)
	  AND ` + db.Age("p.birth_date") + ` BETWEEN mpr.min_age AND mpr.max_age
	  AND ` + db.Age("mp.birth_date") + ` BETWEEN pr.min_age AND pr.max_age
	  AND (mpr.max_distance_km IS NULL OR (d.km IS NOT NULL AND d.km <= mpr.max_distance_km))
	  AND (pr.max_distance_km IS NULL OR (d.km IS NOT NULL AND d.km <= pr.max_distance_km))
	  AND ` + db.NotBlocked("$1::uuid", "p.user_id")

// feedScore is the recommendation hook: a plain SQL expression, easy to replace by a better ranker.
// Shared interests and people who already liked the viewer rank first, recent activity breaks ties.
const feedScore = `
	  2 * (SELECT COUNT(*) FROM user_interests a
	        JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = p.user_id
	        WHERE a.user_id = $1::uuid)
	+ CASE WHEN EXISTS (SELECT 1 FROM swipes lk WHERE lk.from_user_id = p.user_id AND lk.to_user_id = $1::uuid AND lk.action = 'like') THEN 5 ELSE 0 END
	+ CASE WHEN u.last_active_at > NOW() - INTERVAL '7 days' THEN 2 ELSE 0 END`

func (r *Repository) feed(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id `+candidateFrom+`
		  AND `+db.ProfileComplete("$1::uuid")+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = $1::uuid AND s.to_user_id = p.user_id)
		ORDER BY (`+feedScore+`) DESC, md5($1::text || p.user_id::text)
		LIMIT $2
	`, userID, limit)
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

// swipe records the decision atomically. A per-pair advisory lock serializes concurrent swipes of
// the two users, so a mutual like can never be missed by both transactions.
func (r *Repository) swipe(ctx context.Context, userID, targetID string, action Action) (outcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return outcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::uuid, $2::uuid)::text || GREATEST($1::uuid, $2::uuid)::text, 0))
	`, userID, targetID); err != nil {
		return outcome{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		return outcome{Result: ResultAlreadySwiped}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return outcome{}, err
	}

	var complete bool
	if err := tx.QueryRow(ctx, `SELECT `+db.ProfileComplete("$1::uuid"), userID).Scan(&complete); err != nil {
		return outcome{}, err
	}
	if !complete {
		return outcome{}, ErrProfileIncomplete
	}
	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 `+candidateFrom+` AND p.user_id = $2::uuid)`, userID, targetID).Scan(&eligible); err != nil {
		return outcome{}, err
	}
	if !eligible {
		return outcome{}, ErrTargetUnavailable
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, userID, targetID, string(action)); err != nil {
		return outcome{}, err
	}
	if action == ActionPass {
		return outcome{Result: ResultPassed}, tx.Commit(ctx)
	}

	var mutual bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')
	`, targetID, userID).Scan(&mutual); err != nil {
		return outcome{}, err
	}
	if !mutual {
		return outcome{Result: ResultLiked}, tx.Commit(ctx)
	}

	out := outcome{Result: ResultMatched}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id, created_at
	`, userID, targetID).Scan(&out.MatchID, &out.MatchCreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already matched (defensive): report the swipe as a plain like.
		return outcome{Result: ResultLiked}, tx.Commit(ctx)
	}
	if err != nil {
		return outcome{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, out.MatchID).Scan(&out.ConversationID); err != nil {
		return outcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, out.ConversationID, userID, targetID); err != nil {
		return outcome{}, err
	}
	// The swiper sees the match immediately, so their notification is stored as already read.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, conversation_id, read_at) VALUES ($1, 'match', $2, $3, NOW())
	`, userID, targetID, out.ConversationID); err != nil {
		return outcome{}, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, conversation_id) VALUES ($1, 'match', $2, $3) RETURNING id
	`, targetID, userID, out.ConversationID).Scan(&out.TargetNotificationID); err != nil {
		return outcome{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Repository) touch(ctx context.Context, userID string) {
	_, _ = r.pool.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
}
