package matching

import (
	"context"
	"errors"
	"math"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound     = errors.New("not found")
	ErrRepositoryAlreadySwipe = errors.New("already swiped")
	ErrRepositoryUnavailable  = errors.New("target unavailable")
)

type Repository interface {
	ViewerContext(ctx context.Context, userID string) (ViewerContext, error)
	Candidates(ctx context.Context, viewer ViewerContext, today time.Time, limit int) ([]Candidate, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error)
	DeleteMatch(ctx context.Context, userID, matchID string) (otherUserID, conversationID string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) ViewerContext(ctx context.Context, userID string) (ViewerContext, error) {
	v := ViewerContext{UserID: userID}
	var ageMin, ageMax, distance int16
	err := r.dbPool.QueryRow(ctx, `
		SELECT p.gender, p.birthdate, p.location_lat, p.location_lng,
		       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&v.Gender, &v.Birthdate, &v.Latitude, &v.Longitude, &v.InterestedIn, &ageMin, &ageMax, &distance, &v.HasPhoto)
	if errors.Is(err, pgx.ErrNoRows) {
		return ViewerContext{}, ErrRepositoryNotFound
	}
	v.AgeMin, v.AgeMax, v.MaxDistanceKm = int(ageMin), int(ageMax), int(distance)
	return v, err
}

// birthdateBounds converts an age range into (exclusive lower, inclusive
// upper) birthdate bounds so the SQL filter can use the profiles index.
func birthdateBounds(today time.Time, ageMin, ageMax int) (time.Time, time.Time) {
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return day.AddDate(-(ageMax + 1), 0, 0), day.AddDate(-ageMin, 0, 0)
}

const haversineSQL = `6371 * 2 * asin(least(1, sqrt(
	power(sin(radians(p.location_lat - $9::float8) / 2), 2) +
	cos(radians($9::float8)) * cos(radians(p.location_lat)) * power(sin(radians(p.location_lng - $10::float8) / 2), 2)
)))`

// Candidates returns a pool of eligible profiles. Every rule is enforced in
// SQL: mutual gender and age preferences, mutual distance, not already
// swiped, no block in either direction, active account with a photo.
func (r *PGRepository) Candidates(ctx context.Context, viewer ViewerContext, today time.Time, limit int) ([]Candidate, error) {
	lower, upper := birthdateBounds(today, viewer.AgeMin, viewer.AgeMax)
	viewerAge := profiles.AgeOn(viewer.Birthdate, today)

	var lat, lng, latMin, latMax, lngMin, lngMax *float64
	if viewer.Latitude != nil && viewer.Longitude != nil {
		lat, lng = viewer.Latitude, viewer.Longitude
		dLat := float64(viewer.MaxDistanceKm) / 111.0
		cosLat := math.Max(0.01, math.Cos(*lat*math.Pi/180))
		dLng := float64(viewer.MaxDistanceKm) / (111.32 * cosLat)
		a, b := *lat-dLat, *lat+dLat
		latMin, latMax = &a, &b
		// Skip the longitude pre-filter near the antimeridian; haversine still applies.
		if *lng-dLng > -180 && *lng+dLng < 180 {
			c, d := *lng-dLng, *lng+dLng
			lngMin, lngMax = &c, &d
		}
	}

	rows, err := r.dbPool.Query(ctx, `
		SELECT p.user_id, d.distance_km,
		       (SELECT COUNT(*) FROM user_interests ui
		         WHERE ui.user_id = p.user_id
		           AND ui.interest_id IN (SELECT interest_id FROM user_interests WHERE user_id = $1)) AS shared,
		       EXTRACT(EPOCH FROM (NOW() - u.last_active_at)) / 3600 AS idle_hours,
		       EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = p.user_id AND l.target_id = $1 AND l.action = 'like') AS liked_viewer
		FROM profiles p
		JOIN users u ON u.id = p.user_id AND u.moderation_status = 'active'
		JOIN preferences pr ON pr.user_id = p.user_id
		CROSS JOIN LATERAL (
			SELECT CASE WHEN $9::float8 IS NULL OR p.location_lat IS NULL THEN NULL ELSE `+haversineSQL+` END AS distance_km
		) d
		WHERE p.user_id <> $1
		  AND p.discoverable
		  AND p.gender = ANY($2::text[])
		  AND $3::text = ANY(pr.interested_in)
		  AND p.birthdate > $4::date AND p.birthdate <= $5::date
		  AND $6::int BETWEEN pr.age_min AND pr.age_max
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
		  )
		  AND (
			$9::float8 IS NULL
			OR (
				p.location_lat BETWEEN $11::float8 AND $12::float8
				AND ($13::float8 IS NULL OR p.location_lng BETWEEN $13::float8 AND $14::float8)
				AND d.distance_km <= LEAST($8::int, pr.max_distance_km)
			)
		  )
		ORDER BY u.last_active_at DESC, p.user_id
		LIMIT $7
	`, viewer.UserID, viewer.InterestedIn, viewer.Gender, lower, upper, viewerAge, limit, viewer.MaxDistanceKm,
		lat, lng, latMin, latMax, lngMin, lngMax)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.DistanceKm, &c.SharedInterests, &c.IdleHours, &c.LikedViewer)
		return c, err
	})
}

// mutualCompatibilitySQL expects aliases p/pr (other member) and me/mypr
// (current member): each one's gender and age fit the other's preferences.
const mutualCompatibilitySQL = `(
	p.gender = ANY(mypr.interested_in) AND me.gender = ANY(pr.interested_in)
	AND date_part('year', age(CURRENT_DATE, p.birthdate)) BETWEEN mypr.age_min AND mypr.age_max
	AND date_part('year', age(CURRENT_DATE, me.birthdate)) BETWEEN pr.age_min AND pr.age_max
)`

// Swipe records a decision and creates the match atomically. A per-pair
// advisory lock makes two simultaneous mutual likes produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::uuid, $2::uuid)::text || GREATEST($1::uuid, $2::uuid)::text, 0))
	`, swiperID, targetID); err != nil {
		return SwipeResult{}, err
	}

	// The target must be someone the swiper could have discovered: active,
	// visible, with a photo, not blocked, and mutually compatible in gender
	// and age preferences. Distance is not re-checked (people move).
	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.moderation_status = 'active'
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN profiles me ON me.user_id = $1
			JOIN preferences mypr ON mypr.user_id = $1
			WHERE p.user_id = $2 AND p.user_id <> $1 AND p.discoverable
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
			  AND `+mutualCompatibilitySQL+`
		)
	`, swiperID, targetID).Scan(&eligible); err != nil {
		return SwipeResult{}, err
	}
	if !eligible {
		return SwipeResult{}, ErrRepositoryUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrRepositoryAlreadySwipe
	}

	result := SwipeResult{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
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
				if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, result.MatchID).Scan(&result.ConversationID); err != nil {
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

// LikesReceived lists members who liked the user and are still awaiting a
// decision. Seeing who liked you is free in this product.
func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT s.swiper_id
		FROM swipes s
		JOIN profiles p ON p.user_id = s.swiper_id AND p.discoverable
		JOIN preferences pr ON pr.user_id = s.swiper_id
		JOIN users u ON u.id = s.swiper_id AND u.moderation_status = 'active'
		JOIN profiles me ON me.user_id = $1
		JOIN preferences mypr ON mypr.user_id = $1
		WHERE s.target_id = $1 AND s.action = 'like'
		  AND `+mutualCompatibilitySQL+`
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.swiper_id = $1 AND mine.target_id = s.swiper_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = s.swiper_id) OR (b.blocker_id = s.swiper_id AND b.blocked_id = $1)
		  )
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = s.swiper_id)
		ORDER BY s.created_at DESC, s.swiper_id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// DeleteMatch unmatches two members. The conversation and its messages are
// removed by cascade; both swipes remain so the profiles never reappear.
func (r *PGRepository) DeleteMatch(ctx context.Context, userID, matchID string) (string, string, error) {
	var other string
	var conversationID *string
	err := r.dbPool.QueryRow(ctx, `
		WITH target AS (
			SELECT m.id,
			       CASE WHEN m.user_a_id = $2 THEN m.user_b_id ELSE m.user_a_id END AS other_id,
			       (SELECT c.id FROM conversations c WHERE c.match_id = m.id) AS conversation_id
			FROM matches m
			WHERE m.id = $1 AND (m.user_a_id = $2 OR m.user_b_id = $2)
		), deleted AS (
			DELETE FROM matches WHERE id IN (SELECT id FROM target) RETURNING id
		)
		SELECT other_id, conversation_id FROM target WHERE EXISTS (SELECT 1 FROM deleted)
	`, matchID, userID).Scan(&other, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", "", err
	}
	conv := ""
	if conversationID != nil {
		conv = *conversationID
	}
	return other, conv, nil
}
