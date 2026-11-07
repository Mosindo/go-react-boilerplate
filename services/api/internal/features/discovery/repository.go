package discovery

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileIncomplete = errors.New("discovery: complete your profile first")
	ErrTargetUnavailable = errors.New("discovery: profile unavailable")
	ErrAlreadyDecided    = errors.New("discovery: already decided")
	ErrMatchNotFound     = errors.New("discovery: match not found")
)

type Repository interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error)
	// Swipe records the decision atomically and, when the like is reciprocal, creates the match,
	// its conversation and notifications. It returns the match (nil when none).
	Swipe(ctx context.Context, from, to, action string) (*storedMatch, error)
	Matches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const completeSQL = `
	EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
	AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
	AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`

func (r *PGRepository) IsComplete(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT `+completeSQL, userID).Scan(&ok)
	return ok, err
}

// discoveryRanking is the single place that decides the order of recommendations.
// Today: people who already liked you, then shared interests, then recent activity.
// Replace/extend this expression (or the query) to plug a smarter recommender in later.
const discoveryRanking = `liked_me DESC, shared DESC, u.last_active_at DESC, c.user_id`

// Candidates returns up to `limit` user ids the viewer has not decided on yet, filtered by
// mutual orientation, mutual age range, mutual distance, visibility and blocks. Decided
// profiles (like or pass) are excluded, so a profile never comes back once processed.
func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
		  SELECT p.gender, p.latitude, p.longitude, date_part('year', age(p.birth_date))::int AS age,
		         pr.interested_in, pr.age_min, pr.age_max, pr.max_distance_km
		  FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
		  WHERE p.user_id = $1
		)
		SELECT c.user_id
		FROM me, profiles c
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users u ON u.id = c.user_id
		CROSS JOIN LATERAL (SELECT
		    CASE WHEN me.latitude IS NOT NULL AND c.latitude IS NOT NULL
		         THEN geo_distance_km(me.latitude, me.longitude, c.latitude, c.longitude) END AS dist) d
		CROSS JOIN LATERAL (SELECT
		    EXISTS (SELECT 1 FROM likes x WHERE x.from_user_id = c.user_id AND x.to_user_id = $1 AND x.action = 'like') AS liked_me,
		    (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
		       WHERE a.user_id = $1 AND b.user_id = c.user_id) AS shared) s
		WHERE c.user_id <> $1
		  AND c.discoverable
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND c.gender = ANY (me.interested_in)
		  AND me.gender = ANY (cp.interested_in)
		  AND date_part('year', age(c.birth_date)) BETWEEN me.age_min AND me.age_max
		  AND me.age BETWEEN cp.age_min AND cp.age_max
		  AND (me.latitude IS NULL OR me.max_distance_km >= 20000 OR (d.dist IS NOT NULL AND d.dist <= me.max_distance_km))
		  AND (me.latitude IS NULL OR cp.max_distance_km >= 20000 OR d.dist IS NULL OR d.dist <= cp.max_distance_km)
		  AND NOT EXISTS (SELECT 1 FROM likes l WHERE l.from_user_id = $1 AND l.to_user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = c.user_id)
		                                           OR (b.blocker_id = c.user_id AND b.blocked_id = $1))
		ORDER BY `+discoveryRanking+`
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	return scanIDs(rows)
}

func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.from_user_id FROM likes l
		WHERE l.to_user_id = $1 AND l.action = 'like'
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = l.from_user_id)
		  AND EXISTS (SELECT 1 FROM profiles pf WHERE pf.user_id = l.from_user_id)
		  AND NOT EXISTS (SELECT 1 FROM likes mine WHERE mine.from_user_id = $1 AND mine.to_user_id = l.from_user_id)
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = l.from_user_id)
		                                           OR (b.blocker_id = l.from_user_id AND b.blocked_id = $1))
		ORDER BY l.created_at DESC, l.from_user_id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return scanIDs(rows)
}

func scanIDs(rows pgx.Rows) ([]string, error) {
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

func (r *PGRepository) Swipe(ctx context.Context, from, to, action string) (*storedMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise decisions of the same pair: if A and B like each other at the same moment, exactly
	// one of the two transactions observes the other's like and creates the (single) match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || GREATEST($1::text, $2::text), 0))`, from, to); err != nil {
		return nil, err
	}

	var targetOK bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  WHERE p.user_id = $2 AND p.discoverable
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1)))`,
		from, to).Scan(&targetOK); err != nil {
		return nil, err
	}
	if !targetOK {
		return nil, ErrTargetUnavailable
	}

	if _, err := tx.Exec(ctx, `INSERT INTO likes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, from, to, action); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyDecided
		}
		return nil, err
	}
	if action != "like" {
		return nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM likes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, to, from).Scan(&reciprocal); err != nil {
		return nil, err
	}
	if !reciprocal {
		if _, err := tx.Exec(ctx, `INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, 'like', $2, $2)`, to, from); err != nil {
			return nil, err
		}
		return nil, tx.Commit(ctx)
	}

	m := &storedMatch{OtherUserID: to}
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = matches.user_a_id
		RETURNING id, created_at`, from, to).Scan(&m.ID, &m.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = conversations.match_id
		RETURNING id`, m.ID).Scan(&m.ConversationID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING`, m.ConversationID, from, to); err != nil {
		return nil, err
	}
	// The earlier "like" notification of the other user is superseded by the match.
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE type = 'like' AND user_id = $1 AND actor_id = $2`, from, to); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)`,
		from, to, m.ConversationID); err != nil {
		return nil, err
	}
	return m, tx.Commit(ctx)
}

func (r *PGRepository) Matches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END, m.created_at
		FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a_id = $1 OR m.user_b_id = $1
		ORDER BY m.created_at DESC, m.id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedMatch{}
	for rows.Next() {
		var m storedMatch
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherUserID, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Unmatch deletes the match; the conversation and its messages cascade. The pass/like rows are kept
// so the pair never re-appears in discovery.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}
