package discovery

import (
	"context"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrNotEligible = errors.New("profile not available")
)

type Repository interface {
	Touch(ctx context.Context, userID string) error
	Candidates(ctx context.Context, viewerID string, limit int) ([]Candidate, error)
	Visible(ctx context.Context, viewerID, targetID string) (Candidate, error)
	Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, *MatchInfo, error)
	Unmatch(ctx context.Context, userID, matchID string) (conversationID, otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// baseSelect joins the viewer ("me"), a candidate profile ("cp") and the
// candidate's preferences ("cpref"), computing a haversine distance in km.
// Eligibility is symmetric: each side must match the other's preferences.
// Ranking (see rankScore) is the only part meant to evolve with the algorithm.
const baseSelect = `
WITH me AS (
  SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude,
         pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
         date_part('year', age(p.birth_date))::int AS age
  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
  WHERE p.user_id = $1
)
SELECT cp.user_id, cp.first_name, date_part('year', age(cp.birth_date))::int, cp.gender, cp.bio, cp.city,
       CASE WHEN cp.show_distance THEN d.km END AS dist_km,
       sh.n AS shared
FROM me
CROSS JOIN profiles cp
JOIN preferences cpref ON cpref.user_id = cp.user_id
JOIN users cu ON cu.id = cp.user_id
CROSS JOIN LATERAL (
  SELECT CASE WHEN me.latitude IS NULL OR cp.latitude IS NULL THEN NULL
    ELSE 2 * 6371 * asin(sqrt(least(1.0,
      power(sin(radians(cp.latitude - me.latitude) / 2), 2) +
      cos(radians(me.latitude)) * cos(radians(cp.latitude)) *
      power(sin(radians(cp.longitude - me.longitude) / 2), 2)))) END AS km
) d
CROSS JOIN LATERAL (
  SELECT COUNT(*)::int AS n FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
  WHERE a.user_id = me.user_id AND b.user_id = cp.user_id
) sh
WHERE cp.user_id <> me.user_id
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = cp.user_id)
  AND cp.gender = ANY (me.interested_in)
  AND me.gender = ANY (cpref.interested_in)
  AND date_part('year', age(cp.birth_date)) BETWEEN me.min_age AND me.max_age
  AND me.age BETWEEN cpref.min_age AND cpref.max_age
  AND (me.max_distance_km IS NULL OR (d.km IS NOT NULL AND d.km <= me.max_distance_km))
  AND (cpref.max_distance_km IS NULL OR (d.km IS NOT NULL AND d.km <= cpref.max_distance_km))
  AND NOT EXISTS (SELECT 1 FROM blocks b
        WHERE (b.blocker_id = me.user_id AND b.blocked_id = cp.user_id)
           OR (b.blocker_id = cp.user_id AND b.blocked_id = me.user_id))
`

// rankScore orders the feed: shared interests, people who already liked the
// viewer, recent activity, then proximity. Swap this expression (or the whole
// query) to evolve recommendations without touching handlers or services.
const rankScore = `
  sh.n * 10
  + CASE WHEN EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = cp.user_id AND s.target_id = me.user_id AND s.action = 'like') THEN 25 ELSE 0 END
  + CASE WHEN cu.last_active_at > NOW() - INTERVAL '7 days' THEN 15
         WHEN cu.last_active_at > NOW() - INTERVAL '30 days' THEN 5 ELSE 0 END
  - LEAST(COALESCE(d.km, 0) / 10.0, 20)`

const feedFilter = `
  AND cp.is_discoverable
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = cp.user_id)
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = cp.user_id AND s.target_id = me.user_id AND s.action = 'pass')`

func (r *PGRepository) Touch(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_active_at = NOW() WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
	return err
}

func (r *PGRepository) scan(ctx context.Context, sql string, args ...any) ([]Candidate, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		var km *float64
		var shared int
		if err := rows.Scan(&c.ID, &c.FirstName, &c.Age, &c.Gender, &c.Bio, &c.City, &km, &shared); err != nil {
			return nil, err
		}
		c.SharedInterests = shared
		c.DistanceKm = ApproxKm(km)
		c.Interests = []string{}
		c.Photos = []PhotoRef{}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, r.hydrate(ctx, out)
}

// hydrate loads interests and photo refs for all candidates in two queries.
func (r *PGRepository) hydrate(ctx context.Context, cs []Candidate) error {
	if len(cs) == 0 {
		return nil
	}
	ids := make([]string, len(cs))
	idx := make(map[string]int, len(cs))
	for i, c := range cs {
		ids[i] = c.ID
		idx[c.ID] = i
	}
	rows, err := r.db.Query(ctx, `
		SELECT ui.user_id, i.slug FROM user_interests ui JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ANY($1::uuid[]) ORDER BY i.slug`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var uid, slug string
		if err := rows.Scan(&uid, &slug); err != nil {
			rows.Close()
			return err
		}
		cs[idx[uid]].Interests = append(cs[idx[uid]].Interests, slug)
	}
	rows.Close()
	prows, err := r.db.Query(ctx, `SELECT user_id, id, position FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY position`, ids)
	if err != nil {
		return err
	}
	defer prows.Close()
	for prows.Next() {
		var uid string
		var p PhotoRef
		if err := prows.Scan(&uid, &p.ID, &p.Position); err != nil {
			return err
		}
		cs[idx[uid]].Photos = append(cs[idx[uid]].Photos, p)
	}
	return prows.Err()
}

func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]Candidate, error) {
	q := baseSelect + feedFilter + " ORDER BY (" + rankScore + ") DESC, cp.user_id LIMIT $2"
	return r.scan(ctx, q, viewerID, limit)
}

// Visible returns a profile the viewer may open: an eligible candidate, or an
// active match. Blocks always win.
func (r *PGRepository) Visible(ctx context.Context, viewerID, targetID string) (Candidate, error) {
	res, err := r.scan(ctx, baseSelect+` AND cp.user_id = $2 AND (cp.is_discoverable OR EXISTS (
		SELECT 1 FROM matches m WHERE m.ended_at IS NULL
		  AND m.user_a = LEAST(me.user_id, cp.user_id) AND m.user_b = GREATEST(me.user_id, cp.user_id)))`, viewerID, targetID)
	if err != nil {
		if isInvalidUUID(err) {
			return Candidate{}, ErrNotFound
		}
		return Candidate{}, err
	}
	if len(res) == 0 {
		return Candidate{}, ErrNotFound
	}
	return res[0], nil
}

func (r *PGRepository) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, *MatchInfo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	lo, hi := orderPair(viewerID, targetID)
	// Serialize the two directions of a pair so simultaneous mutual likes
	// cannot both miss each other.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "pair:"+lo+":"+hi); err != nil {
		return SwipeResult{}, nil, err
	}

	var already bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2)`, viewerID, targetID).Scan(&already); err != nil {
		return SwipeResult{}, nil, err
	}
	if already {
		res := SwipeResult{AlreadySwiped: true}
		_ = tx.QueryRow(ctx, `
			SELECT m.id, c.id FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.user_a = $1 AND m.user_b = $2 AND m.ended_at IS NULL`, lo, hi).Scan(&res.MatchID, &res.ConversationID)
		res.Matched = res.MatchID != ""
		return res, nil, nil
	}

	// Server-side eligibility: the client may only act on profiles the feed
	// rules would have shown (or ones that already liked the viewer).
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (`+baseSelect+` AND cp.user_id = $2
		AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = cp.user_id AND s.target_id = me.user_id AND s.action = 'pass'))`, viewerID, targetID).Scan(&ok); err != nil {
		return SwipeResult{}, nil, err
	}
	if !ok {
		return SwipeResult{}, nil, ErrNotEligible
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)`, viewerID, targetID, action); err != nil {
		return SwipeResult{}, nil, err
	}
	res := SwipeResult{}
	var info *MatchInfo
	if action == "like" {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')`, targetID, viewerID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, nil, err
		}
		if reciprocal {
			mi := MatchInfo{UserA: lo, UserB: hi}
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
				ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id`, lo, hi).Scan(&mi.MatchID)
			if err == nil {
				if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, mi.MatchID).Scan(&mi.ConversationID); err != nil {
					return SwipeResult{}, nil, err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, mi.ConversationID, lo, hi); err != nil {
					return SwipeResult{}, nil, err
				}
				if err := tx.QueryRow(ctx, `SELECT (SELECT first_name FROM profiles WHERE user_id = $1), (SELECT first_name FROM profiles WHERE user_id = $2)`, lo, hi).Scan(&mi.NameA, &mi.NameB); err != nil {
					return SwipeResult{}, nil, err
				}
				info = &mi
				res.Matched, res.MatchID, res.ConversationID = true, mi.MatchID, mi.ConversationID
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return SwipeResult{}, nil, err
			}
		}
	}
	return res, info, tx.Commit(ctx)
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, string, error) {
	var convID, a, b string
	err := r.db.QueryRow(ctx, `
		UPDATE matches m SET ended_at = NOW(), ended_by = $1
		FROM conversations c
		WHERE m.id = $2 AND c.match_id = m.id AND m.ended_at IS NULL AND ($1 IN (m.user_a, m.user_b))
		RETURNING c.id, m.user_a, m.user_b`, userID, matchID).Scan(&convID, &a, &b)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	other := a
	if a == userID {
		other = b
	}
	return convID, other, nil
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// ApproxKm blurs a distance so exact positions cannot be triangulated:
// <1 → 1, <10 → rounded up to 1 km, otherwise rounded to the nearest 5 km.
func ApproxKm(km *float64) *int {
	if km == nil {
		return nil
	}
	var v int
	switch {
	case *km < 1:
		v = 1
	case *km < 10:
		v = int(math.Ceil(*km))
	default:
		v = int(math.Round(*km/5) * 5)
	}
	return &v
}
