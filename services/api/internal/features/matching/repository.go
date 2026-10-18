package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("profile not available")
	ErrAlreadySwiped     = errors.New("profile already handled")
	ErrProfileIncomplete = errors.New("complete your profile and add a photo first")
	ErrMatchNotFound     = errors.New("match not found")
)

type Repository interface {
	ViewerReady(ctx context.Context, viewerID string) (bool, error)
	Candidates(ctx context.Context, viewerID string, limit int) ([]candidate, error)
	SharedInterests(ctx context.Context, viewerID string, ids []string) (map[string]int, error)
	Swipe(ctx context.Context, viewerID, targetID, action string) (swipeOutcome, error)
	GetCard(ctx context.Context, viewerID, targetID string) (profiles.Card, *float64, error)
	ResetPasses(ctx context.Context, viewerID string) (int64, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// meCTE loads the viewer's own profile and preferences: the reference point of
// every eligibility check. It yields no row when the viewer has no profile.
const meCTE = `
WITH me AS (
	SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude,
	       pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
	FROM profiles p
	JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1
)`

// distanceJoin computes the great-circle distance (haversine, km) between the
// viewer and the target without PostGIS.
const distanceJoin = `
CROSS JOIN LATERAL (SELECT 2 * 6371 * asin(sqrt(least(1,
	power(sin(radians(t.latitude - me.latitude) / 2), 2) +
	cos(radians(me.latitude)) * cos(radians(t.latitude)) *
	power(sin(radians(t.longitude - me.longitude) / 2), 2)))) AS km) d`

const notBlocked = `
	NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = me.user_id AND b.blocked_id = t.user_id)
		   OR (b.blocker_id = t.user_id AND b.blocked_id = me.user_id))`

// eligible holds every server-side rule deciding whether the viewer may be
// shown (or interact with) a target: visibility, mutual gender and age
// preferences, mutual distance limits, at least one photo, and no block in
// either direction.
const eligible = `
	t.is_visible
	AND t.gender = ANY(me.interested_in)
	AND me.gender = ANY(tp.interested_in)
	AND date_part('year', age(t.birth_date)) BETWEEN me.age_min AND me.age_max
	AND date_part('year', age(me.birth_date)) BETWEEN tp.age_min AND tp.age_max
	AND (me.max_distance_km IS NULL OR (
		t.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
		AND d.km <= me.max_distance_km))
	AND (tp.max_distance_km IS NULL OR d.km <= tp.max_distance_km)
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
	AND ` + notBlocked

func (r *PGRepository) ViewerReady(ctx context.Context, viewerID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)
	`, viewerID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]candidate, error) {
	rows, err := r.db.Query(ctx, meCTE+`
		SELECT t.user_id, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance, d.km
		FROM me
		JOIN profiles t ON t.user_id <> me.user_id
		JOIN preferences tp ON tp.user_id = t.user_id
		`+distanceJoin+`
		WHERE `+eligible+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = t.user_id)
		ORDER BY d.km ASC, t.user_id
		LIMIT $2
	`, viewerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.Card.UserID, &c.Card.FirstName, &c.Card.BirthDate, &c.Card.Gender,
			&c.Card.Bio, &c.Card.City, &c.ShowDistance, &c.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PGRepository) SharedInterests(ctx context.Context, viewerID string, ids []string) (map[string]int, error) {
	out := make(map[string]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT b.user_id, count(*)
		FROM user_interests a
		JOIN user_interests b ON b.interest_id = a.interest_id
		WHERE a.user_id = $1 AND b.user_id = ANY($2::uuid[])
		GROUP BY b.user_id
	`, viewerID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// Swipe records the viewer's decision and, on a reciprocal like, creates the
// match, its conversation and both notifications, atomically. An advisory lock
// on the unordered pair serialises two people liking each other at the same
// instant, so exactly one transaction sees the other's like and creates the match.
func (r *PGRepository) Swipe(ctx context.Context, viewerID, targetID, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(least($1::text, $2::text) || greatest($1::text, $2::text), 0))
	`, viewerID, targetID); err != nil {
		return swipeOutcome{}, err
	}

	var allowed bool
	if err := tx.QueryRow(ctx, meCTE+`
		SELECT EXISTS (
			SELECT 1 FROM me
			JOIN profiles t ON t.user_id = $2
			JOIN preferences tp ON tp.user_id = t.user_id
			`+distanceJoin+`
			WHERE t.user_id <> me.user_id AND `+eligible+`)
	`, viewerID, targetID).Scan(&allowed); err != nil {
		return swipeOutcome{}, err
	}
	if !allowed {
		return swipeOutcome{}, ErrNotFound
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, viewerID, targetID, action)
	if err != nil {
		return swipeOutcome{}, err
	}
	if tag.RowsAffected() == 0 {
		return swipeOutcome{}, ErrAlreadySwiped
	}
	if action != ActionLike {
		return swipeOutcome{}, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')
	`, targetID, viewerID).Scan(&reciprocal); err != nil {
		return swipeOutcome{}, err
	}
	if !reciprocal {
		return swipeOutcome{}, tx.Commit(ctx)
	}

	out := swipeOutcome{Matched: true}
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b)
		VALUES (least($1::uuid, $2::uuid), greatest($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id, created_at
	`, viewerID, targetID).Scan(&out.MatchID, &out.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return swipeOutcome{}, ErrAlreadySwiped
		}
		return swipeOutcome{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, out.MatchID).Scan(&out.ConversationID); err != nil {
		return swipeOutcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, out.ConversationID, viewerID, targetID); err != nil {
		return swipeOutcome{}, err
	}

	names := map[string]string{}
	rows, err := tx.Query(ctx, `SELECT user_id, first_name FROM profiles WHERE user_id IN ($1, $2)`, viewerID, targetID)
	if err != nil {
		return swipeOutcome{}, err
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return swipeOutcome{}, err
		}
		names[id] = name
	}
	rows.Close()
	if rows.Err() != nil {
		return swipeOutcome{}, rows.Err()
	}
	out.ViewerName, out.TargetName = names[viewerID], names[targetID]

	data := func(other string) map[string]string {
		return map[string]string{"matchId": out.MatchID, "conversationId": out.ConversationID, "userId": other}
	}
	if err := notifications.Insert(ctx, tx, viewerID, notifications.TypeMatch, "C'est un match !",
		"Vous avez matché avec "+out.TargetName+". Dites-lui bonjour !", data(targetID)); err != nil {
		return swipeOutcome{}, err
	}
	if err := notifications.Insert(ctx, tx, targetID, notifications.TypeMatch, "C'est un match !",
		"Vous avez matché avec "+out.ViewerName+". Dites-lui bonjour !", data(viewerID)); err != nil {
		return swipeOutcome{}, err
	}
	return out, tx.Commit(ctx)
}

// GetCard returns a target's profile if the viewer may see it: not blocked and
// either matched with the viewer or eligible for discovery. The second return
// value is the exact distance (nil when the target hides it).
func (r *PGRepository) GetCard(ctx context.Context, viewerID, targetID string) (profiles.Card, *float64, error) {
	var c profiles.Card
	var km float64
	var show bool
	err := r.db.QueryRow(ctx, meCTE+`
		SELECT t.user_id, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance, d.km
		FROM me
		JOIN profiles t ON t.user_id = $2
		JOIN preferences tp ON tp.user_id = t.user_id
		`+distanceJoin+`
		WHERE t.user_id <> me.user_id
		  AND `+notBlocked+`
		  AND (
			EXISTS (SELECT 1 FROM matches m WHERE m.user_a = least(me.user_id, t.user_id) AND m.user_b = greatest(me.user_id, t.user_id))
			OR (`+eligible+`)
		  )
	`, viewerID, targetID).Scan(&c.UserID, &c.FirstName, &c.BirthDate, &c.Gender, &c.Bio, &c.City, &show, &km)
	if errors.Is(err, pgx.ErrNoRows) {
		return profiles.Card{}, nil, ErrNotFound
	}
	if err != nil {
		return profiles.Card{}, nil, err
	}
	if !show {
		return c, nil, nil
	}
	return c, &km, nil
}

func (r *PGRepository) ResetPasses(ctx context.Context, viewerID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM swipes WHERE swiper_id = $1 AND action = 'pass'`, viewerID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Unmatch ends a match (and, by cascade, its conversation and messages). Both
// swipes stay, so the two people never see each other in discovery again.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2) RETURNING user_a, user_b
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
