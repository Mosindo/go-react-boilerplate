package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile before discovering people")
	ErrTargetUnavailable = errors.New("profile unavailable")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	Viewer(ctx context.Context, userID string) (viewer, error)
	Candidates(ctx context.Context, userID string, v viewer, targetID string, limit int) ([]profiles.StoredPublic, error)
	Swipe(ctx context.Context, userID string, v viewer, targetID, action string) (swipeResult, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	PhotoIDs(ctx context.Context, ids []string) (map[string][]string, error)
	Interests(ctx context.Context, ids []string) (map[string][]profiles.Interest, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Viewer(ctx context.Context, userID string) (viewer, error) {
	var v viewer
	var birthYearAge int
	err := r.pool.QueryRow(ctx, `
		SELECT p.gender, date_part('year', age(p.birth_date))::int,
		       COALESCE(pr.interested_in, '{}'), COALESCE(pr.min_age, 18), COALESCE(pr.max_age, 99), COALESCE(pr.max_distance_km, 50),
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id),
		       p.latitude IS NOT NULL
		FROM profiles p
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1`, userID).Scan(&v.Gender, &birthYearAge,
		&v.Preferences.InterestedIn, &v.Preferences.MinAge, &v.Preferences.MaxAge, &v.Preferences.MaxDistanceKm,
		&v.HasPhoto, &v.HasLocation)
	if errors.Is(err, pgx.ErrNoRows) {
		return viewer{}, ErrProfileIncomplete
	}
	v.Age = birthYearAge
	if err != nil {
		return viewer{}, err
	}
	if !v.HasPhoto || !v.HasLocation || len(v.Preferences.InterestedIn) == 0 {
		return viewer{}, ErrProfileIncomplete
	}
	return v, nil
}

// candidateQuery applies every eligibility rule on the server: mutual gender
// interest, mutual age range, mutual distance, visibility, not already
// swiped, no block in either direction. Ranking favours people who already
// liked the viewer, then shared interests, then proximity and recent activity.
// $9 optionally restricts the query to one target (used to authorize swipes).
const candidateQuery = `
	SELECT t.user_id::text, t.display_name, t.birth_date, t.gender, t.bio, t.city,
	       CASE WHEN t.show_distance THEN d.km END
	FROM profiles t
	JOIN preferences tp ON tp.user_id = t.user_id
	JOIN profiles me ON me.user_id = $1
	CROSS JOIN LATERAL (
	  SELECT 6371 * 2 * asin(sqrt(
	    power(sin(radians(t.latitude - me.latitude) / 2), 2) +
	    cos(radians(me.latitude)) * cos(radians(t.latitude)) *
	    power(sin(radians(t.longitude - me.longitude) / 2), 2))) AS km
	) d
	WHERE t.user_id <> $1
	  AND ($9::uuid IS NULL OR t.user_id = $9::uuid)
	  AND t.is_discoverable
	  AND t.latitude IS NOT NULL
	  AND t.gender = ANY($2::text[])
	  AND $3 = ANY(tp.interested_in)
	  AND date_part('year', age(t.birth_date)) BETWEEN $4 AND $5
	  AND $6 BETWEEN tp.min_age AND tp.max_age
	  AND d.km <= $7 AND d.km <= tp.max_distance_km
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = $1 AND s.target_id = t.user_id)
	  AND NOT EXISTS (
	    SELECT 1 FROM blocks b
	    WHERE (b.blocker_id = $1 AND b.blocked_id = t.user_id) OR (b.blocker_id = t.user_id AND b.blocked_id = $1))
	ORDER BY
	  EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = t.user_id AND s.target_id = $1 AND s.action = 'like') DESC,
	  (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
	    WHERE a.user_id = $1 AND b.user_id = t.user_id) DESC,
	  d.km ASC,
	  t.last_active_at DESC,
	  t.user_id
	LIMIT $8`

func (r *PGRepository) Candidates(ctx context.Context, userID string, v viewer, targetID string, limit int) ([]profiles.StoredPublic, error) {
	return queryCandidates(ctx, r.pool, userID, v, targetID, limit)
}

func queryCandidates(ctx context.Context, q profiles.Querier, userID string, v viewer, targetID string, limit int) ([]profiles.StoredPublic, error) {
	var target any
	if targetID != "" {
		target = targetID
	}
	rows, err := q.Query(ctx, candidateQuery, userID, v.Preferences.InterestedIn, v.Gender,
		v.Preferences.MinAge, v.Preferences.MaxAge, v.Age, v.Preferences.MaxDistanceKm, limit, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []profiles.StoredPublic
	for rows.Next() {
		var p profiles.StoredPublic
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.BirthDate, &p.Gender, &p.Bio, &p.City, &p.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Swipe records a decision exactly once and creates the match (with its
// conversation and notifications) when the interest is mutual. A per-pair
// advisory lock makes simultaneous likes from both sides produce one match.
func (r *PGRepository) Swipe(ctx context.Context, userID string, v viewer, targetID, action string) (swipeResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return swipeResult{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, userID, targetID); err != nil {
		return swipeResult{}, err
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks
		  WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))`, userID, targetID).Scan(&blocked); err != nil {
		return swipeResult{}, err
	}
	if blocked {
		return swipeResult{}, ErrTargetUnavailable
	}

	res := swipeResult{Action: action}
	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE user_id = $1 AND target_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		res.Action, res.Duplicate = existing, true
		if existing == ActionLike {
			res.MatchID, res.ConversationID = lookupMatch(ctx, tx, userID, targetID)
		}
		return res, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return swipeResult{}, err
	}

	eligible, err := queryCandidates(ctx, tx, userID, v, targetID, 1)
	if err != nil {
		return swipeResult{}, err
	}
	if len(eligible) == 0 {
		return swipeResult{}, ErrTargetUnavailable
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (user_id, target_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return swipeResult{}, err
	}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE user_id = $1 AND target_id = $2 AND action = 'like')`, targetID, userID).Scan(&reciprocal); err != nil {
			return swipeResult{}, err
		}
		if reciprocal {
			if err := createMatch(ctx, tx, userID, targetID, &res); err != nil {
				return swipeResult{}, err
			}
		}
	}
	return res, tx.Commit(ctx)
}

func lookupMatch(ctx context.Context, q profiles.Querier, a, b string) (matchID, conversationID string) {
	_ = q.QueryRow(ctx, `
		SELECT m.id::text, c.id::text FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)`, a, b).Scan(&matchID, &conversationID)
	return matchID, conversationID
}

func createMatch(ctx context.Context, tx pgx.Tx, a, b string, res *swipeResult) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id::text`, a, b).Scan(&res.MatchID)
	if errors.Is(err, pgx.ErrNoRows) {
		res.MatchID, res.ConversationID = lookupMatch(ctx, tx, a, b)
		return nil
	}
	if err != nil {
		return err
	}
	res.NewMatch = true

	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id::text`, res.MatchID).Scan(&res.ConversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, res.ConversationID, a, b); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT me.user_id, 'match', 'C''est un match !', 'Vous avez un match avec ' || other.display_name || '.',
		       jsonb_build_object('matchId', $3::text, 'conversationId', $4::text, 'userId', other.user_id::text)
		FROM profiles me JOIN profiles other ON other.user_id <> me.user_id
		WHERE me.user_id IN ($1::uuid, $2::uuid) AND other.user_id IN ($1::uuid, $2::uuid)`, a, b, res.MatchID, res.ConversationID)
	return err
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var userA, userB string
	err := r.pool.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)
		RETURNING user_a::text, user_b::text`, matchID, userID).Scan(&userA, &userB)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	if err != nil {
		return "", err
	}
	if userA == userID {
		return userB, nil
	}
	return userA, nil
}

func (r *PGRepository) PhotoIDs(ctx context.Context, ids []string) (map[string][]string, error) {
	return profiles.LoadPhotoIDs(ctx, r.pool, ids)
}

func (r *PGRepository) Interests(ctx context.Context, ids []string) (map[string][]profiles.Interest, error) {
	return profiles.LoadInterests(ctx, r.pool, ids)
}
