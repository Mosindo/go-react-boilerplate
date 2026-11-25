package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoTargetUnavailable = errors.New("target unavailable")
	errRepoConflict          = errors.New("swipe conflict")
	errRepoNotFound          = errors.New("not found")
)

type Repository interface {
	Discover(ctx context.Context, userID string, limit int, includeDemo bool) ([]profiles.CardRow, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (swipeResult, error)
	EndMatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// rankingOrder is the single place that defines how candidates are ordered.
// It currently favours shared interests, recent activity, then proximity.
// A smarter recommender can replace it (or the whole Discover query) without
// touching handlers or clients.
const rankingOrder = `c.common_interests DESC, c.recently_active DESC, c.distance ASC, c.user_id`

// Discover returns candidate profiles for userID. Mutual filters are applied:
// each side's gender preference, age range and maximum distance must accept
// the other. Profiles already swiped by the user, profiles that passed on the
// user, blocked users (both directions) and incomplete profiles are excluded.
func (r *PGRepository) Discover(ctx context.Context, userID string, limit int, includeDemo bool) ([]profiles.CardRow, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.latitude, p.longitude,
			       date_part('year', age(p.birthdate))::int AS age,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1 AND p.latitude IS NOT NULL
		),
		candidates AS (
			SELECT `+profiles.CardColumns+`,
			       (SELECT COUNT(*) FROM user_interests a
			         JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = me.user_id
			         WHERE a.user_id = p.user_id) AS common_interests,
			       (u.last_active_at > NOW() - INTERVAL '7 days') AS recently_active,
			       pr.max_distance_km AS their_max_distance
			FROM me
			JOIN profiles p ON p.user_id <> me.user_id
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.discoverable
			  AND p.latitude IS NOT NULL
			  AND ($3 OR NOT u.is_demo)
			  AND p.gender = ANY(me.interested_in)
			  AND me.gender = ANY(pr.interested_in)
			  AND p.birthdate >  CURRENT_DATE - make_interval(years => me.max_age + 1)
			  AND p.birthdate <= CURRENT_DATE - make_interval(years => me.min_age)
			  AND me.age BETWEEN pr.min_age AND pr.max_age
			  AND p.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = p.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = p.user_id AND s.target_id = me.user_id AND s.action = 'pass')
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
			       OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id)
			  )
		)
		SELECT c.user_id, c.first_name, c.birthdate, c.gender, c.bio, c.job_title, c.city, c.height_cm,
		       c.looking_for, c.show_distance, c.distance, c.is_demo_profile
		FROM candidates c, me
		WHERE c.distance <= me.max_distance_km AND c.distance <= c.their_max_distance
		ORDER BY `+rankingOrder+`
		LIMIT $2
	`, userID, limit, includeDemo)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, profiles.ScanCard)
}

// Swipe records a like or a pass. Concurrent swipes of the same pair are
// serialised with an advisory lock so a mutual like always yields exactly one
// match, even when both users like each other at the same instant.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (swipeResult, error) {
	res := swipeResult{action: action}
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtext('pair:' || LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))
		`, swiperID, targetID); err != nil {
			return err
		}

		err := tx.QueryRow(ctx, `
			SELECT me.first_name, t.first_name
			FROM profiles t
			JOIN profiles me ON me.user_id = $1
			WHERE t.user_id = $2
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
		`, swiperID, targetID).Scan(&res.swiperName, &res.targetName)
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoTargetUnavailable
		}
		if err != nil {
			return err
		}

		var existing string
		err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, swiperID, targetID).Scan(&existing)
		switch {
		case err == nil:
			if existing != action {
				return errRepoConflict
			}
			res.alreadySwiped = true
		case errors.Is(err, pgx.ErrNoRows):
			if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)`, swiperID, targetID, action); err != nil {
				return err
			}
		default:
			return err
		}

		if action != ActionLike {
			return nil
		}

		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
		`, swiperID, targetID).Scan(&reciprocal); err != nil {
			return err
		}
		if !reciprocal {
			return nil
		}

		var endedAt *string
		err = tx.QueryRow(ctx, `
			SELECT m.id, c.id, m.ended_at::text
			FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
		`, swiperID, targetID).Scan(&res.matchID, &res.conversationID, &endedAt)
		if err == nil {
			if endedAt != nil { // unmatched pairs never re-match
				res.matchID, res.conversationID = "", ""
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err := tx.QueryRow(ctx, `
			INSERT INTO matches (user_a_id, user_b_id)
			VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			RETURNING id
		`, swiperID, targetID).Scan(&res.matchID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, res.matchID).Scan(&res.conversationID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		`, res.conversationID, swiperID, targetID)
		return err
	})
	return res, err
}

func (r *PGRepository) EndMatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET ended_at = NOW(), ended_by = $1
		WHERE id = $2 AND ended_at IS NULL AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errRepoNotFound
	}
	return other, err
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
