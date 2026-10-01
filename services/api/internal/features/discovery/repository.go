package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// candidateSQL selects the user ids a viewer ($1) may be shown. It is the single source of truth
// for eligibility: used both to build the discovery feed and to authorise a swipe, so the server
// never trusts the client about who it was allowed to see.
//
// Rules: visible, complete profiles; mutual gender interest; mutual age ranges; mutual distance
// limits (only when the viewer shared a location); nobody already swiped; nobody blocked in
// either direction.
//
// Ranking lives in the ORDER BY below and is deliberately isolated: replace it (or call out to a
// recommender that returns ordered ids) without touching eligibility.
func candidateSQL(extraWhere, tail string) string {
	return `
	WITH me AS (
	  SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude,
	         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	  WHERE p.user_id = $1
	)
	SELECT c.user_id
	FROM me
	JOIN profiles c ON c.user_id <> me.user_id
	JOIN preferences cp ON cp.user_id = c.user_id
	JOIN users cu ON cu.id = c.user_id
	WHERE c.is_visible
	  AND c.gender = ANY(me.interested_in)
	  AND me.gender = ANY(cp.interested_in)
	  AND date_part('year', age(CURRENT_DATE, c.birth_date)) BETWEEN me.min_age AND me.max_age
	  AND date_part('year', age(CURRENT_DATE, me.birth_date)) BETWEEN cp.min_age AND cp.max_age
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)
	  AND NOT EXISTS (
	    SELECT 1 FROM blocks b
	    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
	  AND (
	    me.latitude IS NULL
	    OR (
	      c.latitude IS NOT NULL
	      AND c.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
	      AND ` + profiles.HaversineKM("c", "me") + ` <= LEAST(me.max_distance_km, cp.max_distance_km)
	    )
	  )
	  ` + extraWhere + `
	` + tail
}

const rankTail = `
	ORDER BY
	  EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = c.user_id AND s.to_user_id = me.user_id AND s.action = 'like') DESC,
	  (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
	    WHERE a.user_id = c.user_id AND b.user_id = me.user_id) DESC,
	  CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN ` + `(c.latitude - me.latitude) * (c.latitude - me.latitude) + (c.longitude - me.longitude) * (c.longitude - me.longitude)` + ` END ASC NULLS LAST,
	  cu.last_active_at DESC,
	  c.user_id
	LIMIT $2`

// Candidates returns up to limit eligible user ids, best first.
func (r *Repository) Candidates(ctx context.Context, viewerID string, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx, candidateSQL("", rankTail), viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) isEligible(ctx context.Context, q pgx.Tx, viewerID, targetID string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (`+candidateSQL("AND c.user_id = $2", "")+`)`, viewerID, targetID).Scan(&ok)
	return ok, err
}

// HasCompleteProfile reports whether the viewer can take part in discovery.
func (r *Repository) HasCompleteProfile(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, userID).Scan(&ok)
	return ok, err
}

func isUUIDError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// Swipe records a like/pass and, on mutual likes, creates the match and its conversation.
// A per-pair advisory lock serialises concurrent opposite swipes so exactly one match is created
// and neither side can miss the other's like.
func (r *Repository) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, error) {
	res := SwipeResult{Action: action}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, viewerID, targetID); err != nil {
		if isUUIDError(err) {
			return res, ErrNotFound
		}
		return res, err
	}

	var swiped bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2)`, viewerID, targetID).Scan(&swiped); err != nil {
		return res, err
	}
	if swiped {
		return res, ErrAlreadySwiped
	}
	ok, err := r.isEligible(ctx, tx, viewerID, targetID)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, viewerID, targetID, action); err != nil {
		return res, err
	}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, targetID, viewerID).Scan(&reciprocal); err != nil {
			return res, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				WITH m AS (
				  INSERT INTO matches (user_a_id, user_b_id)
				  VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				  ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = matches.user_a_id
				  RETURNING id
				), c AS (
				  INSERT INTO conversations (match_id) SELECT id FROM m
				  ON CONFLICT (match_id) DO UPDATE SET match_id = conversations.match_id
				  RETURNING id, match_id
				), p AS (
				  INSERT INTO conversation_participants (conversation_id, user_id)
				  SELECT c.id, u FROM c, unnest(ARRAY[$1::uuid, $2::uuid]) AS u
				  ON CONFLICT DO NOTHING
				)
				SELECT c.match_id, c.id FROM c
			`, viewerID, targetID).Scan(&res.MatchID, &res.ConversationID)
			if err != nil {
				return res, err
			}
			res.Matched = true
		}
	}
	return res, tx.Commit(ctx)
}

// Unswipe removes the viewer's like/pass so the profile can be shown again. Refused when matched.
func (r *Repository) Unswipe(ctx context.Context, viewerID, targetID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, viewerID, targetID); err != nil {
		if isUUIDError(err) {
			return ErrNotFound
		}
		return err
	}
	var matched bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid))`, viewerID, targetID).Scan(&matched); err != nil {
		return err
	}
	if matched {
		return ErrMatched
	}
	tag, err := tx.Exec(ctx, `DELETE FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, viewerID, targetID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// CanView applies the profile-visibility rule: not blocked, target visible, and a real relationship
// (matched, they liked the viewer, the viewer already swiped them, or they are an eligible candidate).
func (r *Repository) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $2 AND is_visible)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		  AND (
		    EXISTS (SELECT 1 FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid))
		    OR EXISTS (SELECT 1 FROM swipes WHERE (from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1 AND action = 'like'))
		    OR EXISTS (`+candidateSQL("AND c.user_id = $2", "")+`)
		  )
	`, viewerID, targetID).Scan(&ok)
	if err != nil {
		if isUUIDError(err) {
			return false, nil
		}
		return false, err
	}
	return ok, nil
}

func (r *Repository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]matchRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, c.id, CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END, m.created_at
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a_id = $1 OR m.user_b_id = $1
		ORDER BY m.created_at DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []matchRow
	for rows.Next() {
		var m matchRow
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Unmatch deletes the match (and, by cascade, its conversation and messages) for both users.
// It returns the other participant so they can be informed.
func (r *Repository) Unmatch(ctx context.Context, userID, matchID string) (otherID string, err error) {
	err = r.pool.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
	`, userID, matchID).Scan(&otherID)
	if errors.Is(err, pgx.ErrNoRows) || (err != nil && isUUIDError(err)) {
		return "", ErrNotFound
	}
	return otherID, err
}
