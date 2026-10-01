package matching

import (
	"context"
	"errors"
	"fmt"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotEligible   = errors.New("profile is not available")
	ErrMatchNotFound = errors.New("match not found")
)

type Repository interface {
	Touch(ctx context.Context, userID string) error
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Eligible(ctx context.Context, userID, targetID string) (bool, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchRow, error)
	GetMatchFor(ctx context.Context, userID, matchID string) (MatchRow, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Touch(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND (last_active_at IS NULL OR last_active_at < NOW() - INTERVAL '5 minutes')`, userID)
	return err
}

// candidateScore is THE extension point of the recommendation algorithm.
// Higher is shown first. Today: shared interests, people who already liked
// the viewer, recent activity, proximity, plus a little jitter for variety.
// Swap this expression (or move ranking out of SQL) without touching filters.
const candidateScore = `(
	3 * (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
	     WHERE a.user_id = me.user_id AND b.user_id = c.user_id)
	+ CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = c.user_id AND s.to_user_id = me.user_id AND s.action = 'like') THEN 5 ELSE 0 END
	+ CASE WHEN cu.last_active_at > NOW() - INTERVAL '7 days' THEN 2 ELSE 0 END
	- COALESCE(d.km, 0) / 25.0
	+ random() * 1.5
)`

// eligibleFrom is the single source of truth for "viewer may be shown candidate":
// profile complete and discoverable, mutual gender/age/distance preferences,
// accounts active, nobody blocked. extra narrows it (swipe-history exclusion,
// target id...).
func eligibleFrom(extra string) string {
	return fmt.Sprintf(`
	WITH me AS (
		SELECT p.user_id, p.gender, date_part('year', age(p.birth_date))::int AS age, p.latitude, p.longitude,
		       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
		FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	)
	SELECT c.user_id, %s AS score
	FROM me
	JOIN profiles c ON c.user_id <> me.user_id
	JOIN users cu ON cu.id = c.user_id AND cu.status = 'active'
	JOIN preferences cp ON cp.user_id = c.user_id
	CROSS JOIN LATERAL (SELECT date_part('year', age(c.birth_date))::int AS age) ca
	CROSS JOIN LATERAL (SELECT CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL
	                                THEN %s END AS km) d
	WHERE c.discoverable
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND c.gender = ANY(me.interested_in)
	  AND ca.age BETWEEN me.age_min AND me.age_max
	  AND me.gender = ANY(cp.interested_in)
	  AND me.age BETWEEN cp.age_min AND cp.age_max
	  AND (me.latitude IS NULL OR (c.latitude IS NOT NULL AND d.km <= LEAST(me.max_distance_km, cp.max_distance_km)))
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	                  WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	                     OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  %s`,
		candidateScore,
		profiles.DistanceKmSQL("me.latitude", "me.longitude", "c.latitude", "c.longitude"),
		extra)
}

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	query := eligibleFrom(`AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)`) +
		` ORDER BY score DESC, c.user_id LIMIT $2`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		var score float64
		if err := rows.Scan(&id, &score); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PGRepository) Eligible(ctx context.Context, userID, targetID string) (bool, error) {
	rows, err := r.pool.Query(ctx, eligibleFrom(`AND c.user_id = $2`)+` LIMIT 1`, userID, targetID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	found := rows.Next()
	return found, rows.Err()
}

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	a, b := userID, targetID
	if a > b {
		a, b = b, a
	}
	// Serialize concurrent swipes between the same pair so that two simultaneous
	// likes can never both miss each other: exactly one creates the match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, a+":"+b); err != nil {
		return SwipeResult{}, err
	}

	res := SwipeResult{Action: action}
	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING`, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	res.Inserted = tag.RowsAffected() == 1
	if !res.Inserted {
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&res.Action); err != nil {
			return SwipeResult{}, err
		}
	}

	if res.Action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $2 AND to_user_id = $1 AND action = 'like')`,
			userID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
				ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = matches.user_a
				RETURNING id, created_at`, a, b).Scan(&res.MatchID, &res.MatchedAt)
			if err != nil {
				return SwipeResult{}, err
			}
			if err := ensureConversation(ctx, tx, res.MatchID, a, b, &res.ConversationID); err != nil {
				return SwipeResult{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return res, nil
}

func ensureConversation(ctx context.Context, tx pgx.Tx, matchID, a, b string, out *string) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, match_id) VALUES ('match', $1)
		ON CONFLICT (match_id) WHERE match_id IS NOT NULL DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id`, matchID).Scan(out)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING`, *out, a, b)
	return err
}

const matchSelect = `
	SELECT m.id, m.created_at,
	       CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END,
	       conv.id,
	       EXISTS (SELECT 1 FROM messages msg WHERE msg.conversation_id = conv.id)
	FROM matches m
	JOIN conversations conv ON conv.match_id = m.id
	WHERE (m.user_a = $1 OR m.user_b = $1)
	  AND m.unmatched_at IS NULL
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	                  WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
	                     OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))`

func scanMatches(rows pgx.Rows) ([]MatchRow, error) {
	defer rows.Close()
	out := make([]MatchRow, 0, 16)
	for rows.Next() {
		var m MatchRow
		if err := rows.Scan(&m.ID, &m.CreatedAt, &m.OtherUserID, &m.ConversationID, &m.HasMessages); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchRow, error) {
	rows, err := r.pool.Query(ctx, matchSelect+` ORDER BY m.created_at DESC, m.id LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return scanMatches(rows)
}

func (r *PGRepository) GetMatchFor(ctx context.Context, userID, matchID string) (MatchRow, error) {
	rows, err := r.pool.Query(ctx, matchSelect+` AND m.id = $2`, userID, matchID)
	if err != nil {
		return MatchRow{}, err
	}
	list, err := scanMatches(rows)
	if err != nil {
		return MatchRow{}, err
	}
	if len(list) == 0 {
		return MatchRow{}, ErrMatchNotFound
	}
	return list[0], nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.pool.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
		WHERE id = $2 AND (user_a = $1 OR user_b = $1) AND unmatched_at IS NULL
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}
