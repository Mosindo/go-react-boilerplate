package matching

import (
	"context"
	"errors"
	"math"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoTargetUnavailable = errors.New("swipe target unavailable")
	errRepoNotFound          = errors.New("match not found")
)

type Repository interface {
	Viewer(ctx context.Context, userID string) (viewer, error)
	Candidates(ctx context.Context, v viewer, limit int) ([]Candidate, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error)
	ListMatches(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]matchRow, error)
	GetMatch(ctx context.Context, userID, matchID string) (matchRow, error)
	DeleteMatch(ctx context.Context, userID, matchID string) (matchRow, error)
	DeleteMatchBetween(ctx context.Context, userA, userB string) (*matchRow, error)
	LikesReceived(ctx context.Context, userID string, limit int) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) Viewer(ctx context.Context, userID string) (viewer, error) {
	v := viewer{UserID: userID}
	err := r.db.QueryRow(ctx, `
		SELECT p.gender, date_part('year', age(current_date, p.birthdate))::int,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
		       p.latitude, p.longitude,
		       EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&v.Gender, &v.Age, &v.InterestedIn, &v.MinAge, &v.MaxAge, &v.MaxDistanceKm, &v.Latitude, &v.Longitude, &v.Complete)
	if errors.Is(err, pgx.ErrNoRows) {
		return viewer{UserID: userID}, nil
	}
	return v, err
}

type box struct{ latMin, latMax, lngMin, lngMax float64 }

// boundingBox returns a lat/lng rectangle enclosing a radius, used to prune candidates via
// the location index before computing exact distances. Returns nil where a box is not meaningful.
func boundingBox(lat, lng float64, radiusKm int) *box {
	dLat := float64(radiusKm) / 111.0
	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.05 {
		return nil
	}
	dLng := float64(radiusKm) / (111.0 * cos)
	b := &box{latMin: lat - dLat, latMax: lat + dLat, lngMin: lng - dLng, lngMax: lng + dLng}
	if b.lngMin < -180 || b.lngMax > 180 {
		return nil
	}
	return b
}

// Candidates applies every hard rule of discovery in SQL:
// complete & active & discoverable profiles, mutual gender and age preferences,
// mutual distance limits, no previous swipe by the viewer, and no block in either direction.
// Without a location the viewer only meets members without one, so nobody's distance limit is bypassed.
func (r *PGRepository) Candidates(ctx context.Context, v viewer, limit int) ([]Candidate, error) {
	var latMin, latMax, lngMin, lngMax *float64
	if v.Latitude != nil {
		if b := boundingBox(*v.Latitude, *v.Longitude, v.MaxDistanceKm); b != nil {
			latMin, latMax, lngMin, lngMax = &b.latMin, &b.latMax, &b.lngMin, &b.lngMax
		}
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, u.last_active_at, d.km,
		       EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = p.user_id AND s.target_id = $1 AND s.action = 'like') AS liked_me,
		       (SELECT COUNT(*) FROM user_interests a
		          JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = $1
		         WHERE a.user_id = p.user_id)::int AS shared
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		JOIN preferences pr ON pr.user_id = p.user_id
		CROSS JOIN LATERAL (
			SELECT CASE WHEN $8::float8 IS NULL OR p.latitude IS NULL THEN NULL ELSE
				6371 * 2 * asin(sqrt(
					power(sin(radians(p.latitude - $8) / 2), 2) +
					cos(radians($8)) * cos(radians(p.latitude)) * power(sin(radians(p.longitude - $9) / 2), 2)
				))
			END AS km
		) d
		WHERE p.user_id <> $1
		  AND u.moderation_status = 'active'
		  AND p.discoverable
		  AND p.gender = ANY($2::text[])
		  AND $3 = ANY(pr.interested_in)
		  AND p.birthdate <= (current_date - make_interval(years => $4))::date
		  AND p.birthdate > (current_date - make_interval(years => $5 + 1))::date
		  AND $6 BETWEEN pr.min_age AND pr.max_age
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = p.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1)
		  )
		  AND (
			($8::float8 IS NULL AND p.latitude IS NULL)
			OR (
				p.latitude IS NOT NULL
				AND ($10::float8 IS NULL OR (p.latitude BETWEEN $10 AND $11 AND p.longitude BETWEEN $12 AND $13))
				AND d.km <= LEAST($7, pr.max_distance_km)
			)
		  )
		ORDER BY u.last_active_at DESC
		LIMIT $14
	`, v.UserID, v.InterestedIn, v.Gender, v.MinAge, v.MaxAge, v.Age, v.MaxDistanceKm,
		v.Latitude, v.Longitude, latMin, latMax, lngMin, lngMax, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Candidate, error) {
		var c Candidate
		err := row.Scan(&c.UserID, &c.LastActiveAt, &c.DistanceKm, &c.LikedMe, &c.SharedInterests)
		return c, err
	})
}

func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + ":" + b
}

// Swipe records a decision and creates the match when interest is mutual.
// A transaction-scoped advisory lock on the pair serialises concurrent swipes between the
// same two users, so two simultaneous likes always produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error) {
	var out swipeOutcome
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "swipe:"+pairKey(swiperID, targetID)); err != nil {
			return err
		}

		var eligible bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM profiles p
				JOIN users u ON u.id = p.user_id
				JOIN preferences pr ON pr.user_id = p.user_id
				JOIN profiles me ON me.user_id = $1
				JOIN preferences my_pr ON my_pr.user_id = $1
				WHERE p.user_id = $2 AND p.user_id <> $1
				  AND u.moderation_status = 'active'
				  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
				  AND p.gender = ANY(my_pr.interested_in)
				  AND me.gender = ANY(pr.interested_in)
				  AND (
					p.discoverable
					OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like')
				  )
				  AND NOT EXISTS (
					SELECT 1 FROM blocks b
					WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)
				  )
			)
		`, swiperID, targetID).Scan(&eligible); err != nil {
			return err
		}
		if !eligible {
			return errRepoTargetUnavailable
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
			ON CONFLICT (swiper_id, target_id) DO NOTHING
		`, swiperID, targetID, action)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			out.AlreadySwiped = true
			return nil
		}
		if action != ActionLike {
			return nil
		}

		var mutual bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
		`, swiperID, targetID).Scan(&mutual); err != nil {
			return err
		}
		if !mutual {
			return nil
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
			ON CONFLICT (user_a_id, user_b_id) DO NOTHING
			RETURNING id, created_at
		`, swiperID, targetID).Scan(&out.MatchID, &out.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already matched: nothing new to announce
		}
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO conversations (match_id) VALUES ($1) RETURNING id
		`, out.MatchID).Scan(&out.ConversationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		`, out.ConversationID, swiperID, targetID); err != nil {
			return err
		}
		out.Matched = true
		return nil
	})
	return out, err
}

const matchSelect = `
	SELECT m.id,
	       CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END,
	       c.id, m.created_at, c.last_message_at IS NOT NULL
	FROM matches m
	JOIN conversations c ON c.match_id = m.id
`

func scanMatch(row pgx.Row) (matchRow, error) {
	var m matchRow
	err := row.Scan(&m.ID, &m.OtherUserID, &m.ConversationID, &m.CreatedAt, &m.HasMessages)
	return m, err
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]matchRow, error) {
	args := []any{userID, limit}
	cond := ""
	if cursor != nil {
		cond = "AND (m.created_at, m.id) < ($3, $4::uuid)"
		args = append(args, cursor.Time, cursor.ID)
	}
	rows, err := r.db.Query(ctx, matchSelect+`
		JOIN users other ON other.id = CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1) `+cond+`
		  AND other.moderation_status <> 'suspended'
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = m.user_a_id AND b.blocked_id = m.user_b_id) OR (b.blocker_id = m.user_b_id AND b.blocked_id = m.user_a_id)
		  )
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (matchRow, error) { return scanMatch(row) })
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (matchRow, error) {
	m, err := scanMatch(r.db.QueryRow(ctx, matchSelect+`
		WHERE m.id = $2 AND (m.user_a_id = $1 OR m.user_b_id = $1)
	`, userID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return matchRow{}, errRepoNotFound
	}
	return m, err
}

// DeleteMatch removes a match the user belongs to; the conversation and its messages cascade.
func (r *PGRepository) DeleteMatch(ctx context.Context, userID, matchID string) (matchRow, error) {
	var m matchRow
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		var err error
		m, err = scanMatch(tx.QueryRow(ctx, matchSelect+`
			WHERE m.id = $2 AND (m.user_a_id = $1 OR m.user_b_id = $1) FOR UPDATE OF m
		`, userID, matchID))
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `DELETE FROM matches WHERE id = $1`, matchID)
		return err
	})
	return m, err
}

func (r *PGRepository) DeleteMatchBetween(ctx context.Context, userA, userB string) (*matchRow, error) {
	m, err := scanMatch(r.db.QueryRow(ctx, matchSelect+`
		WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
	`, userA, userB))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := r.db.Exec(ctx, `DELETE FROM matches WHERE id = $1`, m.ID); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT s.swiper_id
		FROM swipes s
		JOIN users u ON u.id = s.swiper_id
		WHERE s.target_id = $1 AND s.action = 'like'
		  AND u.moderation_status = 'active'
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.swiper_id = $1 AND mine.target_id = s.swiper_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id = s.swiper_id) OR (b.blocker_id = s.swiper_id AND b.blocked_id = $1)
		  )
		ORDER BY s.created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
