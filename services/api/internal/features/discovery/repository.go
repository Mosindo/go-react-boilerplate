package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("not found")

type Repository interface {
	ViewerReady(ctx context.Context, viewerID string) (bool, error)
	Candidates(ctx context.Context, viewerID string, limit int) ([]storedCard, error)
	ProfileCard(ctx context.Context, viewerID, targetID string) (storedCard, error)
	Photos(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	Interests(ctx context.Context, viewerID string, userIDs []string) ([]storedInterest, error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// distanceSQL is the haversine distance in km between `me` and `c`.
const distanceSQL = `(6371 * 2 * asin(least(1, sqrt(
	power(sin(radians(c.latitude - me.latitude) / 2), 2) +
	cos(radians(me.latitude)) * cos(radians(c.latitude)) *
	power(sin(radians(c.longitude - me.longitude) / 2), 2)
))))`

const blockedEitherWaySQL = `EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_user_id = me.user_id AND b.blocked_user_id = c.user_id)
	   OR (b.blocker_user_id = c.user_id AND b.blocked_user_id = me.user_id)
)`

const sharedInterestsSQL = `(
	SELECT COUNT(*) FROM user_interests mine
	JOIN user_interests theirs ON theirs.interest_id = mine.interest_id AND theirs.user_id = c.user_id
	WHERE mine.user_id = me.user_id
)::int`

func (r *PGRepository) ViewerReady(ctx context.Context, viewerID string) (bool, error) {
	var ready bool
	err := r.dbPool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id WHERE p.user_id = $1
		)
	`, viewerID).Scan(&ready)
	return ready, err
}

// Candidates implements the recommendation query.
//
// Hard filters (all mutual): same organisation, discoverable and not hidden by
// moderation, at least one photo, gender and age preferences of both members,
// distance within both members' radius when the viewer shared a location, no
// block in either direction, and never a profile the viewer already swiped.
//
// Ranking is a transparent weighted score that can evolve independently:
// someone who already liked the viewer, recent activity, proximity and shared
// interests. Ties break on user id so pagination is stable.
func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]storedCard, error) {
	rows, err := r.dbPool.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude, u.organization_id,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $1
		),
		pool AS (
			SELECT c.user_id, c.first_name, c.birthdate, c.gender, c.bio, c.job_title, c.city, c.height_cm,
			       c.relationship_goal, c.show_distance, c.last_active_at, cu.is_demo,
			       CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN `+distanceSQL+` END AS distance_km,
			       LEAST(me.max_distance_km, cp.max_distance_km) AS radius_km,
			       me.latitude AS me_latitude,
			       `+sharedInterestsSQL+` AS shared_interests,
			       EXISTS (
			         SELECT 1 FROM swipes s
			         WHERE s.actor_user_id = c.user_id AND s.target_user_id = me.user_id AND s.decision = 'like'
			       ) AS liked_me
			FROM me
			JOIN profiles c ON c.user_id <> me.user_id
			JOIN users cu ON cu.id = c.user_id AND cu.organization_id = me.organization_id
			JOIN preferences cp ON cp.user_id = c.user_id
			WHERE c.is_discoverable
			  AND NOT c.moderation_hidden
			  AND c.gender = ANY (me.interested_in)
			  AND me.gender = ANY (cp.interested_in)
			  AND c.birthdate <= (CURRENT_DATE - make_interval(years => me.min_age))
			  AND c.birthdate > (CURRENT_DATE - make_interval(years => me.max_age + 1))
			  AND me.birthdate <= (CURRENT_DATE - make_interval(years => cp.min_age))
			  AND me.birthdate > (CURRENT_DATE - make_interval(years => cp.max_age + 1))
			  AND (
			    me.latitude IS NULL
			    OR (
			      c.latitude IS NOT NULL
			      AND c.latitude BETWEEN me.latitude - LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
			                         AND me.latitude + LEAST(me.max_distance_km, cp.max_distance_km) / 111.0
			    )
			  )
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
			  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_user_id = me.user_id AND s.target_user_id = c.user_id)
			  AND NOT `+blockedEitherWaySQL+`
		)
		SELECT user_id, first_name, birthdate, gender, bio, job_title, city, height_cm, relationship_goal,
		       distance_km, show_distance, shared_interests, is_demo
		FROM pool
		WHERE me_latitude IS NULL OR distance_km <= radius_km
		ORDER BY
		  (CASE WHEN liked_me THEN 1.5 ELSE 0 END)
		  + 1.0 / (1 + EXTRACT(EPOCH FROM (NOW() - last_active_at)) / 86400.0)
		  + COALESCE(1.0 / (1 + distance_km / 10.0), 0)
		  + LEAST(shared_interests, 5) * 0.2 DESC,
		  user_id
		LIMIT $2
	`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedCard, 0, limit)
	for rows.Next() {
		var c storedCard
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &c.City, &c.HeightCm,
			&c.RelationshipGoal, &c.DistanceKm, &c.ShowDistance, &c.SharedInterests, &c.IsDemo); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ProfileCard returns a profile only when the viewer is allowed to see it:
// no block either way and the target is either visible in discovery or
// already matched with the viewer.
func (r *PGRepository) ProfileCard(ctx context.Context, viewerID, targetID string) (storedCard, error) {
	var c storedCard
	err := r.dbPool.QueryRow(ctx, `
		WITH me AS (
			SELECT u.id AS user_id, u.organization_id, p.latitude, p.longitude
			FROM users u LEFT JOIN profiles p ON p.user_id = u.id
			WHERE u.id = $1
		)
		SELECT c.user_id, c.first_name, c.birthdate, c.gender, c.bio, c.job_title, c.city, c.height_cm,
		       c.relationship_goal,
		       CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL THEN `+distanceSQL+` END,
		       c.show_distance,
		       `+sharedInterestsSQL+`,
		       EXISTS (
		         SELECT 1 FROM matches m
		         WHERE m.user_low_id = LEAST(me.user_id, c.user_id) AND m.user_high_id = GREATEST(me.user_id, c.user_id)
		       ) AS is_match,
		       cu.is_demo
		FROM me
		JOIN profiles c ON c.user_id = $2
		JOIN users cu ON cu.id = c.user_id AND cu.organization_id = me.organization_id
		WHERE NOT `+blockedEitherWaySQL+`
		  AND (
		    c.user_id = me.user_id
		    OR (c.is_discoverable AND NOT c.moderation_hidden)
		    OR EXISTS (
		      SELECT 1 FROM matches m
		      WHERE m.user_low_id = LEAST(me.user_id, c.user_id) AND m.user_high_id = GREATEST(me.user_id, c.user_id)
		    )
		  )
	`, viewerID, targetID).Scan(&c.UserID, &c.FirstName, &c.Birthdate, &c.Gender, &c.Bio, &c.JobTitle, &c.City, &c.HeightCm,
		&c.RelationshipGoal, &c.DistanceKm, &c.ShowDistance, &c.SharedInterests, &c.IsMatch, &c.IsDemo)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedCard{}, ErrRepositoryNotFound
	}
	return c, err
}

// Photos loads the photos of every card in one query (no N+1).
func (r *PGRepository) Photos(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT user_id, id, width, height FROM photos
		WHERE user_id = ANY ($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedPhoto, 0)
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.UserID, &p.ID, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Interests loads the interests of every card in one query, flagging those
// shared with the viewer.
func (r *PGRepository) Interests(ctx context.Context, viewerID string, userIDs []string) ([]storedInterest, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT ui.user_id, i.slug, i.label,
		       EXISTS (SELECT 1 FROM user_interests v WHERE v.user_id = $1 AND v.interest_id = ui.interest_id)
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY ($2::uuid[])
		ORDER BY ui.user_id, i.label
	`, viewerID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]storedInterest, 0)
	for rows.Next() {
		var i storedInterest
		if err := rows.Scan(&i.UserID, &i.Slug, &i.Label, &i.Shared); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// TouchActivity refreshes last_active_at at most every 5 minutes.
func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.dbPool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
