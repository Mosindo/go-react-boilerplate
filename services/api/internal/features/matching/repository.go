package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	LoadViewer(ctx context.Context, userID string) (Viewer, error)
	TouchActivity(ctx context.Context, userID string) error
	Candidates(ctx context.Context, q CandidateQuery) ([]Candidate, error)
	Swipe(ctx context.Context, swiperID, targetID, decision string) (SwipeResult, error)
	LikesReceived(ctx context.Context, userID string, limit int, includeDemo bool) ([]string, error)
	Unmatch(ctx context.Context, userID, matchID string) (partnerID, conversationID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) LoadViewer(ctx context.Context, userID string) (Viewer, error) {
	v := Viewer{UserID: userID}
	err := r.db.QueryRow(ctx, `
		SELECT p.gender, p.birthdate, p.latitude, p.longitude,
		       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&v.Gender, &v.Birthdate, &v.Latitude, &v.Longitude, &v.InterestedIn, &v.AgeMin, &v.AgeMax, &v.MaxDistanceKm, &v.HasPhoto)
	if errors.Is(err, pgx.ErrNoRows) {
		return Viewer{}, ErrProfileIncomplete
	}
	return v, err
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

// Candidates returns a bounded pool of eligible profiles. Every rule that matters
// for safety and consent is enforced here, server-side:
//   - mutual gender preferences and mutual age ranges;
//   - distance within both users' radius (when the viewer shared a location);
//   - discoverable profiles with at least one photo;
//   - never the viewer, never someone already swiped, never a blocked/blocking user.
func (r *PGRepository) Candidates(ctx context.Context, q CandidateQuery) ([]Candidate, error) {
	v := q.Viewer
	viewerAge := profiles.AgeAt(v.Birthdate, q.Now)
	// Birthdate window derived from the viewer's age range.
	youngest := q.Now.AddDate(-v.AgeMin, 0, 0)
	oldest := q.Now.AddDate(-(v.AgeMax + 1), 0, 1)

	useDistance := v.Latitude != nil && v.Longitude != nil
	var lat, lng, minLat, maxLat, minLng, maxLng float64
	wrapLng := false
	if useDistance {
		lat, lng = *v.Latitude, *v.Longitude
		minLat, maxLat, minLng, maxLng = geo.BoundingBox(lat, lng, float64(v.MaxDistanceKm))
		wrapLng = minLng < -180 || maxLng > 180
	}

	rows, err := r.db.Query(ctx, `
		WITH pool AS (
			SELECT p.user_id, p.last_active_at, p.latitude, p.longitude, pr.max_distance_km AS their_radius
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id <> $1
			  AND ($13 OR NOT u.is_demo)
			  AND pr.is_discoverable
			  AND p.gender = ANY($2::text[])
			  AND $3 = ANY(pr.interested_in)
			  AND p.birthdate <= $4 AND p.birthdate >= $5
			  AND $6 BETWEEN pr.age_min AND pr.age_max
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
				   OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
			  )
			  AND (
				NOT $7::boolean
				OR (
					p.latitude BETWEEN $8 AND $9
					AND ($14::boolean OR p.longitude BETWEEN $10 AND $11)
				)
			  )
		), measured AS (
			SELECT pool.*,
				CASE WHEN $7::boolean AND pool.latitude IS NOT NULL THEN
					2 * 6371 * asin(sqrt(
						power(sin(radians(pool.latitude - $15) / 2), 2) +
						cos(radians($15)) * cos(radians(pool.latitude)) *
						power(sin(radians(pool.longitude - $16) / 2), 2)
					))
				END AS distance_km
			FROM pool
		)
		SELECT m.user_id, m.distance_km, m.last_active_at,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = $1
		         WHERE a.user_id = m.user_id) AS shared,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = m.user_id AND s.target_id = $1 AND s.decision = 'like') AS liked_viewer
		FROM measured m
		WHERE NOT $7::boolean OR (m.distance_km <= $17 AND m.distance_km <= m.their_radius)
		ORDER BY m.last_active_at DESC, m.user_id
		LIMIT $12
	`,
		v.UserID, v.InterestedIn, v.Gender, youngest, oldest, viewerAge,
		useDistance, minLat, maxLat, minLng, maxLng, q.Limit, q.IncludeDemo, wrapLng,
		lat, lng, v.MaxDistanceKm,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.DistanceKm, &c.LastActiveAt, &c.SharedInterests, &c.LikedViewer)
		return c, err
	})
}

// Swipe records a decision and atomically creates the match (plus its conversation)
// when the like is reciprocal. A per-pair advisory lock prevents two simultaneous
// mutual likes from both missing each other.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, decision string) (SwipeResult, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SwipeResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
	`, swiperID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $2 AND pr.is_discoverable
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
		)
	`, swiperID, targetID).Scan(&available); err != nil {
		return SwipeResult{}, err
	}
	if !available {
		return SwipeResult{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, decision) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, decision)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrAlreadySwiped
	}

	result := SwipeResult{}
	if decision == DecisionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND decision = 'like')
		`, swiperID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a_id, user_b_id)
				VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a_id, user_b_id) DO NOTHING
				RETURNING id
			`, swiperID, targetID).Scan(&result.MatchID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return SwipeResult{}, err
			}
			if result.MatchID != "" {
				if err := tx.QueryRow(ctx, `
					INSERT INTO conversations (match_id) VALUES ($1) RETURNING id
				`, result.MatchID).Scan(&result.ConversationID); err != nil {
					return SwipeResult{}, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
				`, result.ConversationID, swiperID, targetID); err != nil {
					return SwipeResult{}, err
				}
				result.Matched = true
			}
		}
	}

	return result, tx.Commit(ctx)
}

// LikesReceived lists people who liked the user and are still waiting for an answer.
// Seeing who liked you is free for everyone.
func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit int, includeDemo bool) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.swiper_id
		FROM swipes s
		JOIN users u ON u.id = s.swiper_id
		JOIN preferences pr ON pr.user_id = s.swiper_id
		WHERE s.target_id = $1 AND s.decision = 'like'
		  AND ($3 OR NOT u.is_demo)
		  AND pr.is_discoverable
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = s.swiper_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.swiper_id = $1 AND mine.target_id = s.swiper_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = s.swiper_id) OR (b.blocker_id = s.swiper_id AND b.blocked_id = $1)
		  )
		ORDER BY s.created_at DESC
		LIMIT $2
	`, userID, limit, includeDemo)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, string, error) {
	var partnerID, conversationID string
	err := r.db.QueryRow(ctx, `
		UPDATE matches m
		SET unmatched_at = NOW(), unmatched_by = $2
		FROM conversations c
		WHERE m.id = $1 AND c.match_id = m.id
		  AND ($2 = m.user_a_id OR $2 = m.user_b_id)
		  AND m.unmatched_at IS NULL
		RETURNING CASE WHEN m.user_a_id = $2 THEN m.user_b_id ELSE m.user_a_id END, c.id
	`, matchID, userID).Scan(&partnerID, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrMatchNotFound
	}
	return partnerID, conversationID, err
}
