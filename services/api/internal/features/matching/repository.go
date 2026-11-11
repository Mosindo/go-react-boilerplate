package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadySwiped     = errors.New("already swiped")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrMatchNotFound     = errors.New("match not found")
)

type Readiness struct {
	HasProfile     bool
	HasPreferences bool
	HasPhoto       bool
}

type Repository interface {
	Readiness(ctx context.Context, userID string) (Readiness, error)
	Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, error)
	RecordSwipe(ctx context.Context, swiperID, targetID, direction string) (*Match, error)
	DeleteMatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) Readiness(ctx context.Context, userID string) (Readiness, error) {
	var out Readiness
	err := r.db.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM profiles WHERE user_id = $1),
			EXISTS (SELECT 1 FROM preferences WHERE user_id = $1),
			EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, userID).Scan(&out.HasProfile, &out.HasPreferences, &out.HasPhoto)
	return out, err
}

// haversine SQL between the viewer (me) and a candidate (c), in kilometres.
const haversineSQL = `(6371 * 2 * asin(LEAST(1, sqrt(
	power(sin(radians(c.latitude - me.latitude) / 2), 2) +
	cos(radians(me.latitude)) * cos(radians(c.latitude)) *
	power(sin(radians(c.longitude - me.longitude) / 2), 2)
))))`

// Candidates returns eligible profiles for the viewer. Rules are mutual:
// each side must match the other's gender, age range and distance limit.
// Profiles already swiped, blocked in either direction, hidden, without a
// photo or the viewer itself are excluded.
func (r *PGRepository) Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
			FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1
		)
		SELECT c.user_id,
		       CASE WHEN me.latitude IS NULL OR c.latitude IS NULL THEN NULL ELSE `+haversineSQL+` END AS distance_km,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = c.user_id
		         WHERE a.user_id = me.user_id) AS shared,
		       c.last_active_at,
		       EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = c.user_id AND l.target_id = me.user_id AND l.direction = 'like') AS liked_me
		FROM me
		JOIN profiles c ON c.discoverable AND c.user_id <> me.user_id
		JOIN preferences cp ON cp.user_id = c.user_id
		WHERE c.gender = ANY (me.interested_in)
		  AND me.gender = ANY (cp.interested_in)
		  AND c.birthdate <= CURRENT_DATE - make_interval(years => me.age_min::int)
		  AND c.birthdate > CURRENT_DATE - make_interval(years => me.age_max::int + 1)
		  AND me.birthdate <= CURRENT_DATE - make_interval(years => cp.age_min::int)
		  AND me.birthdate > CURRENT_DATE - make_interval(years => cp.age_max::int + 1)
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
			   OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id)
		  )
		  AND (
			me.latitude IS NULL OR c.latitude IS NULL OR (
				c.latitude BETWEEN me.latitude - LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
				               AND me.latitude + LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
				AND `+haversineSQL+` <= LEAST(me.max_distance_km, cp.max_distance_km)
			)
		  )
		ORDER BY c.last_active_at DESC, c.user_id
		LIMIT $2
	`, userID, poolSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Candidate, 0, poolSize)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.SharedInterests, &c.LastActiveAt, &c.LikedViewer); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RecordSwipe stores a like/pass and creates the match when the like is
// reciprocal. A per-pair advisory lock serializes concurrent swipes of the two
// users so simultaneous mutual likes always produce exactly one match.
func (r *PGRepository) RecordSwipe(ctx context.Context, swiperID, targetID, direction string) (*Match, error) {
	var match *Match
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtext('pair:' || LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))
		`, swiperID, targetID); err != nil {
			return err
		}

		var available bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM profiles p
				WHERE p.user_id = $2 AND p.discoverable
				  AND NOT EXISTS (
					SELECT 1 FROM blocks b
					WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
				  )
			)
		`, swiperID, targetID).Scan(&available); err != nil {
			return err
		}
		if !available {
			return ErrTargetUnavailable
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, direction) VALUES ($1, $2, $3)
			ON CONFLICT (swiper_id, target_id) DO NOTHING
		`, swiperID, targetID, direction)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrAlreadySwiped
		}
		if direction != DirectionLike {
			return nil
		}

		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND direction = 'like')
		`, swiperID, targetID).Scan(&reciprocal); err != nil {
			return err
		}
		if !reciprocal {
			return nil
		}

		m := Match{UserA: swiperID, UserB: targetID}
		err = tx.QueryRow(ctx, `
			INSERT INTO matches (user_low_id, user_high_id)
			VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			ON CONFLICT (user_low_id, user_high_id) DO NOTHING
			RETURNING id, created_at
		`, swiperID, targetID).Scan(&m.ID, &m.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO match_members (match_id, user_id) VALUES ($1, $2), ($1, $3)
		`, m.ID, swiperID, targetID); err != nil {
			return err
		}
		match = &m
		return nil
	})
	return match, err
}

// DeleteMatch removes a match the user belongs to (unmatch). Messages cascade.
func (r *PGRepository) DeleteMatch(ctx context.Context, userID, matchID string) (string, error) {
	var low, high string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
		RETURNING user_low_id, user_high_id
	`, matchID, userID).Scan(&low, &high)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	if err != nil {
		return "", err
	}
	if low == userID {
		return high, nil
	}
	return low, nil
}
