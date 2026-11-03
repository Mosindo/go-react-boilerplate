package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadySwiped     = errors.New("already swiped")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrIncompleteProfile = errors.New("incomplete profile")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// discoverSQL builds candidates for viewer $1. Hard filters (mutual gender and
// age interest, distance, blocks, already-swiped, has photo) are in WHERE; the
// ordering is a pluggable score: shared interests, people who already liked the
// viewer, and proximity. Replace the score expression to evolve the algorithm.
const discoverSQL = `
SELECT ` + profiles.CardColumns + `
FROM profiles v
JOIN preferences mp ON mp.user_id = v.user_id
JOIN profiles p ON p.user_id <> v.user_id
JOIN preferences pp ON pp.user_id = p.user_id
CROSS JOIN LATERAL (
	SELECT CASE WHEN v.lat IS NOT NULL AND p.lat IS NOT NULL THEN ` + profiles.DistanceKmExpr + ` END AS d
) dist
WHERE v.user_id = $1
  AND p.discoverable
  AND p.gender = ANY(mp.interested_in)
  AND v.gender = ANY(pp.interested_in)
  AND date_part('year', age(p.birth_date)) BETWEEN mp.age_min AND mp.age_max
  AND date_part('year', age(v.birth_date)) BETWEEN pp.age_min AND pp.age_max
  AND (LEAST(mp.max_distance_km, pp.max_distance_km) >= 500
       OR (dist.d IS NOT NULL AND dist.d <= LEAST(mp.max_distance_km, pp.max_distance_km)))
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = v.user_id AND s.target_id = p.user_id)
  AND NOT EXISTS (SELECT 1 FROM blocks b
       WHERE (b.blocker_id = v.user_id AND b.blocked_id = p.user_id)
          OR (b.blocker_id = p.user_id AND b.blocked_id = v.user_id))
ORDER BY
  ( (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
      WHERE a.user_id = v.user_id AND b.user_id = p.user_id) * 10
    + CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.actor_id = p.user_id AND s.target_id = v.user_id AND s.action = 'like') THEN 15 ELSE 0 END
    - COALESCE(dist.d, 0) * 0.05 ) DESC,
  md5(p.user_id::text || current_date::text)
LIMIT $2`

func (r *Repository) Discover(ctx context.Context, viewerID string, limit int) ([]profiles.Card, error) {
	rows, err := r.db.Query(ctx, discoverSQL, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := []profiles.Card{}
	for rows.Next() {
		card, err := profiles.ScanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func pairKey(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Swipe records a decision atomically. A pair-level advisory lock serialises
// two users liking each other at the same instant, so exactly one match row
// (and conversation) is ever produced.
func (r *Repository) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lo, hi := pairKey(actorID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 2))`, lo+hi); err != nil {
		return SwipeResult{}, err
	}

	var ready bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, actorID).Scan(&ready); err != nil {
		return SwipeResult{}, err
	}
	if !ready {
		return SwipeResult{}, ErrIncompleteProfile
	}

	var visible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles p WHERE p.user_id = $2 AND `+profiles.VisibleToViewer+`)`,
		actorID, targetID).Scan(&visible); err != nil {
		return SwipeResult{}, err
	}
	if !visible {
		return SwipeResult{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (actor_id, target_id) DO NOTHING`, actorID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrAlreadySwiped
	}

	result := SwipeResult{}
	if action == "like" {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_id = $1 AND target_id = $2 AND action = 'like')`,
			targetID, actorID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			result, err = createMatch(ctx, tx, lo, hi)
			if err != nil {
				return SwipeResult{}, err
			}
		}
	}
	return result, tx.Commit(ctx)
}

// createMatch inserts the match, its conversation, both participants and a
// notification per user in the caller's transaction.
func createMatch(ctx context.Context, tx pgx.Tx, userA, userB string) (SwipeResult, error) {
	var res SwipeResult
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
		RETURNING id`, userA, userB).Scan(&res.MatchID)
	if err != nil {
		return res, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id`, res.MatchID).Scan(&res.ConversationID)
	if err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, res.ConversationID, userA, userB); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id)
		VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)`, userA, userB, res.MatchID); err != nil {
		return res, err
	}
	res.Matched = true
	return res, nil
}

func (r *Repository) ListMatches(ctx context.Context, viewerID string, before time.Time, limit int) ([]Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profiles.CardColumns+`, m.id, c.id, m.created_at
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE (m.user_a = $1 OR m.user_b = $1) AND m.created_at < $2
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3`, viewerID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Match{}
	for rows.Next() {
		var m Match
		card, err := profiles.ScanCard(rows, &m.ID, &m.ConversationID, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		m.User = card
		out = append(out, m)
	}
	return out, rows.Err()
}

// Unmatch deletes the match (and its conversation by cascade). Swipe rows are
// kept so the pair never resurfaces in discovery.
func (r *Repository) Unmatch(ctx context.Context, userID, matchID string) (otherID string, err error) {
	var a, b string
	err = r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)
		RETURNING user_a, user_b`, matchID, userID).Scan(&a, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	if err != nil {
		return "", err
	}
	if a == userID {
		return b, nil
	}
	return a, nil
}

// LikesReceived lists people who liked the viewer and are still undecided.
func (r *Repository) LikesReceived(ctx context.Context, viewerID string, before time.Time, limit int) ([]LikeReceived, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profiles.CardColumns+`, s.created_at
		FROM swipes s
		JOIN profiles p ON p.user_id = s.actor_id
		LEFT JOIN profiles v ON v.user_id = $1
		WHERE s.target_id = $1 AND s.action = 'like' AND s.created_at < $2
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.actor_id = $1 AND mine.target_id = s.actor_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		       WHERE (b.blocker_id = $1 AND b.blocked_id = s.actor_id) OR (b.blocker_id = s.actor_id AND b.blocked_id = $1))
		ORDER BY s.created_at DESC
		LIMIT $3`, viewerID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LikeReceived{}
	for rows.Next() {
		var l LikeReceived
		card, err := profiles.ScanCard(rows, &l.LikedAt)
		if err != nil {
			return nil, err
		}
		l.User = card
		out = append(out, l)
	}
	return out, rows.Err()
}
