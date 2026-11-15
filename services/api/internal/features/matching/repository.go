package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ViewerStatus(ctx context.Context, userID string) (complete bool, maxDistanceKm int, err error)
	Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (*Match, error)
	ReceivedLikes(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]ReceivedLike, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const blockedEitherWay = `EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
)`

func (r *PGRepository) ViewerStatus(ctx context.Context, userID string) (bool, int, error) {
	var complete bool
	var dist int16
	err := r.db.QueryRow(ctx, `
		SELECT `+profiles.CompleteProfileSQL+`, pr.max_distance_km
		FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&complete, &dist)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, nil
	}
	return complete, int(dist), err
}

// Candidates returns eligible profiles with ranking features. Eligibility is
// mutual: each side must fit the other's gender, age and distance
// preferences. Already-swiped, blocked, paused and incomplete profiles are
// excluded.
func (r *PGRepository) Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT v.user_id, v.gender, v.birthdate, v.approx_latitude AS lat, v.approx_longitude AS lng,
			       vp.interested_in, vp.min_age, vp.max_age, vp.max_distance_km
			FROM profiles v JOIN preferences vp ON vp.user_id = v.user_id
			WHERE v.user_id = $1
		),
		pool AS (
			SELECT p.user_id, u.last_active_at, p.bio,
			       CASE WHEN me.lat IS NOT NULL AND p.approx_latitude IS NOT NULL
			            THEN distance_km(me.lat, me.lng, p.approx_latitude, p.approx_longitude)
			       END AS dist,
			       cp.max_distance_km AS their_max_distance
			FROM me
			JOIN profiles p ON p.user_id <> me.user_id AND p.is_paused = FALSE
			JOIN preferences cp ON cp.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE `+profiles.CompleteProfileSQL+`
			  AND p.gender = ANY(me.interested_in)
			  AND me.gender = ANY(cp.interested_in)
			  AND p.birthdate > (current_date - make_interval(years => me.max_age + 1))
			  AND p.birthdate <= (current_date - make_interval(years => me.min_age))
			  AND date_part('year', age(me.birthdate)) BETWEEN cp.min_age AND cp.max_age
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
			  AND NOT `+blockedEitherWay+`
			  AND (
			    me.lat IS NULL
			    OR (
			      p.approx_latitude IS NOT NULL
			      AND p.approx_latitude BETWEEN me.lat - LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
			                                AND me.lat + LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
			      AND distance_km(me.lat, me.lng, p.approx_latitude, p.approx_longitude) <= LEAST(me.max_distance_km, cp.max_distance_km)
			    )
			  )
		)
		SELECT pool.user_id, pool.dist,
		       (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
		         WHERE a.user_id = $1 AND b.user_id = pool.user_id)::int AS shared,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = pool.user_id AND s.target_id = $1 AND s.action = 'like') AS liked_me,
		       pool.last_active_at,
		       (SELECT COUNT(*) FROM photos ph WHERE ph.user_id = pool.user_id)::int AS photo_count,
		       pool.bio <> '' AS has_bio
		FROM pool
		ORDER BY liked_me DESC, pool.last_active_at DESC, pool.user_id
		LIMIT $2
	`, userID, poolSize)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.DistanceKm, &c.SharedInterests, &c.LikedViewer, &c.LastActiveAt, &c.PhotoCount, &c.HasBio)
		return c, err
	})
}

// Swipe records a like/pass and creates the match when the like is mutual.
// A per-pair advisory lock serialises the two users' swipes so simultaneous
// mutual likes always produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (*Match, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := orderPair(swiperID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, low+":"+high); err != nil {
		return nil, err
	}

	var eligible bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2
			  AND `+profiles.CompleteProfileSQL+`
			  AND NOT `+blockedEitherWay+`
			  AND (p.is_paused = FALSE OR EXISTS (
			        SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like'))
		)
	`, swiperID, targetID).Scan(&eligible)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadySwiped
	}

	if action != ActionLike {
		return nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
	`, swiperID, targetID).Scan(&reciprocal); err != nil {
		return nil, err
	}
	if !reciprocal {
		return nil, tx.Commit(ctx)
	}

	var m Match
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id) VALUES ($1, $2)
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING
		RETURNING id, created_at
	`, low, high).Scan(&m.ID, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Defensive: the pair already has a match.
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, m.ID).Scan(&m.ConversationID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, m.ConversationID, swiperID, targetID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PGRepository) ReceivedLikes(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]ReceivedLike, error) {
	var cursorAt *time.Time
	var cursorID *string
	if cursor != nil {
		cursorAt, cursorID = &cursor.At, &cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT s.swiper_id, s.created_at
		FROM swipes s
		JOIN profiles p ON p.user_id = s.swiper_id
		WHERE s.target_id = $1 AND s.action = 'like'
		  AND p.is_paused = FALSE
		  AND `+profiles.CompleteProfileSQL+`
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.swiper_id = $1 AND mine.target_id = s.swiper_id)
		  AND NOT `+blockedEitherWay+`
		  AND ($2::timestamptz IS NULL OR (s.created_at, s.swiper_id) < ($2, $3::uuid))
		ORDER BY s.created_at DESC, s.swiper_id DESC
		LIMIT $4
	`, userID, cursorAt, cursorID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ReceivedLike, error) {
		var l ReceivedLike
		err := row.Scan(&l.UserID, &l.LikedAt)
		return l, err
	})
}

// Unmatch deletes the match; the conversation and its messages cascade. The
// swipes stay, so the two profiles never meet again in discovery.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
		RETURNING user_a_id, user_b_id
	`, matchID, userID).Scan(&a, &b)
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

func orderPair(x, y string) (string, string) {
	if x < y {
		return x, y
	}
	return y, x
}
