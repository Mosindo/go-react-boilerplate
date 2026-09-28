package discovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/geo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTargetUnavailable = errors.New("discovery: target unavailable")
	ErrNotFound          = errors.New("discovery: not found")
)

type notificationPayload = notifications.Notification

// CandidateQuery carries every server-side filter of a discover request.
type CandidateQuery struct {
	Viewer    Viewer
	ViewerAge int
	Today     time.Time
	PoolSize  int
}

type Repository interface {
	GetViewer(ctx context.Context, userID string) (Viewer, error)
	TouchActive(ctx context.Context, userID string)
	Candidates(ctx context.Context, q CandidateQuery) ([]CandidateRow, error)
	ProfileFor(ctx context.Context, viewer Viewer, targetID string) (*CandidateRow, error)
	Interests(ctx context.Context, userIDs []string) (map[string][]profiles.Interest, error)
	Photos(ctx context.Context, userIDs []string) (map[string][]profiles.Photo, error)
	ViewerInterestIDs(ctx context.Context, userID string) (map[int16]struct{}, error)
	Swipe(ctx context.Context, fromID, toID, action string) (SwipeOutcome, error)
	MatchRow(ctx context.Context, viewerID, matchID string) (matches.Row, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) GetViewer(ctx context.Context, userID string) (Viewer, error) {
	v := Viewer{UserID: userID}
	var (
		lat, lon   *float64
		minAge     int16
		maxAge     int16
		maxDist    int32
		interested []string
	)
	err := r.pool.QueryRow(ctx, `
		SELECT p.gender, p.birth_date, p.latitude, p.longitude, p.is_discoverable,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id),
		       COALESCE(pr.interested_in, ARRAY['man','woman','non_binary']),
		       COALESCE(pr.min_age, 18), COALESCE(pr.max_age, 99), COALESCE(pr.max_distance_km, 50)
		FROM profiles p
		LEFT JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1`, userID).Scan(
		&v.Gender, &v.BirthDate, &lat, &lon, &v.Discoverable, &v.HasPhoto,
		&interested, &minAge, &maxAge, &maxDist)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, nil
	}
	if err != nil {
		return Viewer{}, err
	}
	v.HasProfile = true
	if lat != nil && lon != nil {
		v.HasLocation, v.Lat, v.Lon = true, *lat, *lon
	}
	v.Prefs = profiles.Preferences{InterestedIn: interested, MinAge: int(minAge), MaxAge: int(maxAge), MaxDistanceKm: int(maxDist)}
	return v, nil
}

// TouchActive refreshes last_active_at at most once a minute (best effort).
func (r *PGRepository) TouchActive(ctx context.Context, userID string) {
	_, _ = r.pool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
}

const haversine = `2 * 6371.0 * asin(sqrt(least(1.0,
	power(sin(radians(p.latitude - $2::float8) / 2), 2) +
	cos(radians($2::float8)) * cos(radians(p.latitude)) * power(sin(radians(p.longitude - $3::float8) / 2), 2))))`

// Candidates applies, in SQL: self/discoverable/complete exclusion, swiped and
// blocked (both directions) exclusion, the viewer's gender/age/distance
// preferences and the candidate's reciprocal preferences. A lat/lon bounding box
// narrows the rows before the exact haversine distance is computed. Only the
// top `PoolSize` most recently active matches are returned for ranking.
func (r *PGRepository) Candidates(ctx context.Context, q CandidateQuery) ([]CandidateRow, error) {
	v := q.Viewer
	minLat, maxLat, minLon, maxLon, useLon := geo.BoundingBox(v.Lat, v.Lon, float64(v.Prefs.MaxDistanceKm))
	// Coarse, index-friendly birth date bounds (1 day of slack) plus the exact age check below.
	maxBirth := q.Today.AddDate(-v.Prefs.MinAge, 0, 1)
	minBirth := q.Today.AddDate(-(v.Prefs.MaxAge + 1), 0, -1)

	sql := fmt.Sprintf(`
		WITH cand AS (
		  SELECT p.user_id, p.first_name, p.birth_date, p.bio, p.location_label,
		         p.show_distance, p.last_active_at, pr.max_distance_km AS their_max,
		         %s AS dist_km
		  FROM profiles p
		  JOIN preferences pr ON pr.user_id = p.user_id
		  WHERE p.user_id <> $1
		    AND p.is_discoverable
		    AND p.latitude IS NOT NULL
		    AND p.latitude BETWEEN $4 AND $5
		    AND (NOT $6::bool OR p.longitude BETWEEN $7 AND $8)
		    AND p.gender = ANY($9::text[])
		    AND p.birth_date <= $10::date AND p.birth_date > $11::date
		    AND date_part('year', age($12::date, p.birth_date))::int BETWEEN $13 AND $14
		    AND $15::text = ANY(pr.interested_in)
		    AND pr.min_age <= $16 AND pr.max_age >= $16
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = $1 AND s.to_user_id = p.user_id)
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		         OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		)
		SELECT c.user_id, c.first_name, c.birth_date, c.bio, c.location_label, c.show_distance,
		       c.last_active_at, c.dist_km,
		       (SELECT count(*) FROM user_interests ui
		        JOIN user_interests mine ON mine.interest_id = ui.interest_id AND mine.user_id = $1
		        WHERE ui.user_id = c.user_id)
		FROM cand c
		WHERE c.dist_km <= $17 AND c.dist_km <= c.their_max
		ORDER BY c.last_active_at DESC, c.user_id
		LIMIT $18`, haversine)

	rows, err := r.pool.Query(ctx, sql,
		v.UserID, v.Lat, v.Lon, // $1..$3
		minLat, maxLat, useLon, minLon, maxLon, // $4..$8
		v.Prefs.InterestedIn,                                        // $9
		maxBirth, minBirth, q.Today, v.Prefs.MinAge, v.Prefs.MaxAge, // $10..$14
		v.Gender, q.ViewerAge, // $15, $16
		float64(v.Prefs.MaxDistanceKm), q.PoolSize) // $17, $18
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CandidateRow
	for rows.Next() {
		var c CandidateRow
		var dist float64
		var shared int64
		if err := rows.Scan(&c.UserID, &c.FirstName, &c.BirthDate, &c.Bio, &c.LocationLabel,
			&c.ShowDistance, &c.LastActiveAt, &dist, &shared); err != nil {
			return nil, err
		}
		c.DistanceKm = &dist
		c.SharedInterests = int(shared)
		out = append(out, c)
	}
	return out, rows.Err()
}

// ProfileFor returns the target if the viewer may see it: exists, is not the
// viewer, is not blocked in either direction, and is discoverable or matched.
func (r *PGRepository) ProfileFor(ctx context.Context, v Viewer, targetID string) (*CandidateRow, error) {
	var c CandidateRow
	var dist *float64
	var shared int64
	err := r.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT p.user_id, p.first_name, p.birth_date, p.bio, p.location_label, p.show_distance,
		       p.last_active_at,
		       CASE WHEN p.latitude IS NULL THEN NULL ELSE %s END,
		       (SELECT count(*) FROM user_interests ui
		        JOIN user_interests mine ON mine.interest_id = ui.interest_id AND mine.user_id = $1
		        WHERE ui.user_id = p.user_id)
		FROM profiles p
		WHERE p.user_id = $4 AND p.user_id <> $1
		  AND NOT EXISTS (
		    SELECT 1 FROM blocks b
		    WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		       OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		  AND (p.is_discoverable OR EXISTS (
		    SELECT 1 FROM matches m
		    WHERE m.user_a = LEAST(p.user_id, $1::uuid) AND m.user_b = GREATEST(p.user_id, $1::uuid)))`, haversine),
		v.UserID, v.Lat, v.Lon, targetID).Scan(
		&c.UserID, &c.FirstName, &c.BirthDate, &c.Bio, &c.LocationLabel, &c.ShowDistance,
		&c.LastActiveAt, &dist, &shared)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.DistanceKm = dist
	c.SharedInterests = int(shared)
	return &c, nil
}

func (r *PGRepository) Interests(ctx context.Context, ids []string) (map[string][]profiles.Interest, error) {
	return profiles.InterestsFor(ctx, r.pool, ids)
}

func (r *PGRepository) Photos(ctx context.Context, ids []string) (map[string][]profiles.Photo, error) {
	return profiles.PhotosFor(ctx, r.pool, ids)
}

func (r *PGRepository) ViewerInterestIDs(ctx context.Context, userID string) (map[int16]struct{}, error) {
	rows, err := r.pool.Query(ctx, `SELECT interest_id FROM user_interests WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int16]struct{}{}
	for rows.Next() {
		var id int16
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

func (r *PGRepository) MatchRow(ctx context.Context, viewerID, matchID string) (matches.Row, error) {
	return matches.RowByID(ctx, r.pool, viewerID, matchID)
}

// Swipe records a swipe and, on a reciprocal like, creates the match, its
// conversation and both notifications, all in one transaction.
//
// Race safety: an advisory transaction lock keyed on the unordered pair makes
// concurrent swipes of the same two users run one after the other, so the
// second transaction's reciprocal-like read always sees the first's committed
// swipe. Independently, the match insert is ordered (LEAST/GREATEST) and
// ON CONFLICT DO NOTHING, so even without the lock a pair can never get two
// matches; the existing row is re-read when the insert loses.
func (r *PGRepository) Swipe(ctx context.Context, fromID, toID, action string) (SwipeOutcome, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`,
		fromID, toID); err != nil {
		return SwipeOutcome{}, err
	}

	// Idempotency: a repeated swipe returns what was stored.
	var stored string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, fromID, toID).Scan(&stored)
	if err == nil {
		out := SwipeOutcome{Action: stored}
		if stored == ActionLike {
			var matchID string
			err := tx.QueryRow(ctx, `
				SELECT id FROM matches
				WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)`, fromID, toID).Scan(&matchID)
			if err == nil {
				out.Matched, out.MatchID = true, matchID
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return SwipeOutcome{}, err
			}
		}
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeOutcome{}, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2 AND p.is_discoverable
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		         OR (b.blocker_id = p.user_id AND b.blocked_id = $1)))`, fromID, toID).Scan(&eligible); err != nil {
		return SwipeOutcome{}, err
	}
	if !eligible {
		return SwipeOutcome{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING`, fromID, toID, action)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return SwipeOutcome{}, ErrTargetUnavailable
		}
		return SwipeOutcome{}, err
	}
	out := SwipeOutcome{Action: action}
	if tag.RowsAffected() == 0 {
		// Lost an (impossible under the lock) race: report what is stored.
		return SwipeOutcome{}, ErrNotFound
	}
	if action != ActionLike {
		return out, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $2 AND to_user_id = $1 AND action = 'like')`,
		fromID, toID).Scan(&reciprocal); err != nil {
		return SwipeOutcome{}, err
	}
	if !reciprocal {
		return out, tx.Commit(ctx)
	}

	matchID, conversationID, created, err := matches.CreateInTx(ctx, tx, fromID, toID)
	if err != nil {
		return SwipeOutcome{}, err
	}
	out.Matched, out.MatchID, out.Created = true, matchID, created
	if created {
		names, err := firstNames(ctx, tx, fromID, toID)
		if err != nil {
			return SwipeOutcome{}, err
		}
		out.Notifications = map[string]notificationPayload{}
		for _, pair := range [][2]string{{fromID, toID}, {toID, fromID}} {
			n, err := notifications.Create(ctx, tx, notifications.NewNotification{
				UserID: pair[0],
				Type:   notifications.TypeMatch,
				Title:  "It's a match!",
				Body:   "You and " + names[pair[1]] + " liked each other.",
				Data:   notifications.Data{MatchID: matchID, ConversationID: conversationID, UserID: pair[1]},
			})
			if err != nil {
				return SwipeOutcome{}, err
			}
			out.Notifications[pair[0]] = n
		}
	}
	return out, tx.Commit(ctx)
}

func firstNames(ctx context.Context, tx pgx.Tx, a, b string) (map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT user_id, first_name FROM profiles WHERE user_id = ANY($1::uuid[])`, []string{a, b})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}
