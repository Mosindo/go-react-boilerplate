package discovery

import (
	"context"
	"errors"
	"sort"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errAlreadySwiped = platformerrors.Conflict("you already reacted to this profile")
	errNotEligible   = platformerrors.NotFound("profile not found")
)

type Repository interface {
	HasDiscoverableSetup(ctx context.Context, userID string) (bool, error)
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (*createdMatch, error)
	ListMatches(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]MatchItem, error)
	GetMatchItem(ctx context.Context, matchID, forUserID string) (MatchItem, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID, conversationID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// rankOrder is the single extension point for recommendation quality: today
// shared interests first, then proximity. Replace this clause (or pre-score in
// a CTE) to evolve the algorithm without touching the eligibility rules below.
func rankOrder(dist string) string {
	return `(SELECT count(*) FROM user_interests a
	   JOIN user_interests b ON b.interest_id = a.interest_id
	  WHERE a.user_id = me.user_id AND b.user_id = p.user_id) DESC,
	` + dist + ` ASC,
	p.user_id`
}

// eligibleSQL builds the query selecting the profiles `me` may be shown.
// These are the only rules deciding who can be swiped, enforced server side
// for both Candidates and Swipe: mutual gender/age/distance preferences,
// completed + discoverable profiles, nothing already swiped, no blocks.
func eligibleSQL(extraWhere, tail string) string {
	dist := profiles.DistanceKmSQL("p", "me")
	return `
	SELECT p.user_id::text
	FROM profiles me
	JOIN preferences mp ON mp.user_id = me.user_id
	JOIN profiles p ON p.user_id <> me.user_id
	JOIN preferences pr ON pr.user_id = p.user_id
	WHERE me.user_id = $1
	  AND me.latitude IS NOT NULL AND p.latitude IS NOT NULL
	  AND pr.discoverable
	  AND p.gender = ANY(mp.interested_in)
	  AND me.gender = ANY(pr.interested_in)
	  AND date_part('year', age(p.birth_date)) BETWEEN mp.min_age AND mp.max_age
	  AND date_part('year', age(me.birth_date)) BETWEEN pr.min_age AND pr.max_age
	  AND p.latitude BETWEEN me.latitude - mp.max_distance_km / 111.0 AND me.latitude + mp.max_distance_km / 111.0
	  AND ` + dist + ` <= LEAST(mp.max_distance_km, pr.max_distance_km)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = me.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	        WHERE (b.blocker_id = me.user_id AND b.blocked_id = p.user_id)
	           OR (b.blocker_id = p.user_id AND b.blocked_id = me.user_id))
	  ` + extraWhere + `
	ORDER BY ` + rankOrder(dist) + tail
}

func (r *PGRepository) HasDiscoverableSetup(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		  WHERE p.user_id = $1 AND p.latitude IS NOT NULL
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id))`, userID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, eligibleSQL("", " LIMIT $2"), userID, limit)
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

// Swipe records the reaction and, for a mutual like, creates the match and its
// conversation atomically. A per-pair advisory lock makes two simultaneous
// likes deterministic: exactly one of them sees the other and creates the match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (*createdMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	pair := []string{swiperID, targetID}
	sort.Strings(pair)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, pair[0]+":"+pair[1]); err != nil {
		return nil, err
	}

	var already bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2)`, swiperID, targetID).Scan(&already); err != nil {
		return nil, err
	}
	if already {
		return nil, errAlreadySwiped
	}

	var eligible string
	err = tx.QueryRow(ctx, eligibleSQL("AND p.user_id = $2::uuid", " LIMIT 1"), swiperID, targetID).Scan(&eligible)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errNotEligible
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)`, swiperID, targetID, action); err != nil {
		return nil, err
	}

	var result *createdMatch
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')`, targetID, swiperID).Scan(&reciprocal); err != nil {
			return nil, err
		}
		if reciprocal {
			m := &createdMatch{}
			if err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id::text`, swiperID, targetID).Scan(&m.MatchID); err != nil {
				return nil, err
			}
			if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id::text`, m.MatchID).Scan(&m.ConversationID); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, m.ConversationID, swiperID, targetID); err != nil {
				return nil, err
			}
			result = m
		}
	}
	return result, tx.Commit(ctx)
}

const matchSelect = `
	SELECT m.id::text, other.id::text, pf.first_name,
	       CASE WHEN ph.id IS NULL THEN NULL ELSE '/photos/' || ph.id::text || '/file' END,
	       c.id::text, m.created_at
	FROM matches m
	JOIN LATERAL (SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS id) other ON TRUE
	JOIN conversations c ON c.match_id = m.id
	LEFT JOIN profiles pf ON pf.user_id = other.id
	LEFT JOIN photos ph ON ph.user_id = other.id AND ph.position = 0`

func scanMatch(row interface{ Scan(dest ...any) error }) (MatchItem, error) {
	var m MatchItem
	err := row.Scan(&m.ID, &m.User.UserID, &m.User.FirstName, &m.User.PhotoURL, &m.ConversationID, &m.CreatedAt)
	return m, err
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]MatchItem, error) {
	var at *time.Time
	cursorID := "00000000-0000-0000-0000-000000000000"
	if cursor != nil {
		at = &cursor.At
		cursorID = cursor.ID
	}
	rows, err := r.db.Query(ctx, matchSelect+`
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($2::timestamptz IS NULL OR (m.created_at, m.id) < ($2::timestamptz, $3::uuid))
		ORDER BY m.created_at DESC, m.id DESC LIMIT $4`, userID, at, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MatchItem{}
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetMatchItem(ctx context.Context, matchID, forUserID string) (MatchItem, error) {
	return scanMatch(r.db.QueryRow(ctx, matchSelect+` WHERE m.id = $2 AND (m.user_a = $1 OR m.user_b = $1)`, forUserID, matchID))
}

// Unmatch removes the match (cascading to the conversation and messages). The
// swipes stay, so the pair never reappears in discovery.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, string, error) {
	var other, conversationID string
	err := r.db.QueryRow(ctx, `
		WITH gone AS (
		  DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
		  RETURNING id, user_a, user_b
		)
		SELECT CASE WHEN g.user_a = $1 THEN g.user_b ELSE g.user_a END::text,
		       (SELECT c.id::text FROM conversations c WHERE c.match_id = g.id)
		FROM gone g`, userID, matchID).Scan(&other, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", platformerrors.NotFound("match not found")
	}
	return other, conversationID, err
}
