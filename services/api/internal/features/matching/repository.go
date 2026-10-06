package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Viewer(ctx context.Context, userID string) (Viewer, error)
	Candidates(ctx context.Context, userID string, limit int) ([]Candidate, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]Candidate, error)
	Swipe(ctx context.Context, fromID, toID, action string) (SwipeResult, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// Viewer loads the discovering user, or ErrProfileIncomplete when onboarding
// is not finished (profile, location, preferences and at least one photo).
func (r *PGRepository) Viewer(ctx context.Context, userID string) (Viewer, error) {
	v := Viewer{UserID: userID, Interests: map[int16]struct{}{}}
	var hasPhoto bool
	var birthdate time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT p.gender, p.birthdate, pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1 AND p.latitude IS NOT NULL
	`, userID).Scan(&v.Gender, &birthdate, &v.Prefs.InterestedIn, &v.Prefs.MinAge, &v.Prefs.MaxAge, &v.Prefs.MaxDistanceKm, &hasPhoto)
	if errors.Is(err, pgx.ErrNoRows) {
		return Viewer{}, ErrProfileIncomplete
	}
	if err != nil {
		return Viewer{}, err
	}
	if !hasPhoto {
		return Viewer{}, ErrProfileIncomplete
	}
	v.Age = profiles.AgeOn(birthdate, time.Now())

	rows, err := r.pool.Query(ctx, `SELECT interest_id FROM user_interests WHERE user_id = $1`, userID)
	if err != nil {
		return Viewer{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int16
		if err := rows.Scan(&id); err != nil {
			return Viewer{}, err
		}
		v.Interests[id] = struct{}{}
	}
	return v, rows.Err()
}

// Candidates applies every hard filter in SQL and returns a bounded pool,
// most recently active first. Ranking happens in the service.
//
// Filters: mutual gender interest, mutual age range, mutual distance (bounding
// box on the indexed latitude first, then exact haversine), nobody already
// swiped, nobody blocked in either direction, at least one photo, and nobody
// with too many open reports.
func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.city, p.show_age, p.show_distance,
		       d.km, u.last_active_at,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = p.user_id AND s.to_user_id = me.user_id AND s.action = 'like')
		FROM profiles me
		JOIN preferences mp ON mp.user_id = me.user_id
		JOIN profiles p ON p.user_id <> me.user_id AND p.discoverable AND p.latitude IS NOT NULL
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		JOIN preferences pp ON pp.user_id = p.user_id
		CROSS JOIN LATERAL (SELECT haversine_km(me.latitude, me.longitude, p.latitude, p.longitude) AS km) d
		CROSS JOIN LATERAL (SELECT LEAST(mp.max_distance_km, pp.max_distance_km)::DOUBLE PRECISION AS max_km) lim
		WHERE me.user_id = $1
		  AND me.latitude IS NOT NULL
		  AND p.latitude BETWEEN me.latitude - lim.max_km / 111.0 AND me.latitude + lim.max_km / 111.0
		  AND d.km <= lim.max_km
		  AND p.gender = ANY (mp.interested_in)
		  AND me.gender = ANY (pp.interested_in)
		  AND age_years(p.birthdate) BETWEEN mp.min_age AND mp.max_age
		  AND age_years(me.birthdate) BETWEEN pp.min_age AND pp.max_age
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = p.user_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
		       OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id)
		  )
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND (SELECT count(DISTINCT rp.reporter_id) FROM reports rp WHERE rp.reported_id = p.user_id AND rp.status = 'open') < $3
		ORDER BY u.last_active_at DESC, p.user_id
		LIMIT $2
	`, userID, limit, ReportsHideThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidates(rows)
}

// LikesReceived lists people who liked the user and have not been answered yet.
func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]Candidate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id, p.first_name, p.birthdate, p.gender, p.bio, p.city, p.show_age, p.show_distance,
		       CASE WHEN p.latitude IS NOT NULL AND me.latitude IS NOT NULL
		            THEN haversine_km(me.latitude, me.longitude, p.latitude, p.longitude) END,
		       u.last_active_at,
		       TRUE
		FROM swipes s
		JOIN profiles p ON p.user_id = s.from_user_id
		JOIN users u ON u.id = p.user_id AND u.status = 'active'
		LEFT JOIN profiles me ON me.user_id = s.to_user_id
		WHERE s.to_user_id = $1
		  AND s.action = 'like'
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.from_user_id = $1 AND mine.to_user_id = s.from_user_id)
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = s.from_user_id)
		       OR (b.blocker_id = s.from_user_id AND b.blocked_id = $1)
		  )
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		ORDER BY s.created_at DESC, s.from_user_id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCandidates(rows)
}

func scanCandidates(rows pgx.Rows) ([]Candidate, error) {
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.City,
			&c.ShowAge, &c.ShowDistance, &c.DistanceKm, &c.LastActiveAt, &c.LikedMe); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Swipe records the decision and, on a reciprocal like, creates the match and
// its conversation. Everything happens in one transaction guarded by an
// advisory lock on the user pair, so two people liking each other at the same
// instant always produce exactly one match, and repeated requests are
// idempotent (the first decision wins).
func (r *PGRepository) Swipe(ctx context.Context, fromID, toID, action string) (SwipeResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, fromID, toID); err != nil {
		return SwipeResult{}, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.status = 'active'
			WHERE p.user_id = $2
			  AND NOT EXISTS (
			    SELECT 1 FROM blocks b
			    WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
			  )
		)
	`, fromID, toID).Scan(&available); err != nil {
		return SwipeResult{}, err
	}
	if !available {
		return SwipeResult{}, ErrTargetUnavailable
	}

	res := SwipeResult{Action: action}
	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING
	`, fromID, toID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		res.AlreadyProcessed = true
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, fromID, toID).Scan(&res.Action); err != nil {
			return SwipeResult{}, err
		}
	}

	if res.Action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $2 AND to_user_id = $1 AND action = 'like')
		`, fromID, toID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			if err := r.ensureMatch(ctx, tx, fromID, toID, &res); err != nil {
				return SwipeResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return res, nil
}

func (r *PGRepository) ensureMatch(ctx context.Context, tx pgx.Tx, fromID, toID string, res *SwipeResult) error {
	userA, userB := fromID, toID
	if userB < userA {
		userA, userB = userB, userA
	}

	var ref MatchRef
	var ended bool
	err := tx.QueryRow(ctx, `
		SELECT id, conversation_id, ended_at IS NOT NULL FROM matches WHERE user_a = $1::uuid AND user_b = $2::uuid
	`, userA, userB).Scan(&ref.ID, &ref.ConversationID, &ended)
	switch {
	case err == nil:
		// A match that was ended (unmatch/block) is never silently revived.
		if !ended {
			res.Matched = true
			res.Match = &ref
		}
		return nil
	case !errors.Is(err, pgx.ErrNoRows):
		return err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, created_by_user_id) VALUES ('direct', $1 || ':' || $2, $3)
		RETURNING id
	`, userA, userB, fromID).Scan(&ref.ConversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, ref.ConversationID, userA, userB); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b, conversation_id) VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id
	`, userA, userB, ref.ConversationID).Scan(&ref.ID); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `
		SELECT (SELECT first_name FROM profiles WHERE user_id = $1), (SELECT first_name FROM profiles WHERE user_id = $2)
	`, fromID, toID).Scan(&res.SelfName, &res.OtherName); err != nil {
		return err
	}
	res.Matched = true
	res.NewMatch = true
	res.Match = &ref
	return nil
}
