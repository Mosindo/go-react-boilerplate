package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoIneligible = errors.New("target is not an eligible candidate")
	ErrRepoIncomplete = errors.New("caller profile incomplete")
)

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, userID, targetID, action string) (swipeOutcome, error)
	ResetPasses(ctx context.Context, userID string) error
	IsComplete(ctx context.Context, userID string) (bool, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// The eligibility predicate is shared by discovery and swipes so the server enforces, on every swipe,
// the same rules that shaped the queue: mutual gender/age/distance preferences, visibility and blocks.
// Aliases: v/vp = viewer profile/preferences, c/cp = candidate profile/preferences.
const eligibility = `
	c.user_id <> v.user_id
	AND c.discoverable
	AND c.gender = ANY (vp.interested_in)
	AND v.gender = ANY (cp.interested_in)
	AND date_part('year', age(CURRENT_DATE, c.birth_date)) BETWEEN vp.min_age AND vp.max_age
	AND date_part('year', age(CURRENT_DATE, v.birth_date)) BETWEEN cp.min_age AND cp.max_age
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	AND NOT EXISTS (SELECT 1 FROM blocks b
	                WHERE (b.blocker_id = v.user_id AND b.blocked_id = c.user_id)
	                   OR (b.blocker_id = c.user_id AND b.blocked_id = v.user_id))
	AND (v.latitude IS NULL OR c.latitude IS NULL
	     OR haversine_km(v.latitude, v.longitude, c.latitude, c.longitude) <= LEAST(vp.max_distance_km, cp.max_distance_km))`

const viewerComplete = `EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = v.user_id)`

// Candidates returns ids of unseen eligible profiles, best first. Ranking (see README "Matching"):
// people who already liked the viewer, shared interests, then proximity; ties are broken by a stable
// per-viewer-per-day shuffle so the queue does not always show the same faces first.
func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.user_id
		FROM profiles v
		JOIN preferences vp ON vp.user_id = v.user_id
		JOIN profiles c ON TRUE
		JOIN preferences cp ON cp.user_id = c.user_id
		WHERE v.user_id = $1
		  AND `+viewerComplete+`
		  AND `+eligibility+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = v.user_id AND s.target_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST(v.user_id, c.user_id) AND m.user_b = GREATEST(v.user_id, c.user_id))
		ORDER BY
		  (CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = v.user_id AND s.action = 'like')
		        THEN 10 ELSE 0 END)
		  + 3 * (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
		         WHERE a.user_id = c.user_id AND b.user_id = v.user_id)
		  - CASE WHEN v.latitude IS NULL OR c.latitude IS NULL THEN 5
		         ELSE least(10, haversine_km(v.latitude, v.longitude, c.latitude, c.longitude) / 10) END DESC,
		  md5(c.user_id::text || v.user_id::text || CURRENT_DATE::text)
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// Swipe records the decision atomically. A per-pair advisory lock makes two simultaneous mutual likes
// observe each other, so exactly one match is created. Repeating a swipe returns the stored result.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || '|' || GREATEST($1::text, $2::text), 0))`, userID, targetID); err != nil {
		return swipeOutcome{}, err
	}

	var existing string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, userID, targetID).Scan(&existing)
	switch {
	case err == nil:
		matchID, err := findMatch(ctx, tx, userID, targetID)
		if err != nil {
			return swipeOutcome{}, err
		}
		return swipeOutcome{Action: existing, MatchID: matchID}, tx.Commit(ctx)
	case !errors.Is(err, pgx.ErrNoRows):
		return swipeOutcome{}, err
	}

	var eligible, complete bool
	if err := tx.QueryRow(ctx, `
		SELECT `+viewerComplete+`,
		       EXISTS (SELECT 1 FROM profiles c JOIN preferences cp ON cp.user_id = c.user_id
		               WHERE c.user_id = $2 AND `+eligibility+`)
		FROM profiles v JOIN preferences vp ON vp.user_id = v.user_id
		WHERE v.user_id = $1`, userID, targetID).Scan(&complete, &eligible); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return swipeOutcome{}, ErrRepoIncomplete
		}
		return swipeOutcome{}, err
	}
	if !complete {
		return swipeOutcome{}, ErrRepoIncomplete
	}
	if !eligible {
		return swipeOutcome{}, ErrRepoIneligible
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return swipeOutcome{}, err
	}

	out := swipeOutcome{Action: action}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')`, targetID, userID).Scan(&reciprocal); err != nil {
			return swipeOutcome{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id`, userID, targetID).Scan(&out.MatchID)
			switch {
			case err == nil:
				out.Created = true
				if _, err := tx.Exec(ctx, `INSERT INTO match_participants (match_id, user_id) VALUES ($1, $2), ($1, $3)`, out.MatchID, userID, targetID); err != nil {
					return swipeOutcome{}, err
				}
			case errors.Is(err, pgx.ErrNoRows):
				if out.MatchID, err = findMatch(ctx, tx, userID, targetID); err != nil {
					return swipeOutcome{}, err
				}
			default:
				return swipeOutcome{}, err
			}
		}
	}
	return out, tx.Commit(ctx)
}

func findMatch(ctx context.Context, tx pgx.Tx, a, b string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (r *PGRepository) ResetPasses(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM swipes WHERE swiper_id = $1 AND action = 'pass'`, userID)
	return err
}

// IsComplete mirrors auth's profileComplete: profile, preferences and at least one photo.
func (r *PGRepository) IsComplete(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&ok)
	return ok, err
}
