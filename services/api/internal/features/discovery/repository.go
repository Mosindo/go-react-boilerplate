package discovery

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoAlreadySwiped = errors.New("already swiped")
	ErrRepoNotEligible   = errors.New("target not eligible")
	ErrRepoNotFound      = errors.New("not found")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type Candidate struct {
	UserID       string
	FirstName    string
	Age          int
	Gender       string
	Bio          string
	City         string
	DistanceKm   *float64
	ShowDistance bool
}

// The eligibility rules live in SQL so the database does the filtering in one pass:
//   - $1 viewer, $2 limit, $3 optional single target, $4 include already-swiped, $5 bypass preferences
//   - mutual gender / age / distance preferences, target discoverable with at least one photo
//   - never the viewer, never blocked in either direction
const candidatesSQL = `
WITH me AS (
	SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude,
	       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1
), cand AS (
	SELECT c.user_id, c.first_name, c.gender, c.bio, c.city, u.last_active_at, cp.show_distance,
	       date_part('year', age(c.birth_date))::int AS age,
	       CASE WHEN me.latitude IS NULL OR c.latitude IS NULL THEN NULL
	            ELSE 2 * 6371 * asin(sqrt(least(1.0,
	                 power(sin(radians(c.latitude - me.latitude) / 2), 2)
	               + cos(radians(me.latitude)) * cos(radians(c.latitude))
	               * power(sin(radians(c.longitude - me.longitude) / 2), 2)))) END AS dist,
	       me.max_distance_km AS my_max_distance, cp.max_distance_km AS their_max_distance,
	       (SELECT count(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
	         WHERE a.user_id = me.user_id AND b.user_id = c.user_id)::int AS shared,
	       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') AS likes_me
	FROM me
	JOIN profiles c ON c.user_id <> me.user_id
	JOIN preferences cp ON cp.user_id = c.user_id
	JOIN users u ON u.id = c.user_id
	WHERE ($3::uuid IS NULL OR c.user_id = $3::uuid)
	  AND NOT EXISTS (
	        SELECT 1 FROM blocks b
	        WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	           OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND ($4 OR NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id))
	  AND ($5 OR (
	        cp.discoverable
	    AND c.gender = ANY (me.interested_in)
	    AND me.gender = ANY (cp.interested_in)
	    AND c.birth_date <= (current_date - make_interval(years => me.min_age))::date
	    AND c.birth_date >  (current_date - make_interval(years => me.max_age + 1))::date
	    AND me.birth_date <= (current_date - make_interval(years => cp.min_age))::date
	    AND me.birth_date >  (current_date - make_interval(years => cp.max_age + 1))::date
	    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	    AND (me.latitude IS NULL OR c.latitude IS NULL
	         OR c.latitude BETWEEN me.latitude - least(me.max_distance_km, cp.max_distance_km) / 111.0
	                           AND me.latitude + least(me.max_distance_km, cp.max_distance_km) / 111.0)))
)
SELECT user_id, first_name, age, gender, bio, city, dist, show_distance
FROM cand
WHERE $5 OR dist IS NULL OR (dist <= my_max_distance AND dist <= their_max_distance)
ORDER BY (` + rankingScoreSQL + `) DESC, user_id
LIMIT $2`

func (r *Repository) query(ctx context.Context, viewerID string, limit int, target *string, includeSwiped, bypass bool) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, candidatesSQL, viewerID, limit, target, includeSwiped, bypass)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &c.DistanceKm, &c.ShowDistance); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) NextCandidates(ctx context.Context, viewerID string, limit int) ([]Candidate, error) {
	return r.query(ctx, viewerID, limit, nil, false, false)
}

// EligibleTarget returns the target if the viewer is allowed to see/swipe them (swiped or not).
func (r *Repository) EligibleTarget(ctx context.Context, viewerID, targetID string) (*Candidate, error) {
	list, err := r.query(ctx, viewerID, 1, &targetID, true, false)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// MatchedTarget returns the target if viewer and target are matched and not blocked.
func (r *Repository) MatchedTarget(ctx context.Context, viewerID, targetID string) (*Candidate, error) {
	var matched bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid))
	`, viewerID, targetID).Scan(&matched); err != nil || !matched {
		return nil, err
	}
	list, err := r.query(ctx, viewerID, 1, &targetID, true, true)
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return &list[0], nil
}

// HasCompleteProfile reports whether the user has a profile and at least one photo.
func (r *Repository) HasCompleteProfile(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, userID).Scan(&ok)
	return ok, err
}

type MatchRow struct {
	ID             string
	OtherUserID    string
	ConversationID string
	CreatedAt      time.Time
}

// RecordSwipe stores the decision and, when it completes a mutual like, creates the match and
// its conversation in the same transaction. A per-pair advisory lock serialises two people liking
// each other at the same instant so exactly one of them observes the reciprocal row.
func (r *Repository) RecordSwipe(ctx context.Context, viewerID, targetID, action string) (*MatchRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))`, viewerID, targetID); err != nil {
		return nil, err
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))
	`, viewerID, targetID).Scan(&blocked); err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrRepoNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
	`, viewerID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRepoAlreadySwiped
	}

	var match *MatchRow
	if action == "like" {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
		`, viewerID, targetID).Scan(&reciprocal); err != nil {
			return nil, err
		}
		if reciprocal {
			m := MatchRow{OtherUserID: targetID}
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b)
				VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id, created_at
			`, viewerID, targetID).Scan(&m.ID, &m.CreatedAt)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				// Match already existed (e.g. created before an unmatch/re-like edge case): nothing new to announce.
			case err != nil:
				return nil, err
			default:
				if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, m.ID).Scan(&m.ConversationID); err != nil {
					return nil, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
				`, m.ConversationID, viewerID, targetID); err != nil {
					return nil, err
				}
				match = &m
			}
		}
	}
	return match, tx.Commit(ctx)
}

func (r *Repository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END, c.id, m.created_at
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id IN (m.user_a, m.user_b))
		           OR (b.blocked_id = $1 AND b.blocker_id IN (m.user_a, m.user_b)))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MatchRow
	for rows.Next() {
		var m MatchRow
		if err := rows.Scan(&m.ID, &m.OtherUserID, &m.ConversationID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Unmatch deletes the match (and, by cascade, its conversation) if the user is part of it.
// It returns the other participant so they can be told.
func (r *Repository) Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, conversationID string, err error) {
	err = r.pool.QueryRow(ctx, `
		WITH deleted AS (
			DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
			RETURNING id, user_a, user_b
		)
		SELECT CASE WHEN d.user_a = $1 THEN d.user_b ELSE d.user_a END,
		       COALESCE((SELECT id FROM conversations WHERE match_id = d.id), '00000000-0000-0000-0000-000000000000')
		FROM deleted d
	`, userID, matchID).Scan(&otherUserID, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrRepoNotFound
	}
	return otherUserID, conversationID, err
}
