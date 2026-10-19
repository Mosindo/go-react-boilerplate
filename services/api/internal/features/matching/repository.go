package matching

import (
	"context"
	"errors"
	"strings"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	LoadViewer(ctx context.Context, userID string) (Viewer, error)
	Discover(ctx context.Context, v Viewer, limit int) ([]profiles.PublicProfile, error)
	Swipe(ctx context.Context, from, to, action string) (SwipeResult, *matchCreated, error)
	GetCard(ctx context.Context, userID string) (profiles.PublicProfile, error)
	ListMatches(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Match, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID, conversationID string, err error)
	ListReceivedLikes(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]LikedBy, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) LoadViewer(ctx context.Context, userID string) (Viewer, error) {
	v := Viewer{UserID: userID}
	err := r.pool.QueryRow(ctx, `
		SELECT p.gender, date_part('year', age(p.birth_date))::int, p.lat, p.lng,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	`, userID).Scan(&v.Gender, &v.Age, &v.Lat, &v.Lng, &v.InterestedIn, &v.MinAge, &v.MaxAge, &v.MaxDistanceKm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Viewer{}, ErrProfileIncomplete
		}
		return Viewer{}, err
	}
	return v, nil
}

// distanceSQL is the haversine distance (km) between the viewer ($lat, $lng)
// and a candidate row aliased "p". NULL when either position is unknown.
const distanceSQL = `CASE WHEN %LAT% IS NULL OR p.lat IS NULL THEN NULL ELSE
	2 * 6371 * asin(sqrt(least(1.0,
		power(sin(radians(p.lat - %LAT%) / 2), 2) +
		cos(radians(%LAT%)) * cos(radians(p.lat)) * power(sin(radians(p.lng - %LNG%) / 2), 2)))) END`

var distanceExpr = strings.NewReplacer("%LAT%", "$8::float8", "%LNG%", "$9::float8").Replace(distanceSQL)

// Discover selects the next candidates for a viewer. Everything the viewer
// should never see again is excluded in SQL: self, hidden profiles, profiles
// already swiped (like or pass, hence also already matched), blocked pairs in
// both directions, candidates without photos, and anyone whose own preferences
// (gender, age range, distance) exclude the viewer.
//
// Ranking is the extension point for smarter recommendation: today people who
// already liked the viewer come first, then shared interests, then proximity.
func (r *PGRepository) Discover(ctx context.Context, v Viewer, limit int) ([]profiles.PublicProfile, error) {
	dist := distanceExpr
	rows, err := r.pool.Query(ctx, `
		SELECT `+profiles.CardColumns+`, d.km, pr.show_distance
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		CROSS JOIN LATERAL (SELECT `+dist+` AS km) d
		WHERE p.user_id <> $1
		  AND p.is_visible
		  AND p.gender = ANY($2::text[])
		  AND date_part('year', age(p.birth_date)) BETWEEN $3 AND $4
		  AND $5 = ANY(pr.interested_in)
		  AND $6 BETWEEN pr.min_age AND pr.max_age
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user = $1 AND s.to_user = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id)
		           OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		  AND ($7::int IS NULL OR d.km IS NULL OR d.km <= $7)
		  AND (pr.max_distance_km IS NULL OR d.km IS NULL OR d.km <= pr.max_distance_km)
		  AND ($7::int IS NULL OR $8::float8 IS NULL OR p.lat IS NULL
		       OR p.lat BETWEEN $8::float8 - $7::int / 111.0 AND $8::float8 + $7::int / 111.0)
		ORDER BY
		  (EXISTS (SELECT 1 FROM swipes s WHERE s.from_user = p.user_id AND s.to_user = $1 AND s.action = 'like'))::int * 100
		  + (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
		     WHERE a.user_id = p.user_id AND b.user_id = $1) * 10 DESC,
		  d.km ASC NULLS LAST,
		  p.user_id
		LIMIT $10
	`, v.UserID, v.InterestedIn, v.MinAge, v.MaxAge, v.Gender, v.Age, v.MaxDistanceKm, v.Lat, v.Lng, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []profiles.PublicProfile{}
	for rows.Next() {
		var km *float64
		var show bool
		p, err := profiles.ScanCard(rows, &km, &show)
		if err != nil {
			return nil, err
		}
		if km != nil && show {
			d := profiles.ApproxDistanceKm(*km)
			p.DistanceKm = &d
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetCard(ctx context.Context, userID string) (profiles.PublicProfile, error) {
	p, err := profiles.ScanCard(r.pool.QueryRow(ctx, `SELECT `+profiles.CardColumns+` FROM profiles p WHERE p.user_id = $1`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return profiles.PublicProfile{}, ErrTargetUnavailable
		}
		return profiles.PublicProfile{}, err
	}
	return p, nil
}

// Swipe records a decision atomically. A per-pair advisory lock serialises two
// users liking each other at the same instant, so exactly one of the two
// transactions observes the other's like and creates the (single) match.
func (r *PGRepository) Swipe(ctx context.Context, from, to, action string) (SwipeResult, *matchCreated, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`,
		from, to); err != nil {
		return SwipeResult{}, nil, err
	}

	var available bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			WHERE p.user_id = $2 AND p.is_visible
			  AND NOT EXISTS (SELECT 1 FROM blocks b
			       WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)))`,
		from, to).Scan(&available)
	if err != nil {
		if isInvalidUUID(err) {
			return SwipeResult{}, nil, ErrTargetUnavailable
		}
		return SwipeResult{}, nil, err
	}
	if !available {
		return SwipeResult{}, nil, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user, to_user, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user, to_user) DO NOTHING`, from, to, action)
	if err != nil {
		return SwipeResult{}, nil, err
	}
	if tag.RowsAffected() == 0 {
		// Repeated swipe: the first decision stands and nothing else changes.
		var matched bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid))`,
			from, to).Scan(&matched); err != nil {
			return SwipeResult{}, nil, err
		}
		return SwipeResult{Matched: matched, Duplicate: true}, nil, nil
	}
	if action != ActionLike {
		return SwipeResult{}, nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user = $2 AND to_user = $1 AND action = 'like')`,
		from, to).Scan(&reciprocal); err != nil {
		return SwipeResult{}, nil, err
	}
	if !reciprocal {
		return SwipeResult{}, nil, tx.Commit(ctx)
	}

	created := &matchCreated{}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id, created_at`, from, to).Scan(&created.MatchID, &created.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SwipeResult{Matched: true, Duplicate: true}, nil, tx.Commit(ctx)
	}
	if err != nil {
		return SwipeResult{}, nil, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, created.MatchID).Scan(&created.ConversationID); err != nil {
		return SwipeResult{}, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`,
		created.ConversationID, from, to); err != nil {
		return SwipeResult{}, nil, err
	}
	if _, err := notifications.InsertTx(ctx, tx, from, notifications.TypeNewMatch, to, created.ConversationID); err != nil {
		return SwipeResult{}, nil, err
	}
	if _, err := notifications.InsertTx(ctx, tx, to, notifications.TypeNewMatch, from, created.ConversationID); err != nil {
		return SwipeResult{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, nil, err
	}
	return SwipeResult{Matched: true}, created, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, c.id, m.created_at, `+profiles.CardColumns+`
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($2::timestamptz IS NULL OR (m.created_at, m.id) < ($2, $3::uuid))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $4
	`, userID, beforeAt, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Match{}
	for rows.Next() {
		var m Match
		p, err := profiles.ScanCardWithPrefix(rows, []any{&m.ID, &m.ConversationID, &m.CreatedAt})
		if err != nil {
			return nil, err
		}
		m.Profile = p
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var other, conv string
	err = tx.QueryRow(ctx, `
		SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END, c.id
		FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.id = $2 AND (m.user_a = $1 OR m.user_b = $1)
		FOR UPDATE OF m`, userID, matchID).Scan(&other, &conv)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", "", ErrMatchNotFound
		}
		return "", "", err
	}
	// Cascades remove the conversation, its members, messages and notifications.
	if _, err := tx.Exec(ctx, `DELETE FROM matches WHERE id = $1`, matchID); err != nil {
		return "", "", err
	}
	return other, conv, tx.Commit(ctx)
}

func (r *PGRepository) ListReceivedLikes(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]LikedBy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.created_at, `+profiles.CardColumns+`
		FROM swipes s
		JOIN profiles p ON p.user_id = s.from_user
		WHERE s.to_user = $1 AND s.action = 'like' AND p.is_visible
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.from_user = $1 AND mine.to_user = s.from_user)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = s.from_user) OR (b.blocker_id = s.from_user AND b.blocked_id = $1))
		  AND ($2::timestamptz IS NULL OR (s.created_at, s.from_user) < ($2, $3::uuid))
		ORDER BY s.created_at DESC, s.from_user DESC
		LIMIT $4
	`, userID, beforeAt, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []LikedBy{}
	for rows.Next() {
		var l LikedBy
		p, err := profiles.ScanCardWithPrefix(rows, []any{&l.LikedAt})
		if err != nil {
			return nil, err
		}
		l.PublicProfile = p
		out = append(out, l)
	}
	return out, rows.Err()
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
