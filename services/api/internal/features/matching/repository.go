package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("profile not found")
	ErrProfileIncomplete = errors.New("complete your profile before swiping")
	ErrSelf              = errors.New("cannot swipe yourself")
)

type Repository interface {
	Feed(ctx context.Context, userID string, limit int) ([]string, error)
	Touch(ctx context.Context, userID string) error
	ReceivedLikes(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, userID, targetID, kind string) (SwipeOutcome, error)
	ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// feedSQL selects the next candidates for a user. Eligibility is mutual:
// both sides' gender, age and distance preferences must be satisfied, the
// candidate must be discoverable with at least one photo, and anyone already
// swiped on, blocked (either direction) or matched is excluded. The ordering
// is the single place to evolve the recommendation algorithm: people who
// already liked the viewer come first, then the most recently active.
const feedSQL = `
WITH me AS (
  SELECT u.id, u.birth_date, p.gender, p.latitude, p.longitude, pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
  FROM users u
  JOIN profiles p ON p.user_id = u.id
  JOIN preferences pr ON pr.user_id = u.id
  WHERE u.id = $1 AND p.gender IS NOT NULL
)
SELECT c.user_id
FROM me
JOIN profiles c ON c.user_id <> me.id
JOIN users cu ON cu.id = c.user_id
JOIN preferences cp ON cp.user_id = c.user_id
CROSS JOIN LATERAL (
  SELECT CASE WHEN c.latitude IS NOT NULL AND me.latitude IS NOT NULL THEN
    2 * 6371 * asin(sqrt(
      power(sin(radians(c.latitude - me.latitude) / 2), 2) +
      cos(radians(me.latitude)) * cos(radians(c.latitude)) * power(sin(radians(c.longitude - me.longitude) / 2), 2)))
  END AS km
) d
WHERE c.is_discoverable
  AND c.gender = ANY(me.interested_in)
  AND me.gender = ANY(cp.interested_in)
  AND date_part('year', age(cu.birth_date)) BETWEEN me.age_min AND me.age_max
  AND date_part('year', age(me.birth_date)) BETWEEN cp.age_min AND cp.age_max
  AND (d.km IS NULL OR (d.km <= me.max_distance_km AND d.km <= cp.max_distance_km))
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM likes l WHERE l.swiper_id = me.id AND l.target_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = me.id AND b.blocked_id = c.user_id) OR (b.blocker_id = c.user_id AND b.blocked_id = me.id))
ORDER BY EXISTS (SELECT 1 FROM likes l2 WHERE l2.swiper_id = c.user_id AND l2.target_id = me.id AND l2.kind = 'like') DESC,
         cu.last_active_at DESC, c.user_id
LIMIT $2`

// Touch records activity; it feeds the recency ordering of other users' feeds.
func (r *PGRepository) Touch(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
	return err
}

func (r *PGRepository) Feed(ctx context.Context, userID string, limit int) ([]string, error) {
	return r.ids(ctx, feedSQL, userID, limit)
}

func (r *PGRepository) ReceivedLikes(ctx context.Context, userID string, limit int) ([]string, error) {
	return r.ids(ctx, `
		SELECT l.swiper_id FROM likes l
		JOIN profiles p ON p.user_id = l.swiper_id
		WHERE l.target_id = $1 AND l.kind = 'like'
		  AND NOT EXISTS (SELECT 1 FROM likes mine WHERE mine.swiper_id = $1 AND mine.target_id = l.swiper_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = l.swiper_id) OR (b.blocker_id = l.swiper_id AND b.blocked_id = $1))
		ORDER BY l.created_at DESC
		LIMIT $2`, userID, limit)
}

func (r *PGRepository) ids(ctx context.Context, sql string, args ...any) ([]string, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, kind string) (SwipeOutcome, error) {
	if userID == targetID {
		return SwipeOutcome{}, ErrSelf
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := orderPair(userID, targetID)
	// Serialize concurrent swipes on the same pair so two simultaneous
	// reciprocal likes always produce exactly one match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "pair:"+low+":"+high); err != nil {
		return SwipeOutcome{}, err
	}

	var myName string
	var complete bool
	err = tx.QueryRow(ctx, `
		SELECT p.first_name,
		       p.gender IS NOT NULL
		       AND EXISTS (SELECT 1 FROM photos WHERE user_id = p.user_id)
		       AND EXISTS (SELECT 1 FROM preferences WHERE user_id = p.user_id AND cardinality(interested_in) > 0)
		FROM profiles p WHERE p.user_id = $1`, userID).Scan(&myName, &complete)
	if errors.Is(err, pgx.ErrNoRows) || !complete && err == nil {
		return SwipeOutcome{}, ErrProfileIncomplete
	}
	if err != nil {
		return SwipeOutcome{}, err
	}

	var targetName string
	err = tx.QueryRow(ctx, `
		SELECT p.first_name FROM profiles p
		WHERE p.user_id = $2
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (p.is_discoverable OR EXISTS (SELECT 1 FROM likes l WHERE l.swiper_id = $2 AND l.target_id = $1 AND l.kind = 'like'))`,
		userID, targetID).Scan(&targetName)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return SwipeOutcome{}, ErrNotFound
	}
	if err != nil {
		return SwipeOutcome{}, err
	}

	tag, err := tx.Exec(ctx, `INSERT INTO likes (swiper_id, target_id, kind) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, userID, targetID, kind)
	if err != nil {
		return SwipeOutcome{}, err
	}
	if tag.RowsAffected() == 0 {
		// Repeated swipe: the first decision stands. Report the existing match, if any.
		m, err := existingMatch(ctx, tx, userID, low, high)
		if err != nil {
			return SwipeOutcome{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return SwipeOutcome{}, err
		}
		return SwipeOutcome{AlreadyHandled: true, Match: m}, nil
	}
	if kind != KindLike {
		return SwipeOutcome{}, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM likes WHERE swiper_id = $1 AND target_id = $2 AND kind = 'like')`, targetID, userID).Scan(&reciprocal); err != nil {
		return SwipeOutcome{}, err
	}
	if !reciprocal {
		if err := notifications.Insert(ctx, tx, targetID, notifications.TypeLike, "Quelqu'un vous a liké",
			"Découvrez qui s'intéresse à vous.", nil); err != nil {
			return SwipeOutcome{}, err
		}
		return SwipeOutcome{}, tx.Commit(ctx)
	}

	var matchID, convID string
	var createdAt time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_low, user_high) VALUES ($1,$2)
		ON CONFLICT (user_low, user_high) DO UPDATE SET ended_at = NULL, created_at = NOW() WHERE matches.ended_at IS NOT NULL
		RETURNING id, created_at`, low, high).Scan(&matchID, &createdAt); err != nil {
		return SwipeOutcome{}, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id RETURNING id`, matchID).Scan(&convID); err != nil {
		return SwipeOutcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1,$2), ($1,$3)
		ON CONFLICT DO NOTHING`, convID, userID, targetID); err != nil {
		return SwipeOutcome{}, err
	}
	data := map[string]string{"matchId": matchID, "conversationId": convID}
	if err := notifications.Insert(ctx, tx, targetID, notifications.TypeMatch, "C'est un match !", "Vous avez un match avec "+myName+".", withUser(data, userID)); err != nil {
		return SwipeOutcome{}, err
	}
	if err := notifications.Insert(ctx, tx, userID, notifications.TypeMatch, "C'est un match !", "Vous avez un match avec "+targetName+".", withUser(data, targetID)); err != nil {
		return SwipeOutcome{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeOutcome{}, err
	}
	return SwipeOutcome{Match: &StoredMatch{ID: matchID, ConversationID: convID, OtherUserID: targetID, CreatedAt: createdAt}}, nil
}

func withUser(data map[string]string, userID string) map[string]string {
	out := map[string]string{"userId": userID}
	for k, v := range data {
		out[k] = v
	}
	return out
}

func existingMatch(ctx context.Context, tx pgx.Tx, userID, low, high string) (*StoredMatch, error) {
	var m StoredMatch
	err := tx.QueryRow(ctx, `
		SELECT m.id, c.id, m.created_at FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_low = $1 AND m.user_high = $2 AND m.ended_at IS NULL`, low, high).Scan(&m.ID, &m.ConversationID, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if userID == low {
		m.OtherUserID = high
	} else {
		m.OtherUserID = low
	}
	return &m, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, CASE WHEN m.user_low = $1 THEN m.user_high ELSE m.user_low END, m.created_at
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_low = $1 OR m.user_high = $1)
		  AND m.ended_at IS NULL
		  AND ($2::timestamptz IS NULL OR m.created_at < $2)
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredMatch{}
	for rows.Next() {
		var m StoredMatch
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherUserID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET ended_at = NOW()
		WHERE id = $1 AND ended_at IS NULL AND (user_low = $2 OR user_high = $2)
		RETURNING CASE WHEN user_low = $2 THEN user_high ELSE user_low END`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", ErrNotFound
	}
	return other, err
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
