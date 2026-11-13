package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadySwiped     = errors.New("you already answered this profile")
	ErrTargetUnavailable = errors.New("profile not available")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	ProfileStatus(ctx context.Context, userID string) (ProfileStatus, error)
	CandidatePool(ctx context.Context, userID, organizationID string, limit int) ([]Candidate, error)
	CanSwipe(ctx context.Context, swiperID, targetID string) (bool, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error)
	Unmatch(ctx context.Context, userID, otherUserID string) error
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// ProfileStatus returns the flags that decide whether a member may browse
// and swipe: members must be visible themselves to see others.
func (r *PGRepository) ProfileStatus(ctx context.Context, userID string) (ProfileStatus, error) {
	var st ProfileStatus
	err := r.db.QueryRow(ctx, `
		SELECT is_complete, is_paused, moderation_hidden FROM profiles WHERE user_id = $1
	`, userID).Scan(&st.Complete, &st.Paused, &st.Hidden)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProfileStatus{}, nil
	}
	return st, err
}

// CandidatePool applies every hard filter in SQL: mutual gender preference,
// mutual age range, mutual distance, visibility, no previous swipe by the
// viewer, no pass from the candidate, and no block in either direction.
func (r *PGRepository) CandidatePool(ctx context.Context, userID, organizationID string, limit int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
			       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km,
			       date_part('year', age(p.birthdate)) AS my_age
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			WHERE p.user_id = $1 AND p.is_complete
		),
		cand AS (
			SELECT c.user_id, c.last_active_at,
			       LEAST(me.max_distance_km, cp.max_distance_km) AS max_km,
			       6371 * 2 * asin(sqrt(
			         power(sin(radians(c.latitude - me.latitude) / 2), 2) +
			         cos(radians(me.latitude)) * cos(radians(c.latitude)) *
			         power(sin(radians(c.longitude - me.longitude) / 2), 2)
			       )) AS distance_km
			FROM me
			JOIN profiles c
			  ON c.is_complete AND NOT c.is_paused AND NOT c.moderation_hidden
			 AND c.user_id <> me.user_id
			 AND c.latitude BETWEEN me.latitude - (me.max_distance_km / 111.0)
			                    AND me.latitude + (me.max_distance_km / 111.0)
			JOIN preferences cp ON cp.user_id = c.user_id
			JOIN users u ON u.id = c.user_id AND u.organization_id = $2
			WHERE c.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cp.interested_in)
			  AND date_part('year', age(c.birthdate)) BETWEEN me.age_min AND me.age_max
			  AND me.my_age BETWEEN cp.age_min AND cp.age_max
			  AND NOT EXISTS (
				SELECT 1 FROM swipes s
				WHERE s.swiper_user_id = me.user_id AND s.target_user_id = c.user_id
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM swipes s
				WHERE s.swiper_user_id = c.user_id AND s.target_user_id = me.user_id AND s.action = 'pass'
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = me.user_id AND b.blocked_user_id = c.user_id)
				   OR (b.blocker_user_id = c.user_id AND b.blocked_user_id = me.user_id)
			  )
		)
		SELECT cand.user_id, cand.distance_km, cand.max_km, cand.last_active_at,
		       (SELECT COUNT(*) FROM user_interests a
		        JOIN user_interests b ON b.interest_id = a.interest_id
		        WHERE a.user_id = $1 AND b.user_id = cand.user_id) AS shared
		FROM cand
		WHERE cand.distance_km <= cand.max_km
		ORDER BY cand.last_active_at DESC, cand.user_id
		LIMIT $3
	`, userID, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Candidate, 0, limit)
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.MaxDistanceKm, &c.LastActiveAt, &c.SharedInterests); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CanSwipe: target is a visible, complete profile with no block either way.
func (r *PGRepository) CanSwipe(ctx context.Context, swiperID, targetID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND p.is_complete AND NOT p.is_paused AND NOT p.moderation_hidden
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $2)
				   OR (b.blocker_user_id = $2 AND b.blocked_user_id = $1)
			  )
			  AND (SELECT organization_id FROM users WHERE id = $1) =
			      (SELECT organization_id FROM users WHERE id = $2)
		)
	`, swiperID, targetID).Scan(&ok)
	return ok, err
}

// Swipe records a decision and creates the match on a mutual like. A
// transaction-scoped advisory lock on the pair serializes concurrent likes
// from both sides, so the second like always sees the first one and exactly
// one match row is created.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
	`, swiperID, targetID); err != nil {
		return swipeOutcome{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `
		SELECT action FROM swipes WHERE swiper_user_id = $1 AND target_user_id = $2
	`, swiperID, targetID).Scan(&existing)
	switch {
	case err == nil:
		if existing != action {
			return swipeOutcome{}, ErrAlreadySwiped
		}
		// Idempotent replay: report the current state without side effects.
		out, err := activeMatch(ctx, tx, swiperID, targetID)
		if err != nil {
			return swipeOutcome{}, err
		}
		return out, tx.Commit(ctx)
	case !errors.Is(err, pgx.ErrNoRows):
		return swipeOutcome{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_user_id, target_user_id, action) VALUES ($1, $2, $3)
	`, swiperID, targetID, action); err != nil {
		return swipeOutcome{}, err
	}

	if action != ActionLike {
		return swipeOutcome{}, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM swipes
			WHERE swiper_user_id = $2 AND target_user_id = $1 AND action = 'like'
		)
	`, swiperID, targetID).Scan(&reciprocal); err != nil {
		return swipeOutcome{}, err
	}
	if !reciprocal {
		return swipeOutcome{}, tx.Commit(ctx)
	}

	out := swipeOutcome{Matched: true, New: true}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING
		RETURNING id, created_at
	`, swiperID, targetID).Scan(&out.MatchID, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		existingMatch, err := activeMatch(ctx, tx, swiperID, targetID)
		if err != nil {
			return swipeOutcome{}, err
		}
		return existingMatch, tx.Commit(ctx)
	}
	if err != nil {
		return swipeOutcome{}, err
	}
	return out, tx.Commit(ctx)
}

func activeMatch(ctx context.Context, tx pgx.Tx, a, b string) (swipeOutcome, error) {
	var out swipeOutcome
	err := tx.QueryRow(ctx, `
		SELECT id, created_at FROM matches
		WHERE user_a_id = LEAST($1::uuid, $2::uuid)
		  AND user_b_id = GREATEST($1::uuid, $2::uuid)
		  AND unmatched_at IS NULL
	`, a, b).Scan(&out.MatchID, &out.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return swipeOutcome{}, nil
	}
	if err != nil {
		return swipeOutcome{}, err
	}
	out.Matched = true
	return out, nil
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, otherUserID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE matches
		SET unmatched_at = NOW(), unmatched_by_user_id = $1
		WHERE user_a_id = LEAST($1::uuid, $2::uuid)
		  AND user_b_id = GREATEST($1::uuid, $2::uuid)
		  AND unmatched_at IS NULL
	`, userID, otherUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
