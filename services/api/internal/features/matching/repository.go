package matching

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("profile incomplete")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	ViewerReady(ctx context.Context, userID string) (bool, error)
	CandidatePool(ctx context.Context, viewerID string, includeDemo bool, poolSize int) ([]Candidate, error)
	Swipe(ctx context.Context, organizationID, viewerID, targetID, action string) (SwipeResult, error)
	ListMatches(ctx context.Context, userID string, beforeAt *time.Time, beforeID string, limit int) ([]MatchSummary, error)
	GetMatch(ctx context.Context, userID, matchID string) (MatchSummary, error)
	DeleteMatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ViewerReady(ctx context.Context, userID string) (bool, error) {
	var ready bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		)
	`, userID).Scan(&ready)
	return ready, err
}

// haversine distance in km between candidate c and viewer v.
const distanceSQL = `(6371 * 2 * asin(sqrt(
	power(sin(radians(c.latitude - v.latitude) / 2), 2) +
	cos(radians(v.latitude)) * cos(radians(c.latitude)) *
	power(sin(radians(c.longitude - v.longitude) / 2), 2)
)))`

// CandidatePool applies every hard eligibility rule in SQL:
//   - same organization, visible, not hidden by moderation, has a photo;
//   - mutual gender and age preferences;
//   - never swiped by the viewer, no block in either direction, no match;
//   - within both users' maximum distance when the viewer has a location
//     (bounding box first so the location index can be used).
func (r *PGRepository) CandidatePool(ctx context.Context, viewerID string, includeDemo bool, poolSize int) ([]Candidate, error) {
	rows, err := r.dbPool.Query(ctx, `
		WITH v AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
			       u.organization_id,
			       pr.max_distance_km / 111.0 AS dlat,
			       pr.max_distance_km / (111.0 * GREATEST(cos(radians(COALESCE(p.latitude, 0))), 0.01)) AS dlng
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $1
		)
		SELECT c.user_id,
		       c.last_active_at,
		       CASE WHEN v.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN `+distanceSQL+` END AS distance_km,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = v.user_id
		         WHERE a.user_id = c.user_id) AS shared,
		       EXISTS (SELECT 1 FROM swipes s2
		                WHERE s2.swiper_user_id = c.user_id AND s2.target_user_id = v.user_id AND s2.action = 'like') AS liked_viewer
		FROM v
		JOIN profiles c ON c.user_id <> v.user_id
		JOIN users cu ON cu.id = c.user_id AND cu.organization_id = v.organization_id
		JOIN preferences cp ON cp.user_id = c.user_id
		WHERE c.is_visible
		  AND c.moderation_hidden_at IS NULL
		  AND ($2 OR NOT cu.is_demo)
		  AND c.gender = ANY(v.interested_in)
		  AND v.gender = ANY(cp.interested_in)
		  AND c.birthdate <= (CURRENT_DATE - make_interval(years => v.min_age))
		  AND c.birthdate >  (CURRENT_DATE - make_interval(years => v.max_age + 1))
		  AND v.birthdate <= (CURRENT_DATE - make_interval(years => cp.min_age))
		  AND v.birthdate >  (CURRENT_DATE - make_interval(years => cp.max_age + 1))
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_user_id = v.user_id AND s.target_user_id = c.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = v.user_id AND b.blocked_user_id = c.user_id)
			   OR (b.blocker_user_id = c.user_id AND b.blocked_user_id = v.user_id)
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM matches m
			WHERE m.user_low_id = LEAST(v.user_id, c.user_id)
			  AND m.user_high_id = GREATEST(v.user_id, c.user_id)
		  )
		  AND (
			v.latitude IS NULL
			OR (
				c.latitude IS NOT NULL
				AND c.latitude BETWEEN v.latitude - v.dlat AND v.latitude + v.dlat
				AND c.longitude BETWEEN v.longitude - v.dlng AND v.longitude + v.dlng
				AND `+distanceSQL+` <= LEAST(v.max_distance_km, cp.max_distance_km)
			)
		  )
		ORDER BY c.last_active_at DESC, c.user_id
		LIMIT $3
	`, viewerID, includeDemo, poolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Candidate, 0, poolSize)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.LastActiveAt, &c.DistanceKm, &c.SharedInterests, &c.LikedViewer); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Swipe records a like or pass and creates the match when the like is
// reciprocal. A per-pair advisory lock serialises concurrent swipes of the
// same two people, so simultaneous mutual likes always produce exactly one
// match (the unique pair constraint is the final safety net).
func (r *PGRepository) Swipe(ctx context.Context, organizationID, viewerID, targetID, action string) (SwipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := viewerID, targetID
	if high < low {
		low, high = high, low
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('pair:' || $1::text || ':' || $2::text, 0))`, low, high); err != nil {
		return SwipeResult{}, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $2
			  AND u.organization_id = $3
			  AND p.is_visible
			  AND p.moderation_hidden_at IS NULL
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
		)
	`, viewerID, targetID, organizationID).Scan(&eligible); err != nil {
		return SwipeResult{}, err
	}

	result := SwipeResult{Action: action, OtherUserID: targetID}

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO swipes (swiper_user_id, target_user_id, action)
		SELECT $1, $2, $3 WHERE $4::boolean
		ON CONFLICT (swiper_user_id, target_user_id) DO NOTHING
		RETURNING TRUE
	`, viewerID, targetID, action, eligible).Scan(&inserted)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// Either not eligible or already swiped: report the stored state.
		var existing string
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_user_id = $1 AND target_user_id = $2`, viewerID, targetID).Scan(&existing); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return SwipeResult{}, ErrTargetUnavailable
			}
			return SwipeResult{}, err
		}
		result.Action = existing
		result.AlreadySwiped = true
	case err != nil:
		return SwipeResult{}, err
	}

	if result.Action == ActionLike {
		var matchID string
		err := tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_low_id = $1 AND user_high_id = $2`, low, high).Scan(&matchID)
		switch {
		case err == nil:
			result.Matched = true
			result.MatchID = matchID
		case !errors.Is(err, pgx.ErrNoRows):
			return SwipeResult{}, err
		case !result.AlreadySwiped:
			var reciprocal bool
			if err := tx.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM swipes
					WHERE swiper_user_id = $1 AND target_user_id = $2 AND action = 'like'
				)
			`, targetID, viewerID).Scan(&reciprocal); err != nil {
				return SwipeResult{}, err
			}
			if reciprocal {
				matchID, err := createMatch(ctx, tx, low, high)
				if err != nil {
					return SwipeResult{}, err
				}
				result.Matched = true
				result.MatchID = matchID
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return result, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, low, high string) (string, error) {
	var matchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_low_id, user_high_id)
		VALUES ($1, $2)
		ON CONFLICT (user_low_id, user_high_id) DO UPDATE SET user_low_id = EXCLUDED.user_low_id
		RETURNING id
	`, low, high).Scan(&matchID); err != nil {
		return "", err
	}
	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, match_id, created_by_user_id)
		VALUES ('match', $1, NULL)
		ON CONFLICT (match_id) WHERE match_id IS NOT NULL DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, matchID).Scan(&conversationID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, low, high); err != nil {
		return "", err
	}
	return matchID, nil
}

const matchSummarySelect = `
	SELECT m.id, other.user_id, m.created_at,
	       COALESCE(c.last_message_at, m.created_at) AS activity_at,
	       lm.id, lm.sender_user_id, lm.content, lm.created_at,
	       (SELECT COUNT(*) FROM messages um
	         WHERE um.conversation_id = c.id
	           AND um.sender_user_id <> self.user_id
	           AND um.created_at > COALESCE(self.last_read_at, 'epoch'::timestamptz)) AS unread
	FROM conversation_participants self
	JOIN conversations c ON c.id = self.conversation_id AND c.match_id IS NOT NULL
	JOIN matches m ON m.id = c.match_id
	JOIN conversation_participants other ON other.conversation_id = c.id AND other.user_id <> self.user_id
	LEFT JOIN LATERAL (
		SELECT id, sender_user_id, content, created_at
		FROM messages
		WHERE conversation_id = c.id
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	) lm ON TRUE
	WHERE self.user_id = $1`

func scanSummary(row pgx.Row) (MatchSummary, error) {
	var s MatchSummary
	var lmID, lmSender, lmContent *string
	var lmAt *time.Time
	if err := row.Scan(&s.ID, &s.OtherUserID, &s.MatchedAt, &s.ActivityAt, &lmID, &lmSender, &lmContent, &lmAt, &s.UnreadCount); err != nil {
		return MatchSummary{}, err
	}
	if lmID != nil && lmSender != nil && lmContent != nil && lmAt != nil {
		s.LastMessage = &LastMessage{ID: *lmID, SenderUserID: *lmSender, Content: *lmContent, CreatedAt: *lmAt}
	}
	return s, nil
}

// ListMatches is the inbox: matches ordered by latest activity (last message
// or match time), keyset-paginated.
func (r *PGRepository) ListMatches(ctx context.Context, userID string, beforeAt *time.Time, beforeID string, limit int) ([]MatchSummary, error) {
	var cursorID any
	if beforeID != "" {
		cursorID = beforeID
	}
	rows, err := r.dbPool.Query(ctx, matchSummarySelect+`
		  AND ($2::timestamptz IS NULL OR (COALESCE(c.last_message_at, m.created_at), m.id) < ($2, $3::uuid))
		ORDER BY activity_at DESC, m.id DESC
		LIMIT $4
	`, userID, beforeAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MatchSummary, 0, limit)
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (MatchSummary, error) {
	s, err := scanSummary(r.dbPool.QueryRow(ctx, matchSummarySelect+` AND m.id = $2`, userID, matchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MatchSummary{}, ErrMatchNotFound
		}
		return MatchSummary{}, err
	}
	return s, nil
}

// DeleteMatch ends a match for both people; the conversation and its
// messages cascade. Swipes are kept so the profiles never resurface.
func (r *PGRepository) DeleteMatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.dbPool.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
		RETURNING CASE WHEN user_low_id = $2 THEN user_high_id ELSE user_low_id END
	`, matchID, userID).Scan(&other)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrMatchNotFound
		}
		return "", err
	}
	return other, nil
}
