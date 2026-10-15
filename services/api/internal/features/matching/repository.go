package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotEligible   = errors.New("matching: target not eligible")
	ErrRepositoryAlreadySwiped = errors.New("matching: already decided")
	ErrRepositoryNotMatched    = errors.New("matching: no match")
)

type Repository interface {
	Candidates(ctx context.Context, viewerID string, exclude []string, limit int) ([]users.CardBase, error)
	Candidate(ctx context.Context, viewerID, targetID string) (users.CardBase, error)
	Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, error)
	Unmatch(ctx context.Context, viewerID, targetID string) (conversationID string, err error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// The candidate rules live in one place so discovery, profile viewing and
// swiping can never disagree about who is allowed to see whom. $1 is always
// the viewer. To change how people are recommended later, only the ORDER BY of
// Candidates needs to evolve (or be swapped for a scoring query).
const viewerCTE = `
WITH v AS (
	SELECT pr.user_id, pr.gender, pr.birth_date, pr.latitude, pr.longitude,
	       pf.interested_in, pf.min_age, pf.max_age, pf.max_distance_km
	FROM profiles pr
	JOIN preferences pf ON pf.user_id = pr.user_id
	WHERE pr.user_id = $1
)`

// distance_km is a haversine distance on the stored (already coarse) coordinates.
const candidateFrom = `
FROM v
JOIN profiles p ON p.user_id <> v.user_id
JOIN preferences tp ON tp.user_id = p.user_id
CROSS JOIN LATERAL (
	SELECT CASE WHEN p.latitude IS NOT NULL AND v.latitude IS NOT NULL THEN
		12742 * asin(least(1, sqrt(
			power(sin(radians(p.latitude - v.latitude) / 2), 2) +
			cos(radians(v.latitude)) * cos(radians(p.latitude)) *
			power(sin(radians(p.longitude - v.longitude) / 2), 2)
		)))
	END AS km
) d`

const notBlocked = `
NOT EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_id = v.user_id AND b.blocked_id = p.user_id)
	   OR (b.blocker_id = p.user_id AND b.blocked_id = v.user_id)
)`

// mutualInterest: both people must fit each other's gender, age and distance preferences.
const mutualInterest = `
p.is_visible
AND p.gender = ANY(v.interested_in)
AND v.gender = ANY(tp.interested_in)
AND date_part('year', age(p.birth_date)) BETWEEN v.min_age AND v.max_age
AND date_part('year', age(v.birth_date)) BETWEEN tp.min_age AND tp.max_age
AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
AND (v.latitude IS NULL OR p.latitude IS NULL
     OR p.latitude BETWEEN v.latitude - v.max_distance_km / 111.0 AND v.latitude + v.max_distance_km / 111.0)
AND (d.km IS NULL OR (d.km <= v.max_distance_km AND d.km <= tp.max_distance_km))`

const eligibleWhere = `WHERE ` + notBlocked + ` AND ` + mutualInterest

const cardColumns = `
p.user_id, p.first_name, p.birth_date, p.gender, p.bio, p.city,
CASE WHEN p.show_distance THEN d.km END AS distance_km,
EXISTS (SELECT 1 FROM likes l WHERE l.from_user_id = p.user_id AND l.to_user_id = v.user_id AND l.kind = 'like') AS liked_me`

func scanCard(row pgx.Row) (users.CardBase, error) {
	var c users.CardBase
	err := row.Scan(&c.UserID, &c.FirstName, &c.BirthDate, &c.Gender, &c.Bio, &c.City, &c.DistanceKm, &c.LikedMe)
	return c, err
}

// Candidates returns the next profiles to show: eligible, never decided on,
// not already queued on the client. People who already liked the viewer come
// first, then shared interests, proximity and recent activity.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, exclude []string, limit int) ([]users.CardBase, error) {
	if exclude == nil {
		exclude = []string{}
	}
	rows, err := r.dbPool.Query(ctx, viewerCTE+`
	SELECT `+cardColumns+`
	`+candidateFrom+`
	`+eligibleWhere+`
	AND NOT EXISTS (SELECT 1 FROM likes l2 WHERE l2.from_user_id = v.user_id AND l2.to_user_id = p.user_id)
	AND p.user_id <> ALL($2::uuid[])
	ORDER BY
		liked_me DESC,
		(SELECT COUNT(*) FROM user_interests a
		 JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = v.user_id
		 WHERE a.user_id = p.user_id) DESC,
		d.km ASC NULLS LAST,
		p.last_active_at DESC,
		p.user_id
	LIMIT $3
	`, viewerID, exclude, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cards := make([]users.CardBase, 0, limit)
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// Candidate returns one profile if the viewer may see it: matched with it, or
// it fits both people's preferences. Blocked people are never visible.
func (r *PGRepository) Candidate(ctx context.Context, viewerID, targetID string) (users.CardBase, error) {
	c, err := scanCard(r.dbPool.QueryRow(ctx, viewerCTE+`
	SELECT `+cardColumns+`
	`+candidateFrom+`
	WHERE p.user_id = $2 AND `+notBlocked+`
	AND (
		EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST(v.user_id, p.user_id) AND m.user_b = GREATEST(v.user_id, p.user_id))
		OR (`+mutualInterest+`)
	)`, viewerID, targetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return users.CardBase{}, ErrRepositoryNotEligible
	}
	return c, err
}

// Swipe records one decision and, when it completes a mutual like, creates the
// match and its conversation in the same transaction. The pair lock makes two
// people liking each other at the same instant produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
	`, viewerID, targetID); err != nil {
		return SwipeResult{}, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, viewerCTE+`
	SELECT EXISTS (SELECT 1 `+candidateFrom+` `+eligibleWhere+` AND p.user_id = $2)`, viewerID, targetID).Scan(&eligible); err != nil {
		return SwipeResult{}, err
	}
	if !eligible {
		return SwipeResult{}, ErrRepositoryNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO likes (from_user_id, to_user_id, kind) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING
	`, viewerID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return SwipeResult{}, ErrRepositoryAlreadySwiped
	}

	result := SwipeResult{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM likes WHERE from_user_id = $2 AND to_user_id = $1 AND kind = 'like')
		`, viewerID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			if err := createMatch(ctx, tx, viewerID, targetID, &result); err != nil {
				return SwipeResult{}, err
			}
		}
	}
	return result, tx.Commit(ctx)
}

func createMatch(ctx context.Context, tx pgx.Tx, viewerID, targetID string, out *SwipeResult) error {
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
		RETURNING id
	`, viewerID, targetID).Scan(&out.MatchID); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, match_id)
		VALUES ('direct', LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), $3)
		ON CONFLICT (direct_key) DO UPDATE SET match_id = EXCLUDED.match_id, updated_at = NOW()
		RETURNING id
	`, viewerID, targetID, out.MatchID).Scan(&out.ConversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, out.ConversationID, viewerID, targetID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT user_id, first_name FROM profiles WHERE user_id IN ($1, $2)`, viewerID, targetID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		if id == viewerID {
			out.ViewerName = name
		} else {
			out.TargetName = name
		}
	}
	out.Matched = true
	return rows.Err()
}

// Unmatch deletes the match; the conversation and its messages cascade.
func (r *PGRepository) Unmatch(ctx context.Context, viewerID, targetID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var conversationID string
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(c.id::text, '') FROM matches m
		LEFT JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
	`, viewerID, targetID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotMatched
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
	`, viewerID, targetID); err != nil {
		return "", err
	}
	return conversationID, tx.Commit(ctx)
}
