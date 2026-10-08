package discovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotEligible = errors.New("target not eligible")
	ErrNoMatch     = errors.New("match not found")
)

// candidateSQL selects users that $1 may be shown, in both directions of compatibility:
// gender interest, age range, distance, visibility, no prior swipe, no block.
// %s is replaced by the select list; the optional target ($3) narrows to one user for eligibility checks.
//
// Ranking: the ORDER BY expression below is the single place to evolve the recommendation
// algorithm (shared interests, "already liked you", recent activity, deterministic per-viewer shuffle).
const candidateSQL = `
SELECT %s
FROM profiles me
JOIN preferences mp ON mp.user_id = me.user_id
JOIN profiles c ON c.user_id <> me.user_id
JOIN users u ON u.id = c.user_id AND u.status = 'active'
JOIN preferences cp ON cp.user_id = c.user_id
WHERE me.user_id = $1::uuid
  AND ($3::uuid IS NULL OR c.user_id = $3::uuid)
  AND cp.is_visible
  AND c.gender = ANY(mp.interested_in)
  AND me.gender = ANY(cp.interested_in)
  AND c.birthdate <= (CURRENT_DATE - mp.min_age * INTERVAL '1 year')::date
  AND c.birthdate >  (CURRENT_DATE - (mp.max_age + 1) * INTERVAL '1 year')::date
  AND me.birthdate <= (CURRENT_DATE - cp.min_age * INTERVAL '1 year')::date
  AND me.birthdate >  (CURRENT_DATE - (cp.max_age + 1) * INTERVAL '1 year')::date
  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = c.user_id)
  AND NOT EXISTS (SELECT 1 FROM blocks b
        WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
           OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
  AND (me.latitude IS NULL OR c.latitude IS NULL
       OR ((mp.max_distance_km IS NULL OR distance_km(me.latitude, me.longitude, c.latitude, c.longitude) <= mp.max_distance_km)
       AND (cp.max_distance_km IS NULL OR distance_km(me.latitude, me.longitude, c.latitude, c.longitude) <= cp.max_distance_km)))
`

const rankOrder = `
ORDER BY
  (SELECT count(*) FROM user_interests a JOIN user_interests b ON a.interest_id = b.interest_id
     WHERE a.user_id = c.user_id AND b.user_id = me.user_id) * 10
  + CASE WHEN EXISTS (SELECT 1 FROM swipes l WHERE l.from_user_id = c.user_id AND l.to_user_id = me.user_id AND l.action = 'like') THEN 25 ELSE 0 END
  + CASE WHEN u.last_active_at > NOW() - INTERVAL '7 days' THEN 5 ELSE 0 END DESC,
  md5(c.user_id::text || me.user_id::text)
LIMIT $2
`

type SwipeOutcome struct {
	AlreadyDone    bool
	Matched        bool
	MatchID        string
	ConversationID string
	OtherName      string
	MyName         string
}

type MatchRow struct {
	ID             string
	ConversationID string
	OtherID        string
	CreatedAt      time.Time
	HasMessages    bool
}

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]string, error)
	HasCompleteProfile(ctx context.Context, userID string) (bool, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error)
	ListMatches(ctx context.Context, userID string, limit int, afterAt *time.Time, afterID string) ([]MatchRow, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) HasCompleteProfile(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1) AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`,
		userID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, selectCandidates("c.user_id")+rankOrder, userID, limit, nil)
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

func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// One lock per unordered pair: two simultaneous mutual likes must observe each other.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, userID, targetID); err != nil {
		return SwipeOutcome{}, err
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM blocks WHERE (blocker_id = $1::uuid AND blocked_id = $2::uuid) OR (blocker_id = $2::uuid AND blocked_id = $1::uuid))`, userID, targetID).Scan(&blocked); err != nil {
		return SwipeOutcome{}, err
	}
	if blocked {
		return SwipeOutcome{}, ErrNotEligible
	}

	// Idempotency: a repeated swipe returns the existing state without side effects.
	var prior string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&prior)
	if err == nil {
		out := SwipeOutcome{AlreadyDone: true}
		out.MatchID, out.ConversationID = existingMatch(ctx, tx, userID, targetID)
		out.Matched = out.MatchID != ""
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SwipeOutcome{}, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS ("+selectCandidates("1")+" AND $2::int >= 1)", userID, 1, targetID).Scan(&eligible); err != nil {
		return SwipeOutcome{}, err
	}
	if !eligible {
		return SwipeOutcome{}, ErrNotEligible
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return SwipeOutcome{}, err
	}
	out := SwipeOutcome{}
	if action == "like" {
		var reverse bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $2 AND to_user_id = $1 AND action = 'like')`, userID, targetID).Scan(&reverse); err != nil {
			return SwipeOutcome{}, err
		}
		if reverse {
			if err := createMatch(ctx, tx, userID, targetID, &out); err != nil {
				return SwipeOutcome{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeOutcome{}, err
	}
	return out, nil
}

func existingMatch(ctx context.Context, tx pgx.Tx, a, b string) (matchID, conversationID string) {
	_ = tx.QueryRow(ctx, `
		SELECT m.id, c.id FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)`, a, b).Scan(&matchID, &conversationID)
	return matchID, conversationID
}

func createMatch(ctx context.Context, tx pgx.Tx, userID, targetID string, out *SwipeOutcome) error {
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a_id, user_b_id) DO NOTHING RETURNING id`, userID, targetID).Scan(&out.MatchID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // already matched (defensive; the pair lock makes this unreachable)
			out.MatchID, out.ConversationID = existingMatch(ctx, tx, userID, targetID)
			out.Matched = out.MatchID != ""
			return nil
		}
		return err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, out.MatchID).Scan(&out.ConversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, out.ConversationID, userID, targetID); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT (SELECT first_name FROM profiles WHERE user_id = $1), (SELECT first_name FROM profiles WHERE user_id = $2)`, userID, targetID).
		Scan(&out.MyName, &out.OtherName); err != nil {
		return err
	}
	for _, n := range []struct{ to, other, name string }{{userID, targetID, out.OtherName}, {targetID, userID, out.MyName}} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO notifications (user_id, type, title, body, data)
			VALUES ($1, 'match', $2, $3, jsonb_build_object('matchId', $4::text, 'conversationId', $5::text, 'userId', $6::text))`,
			n.to, "C'est un match !", "Vous et "+n.name+" vous plaisez mutuellement. Dites bonjour !", out.MatchID, out.ConversationID, n.other); err != nil {
			return err
		}
	}
	out.Matched = true
	return nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit int, afterAt *time.Time, afterID string) ([]MatchRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id,
		       CASE WHEN m.user_a_id = $1::uuid THEN m.user_b_id ELSE m.user_a_id END,
		       m.created_at,
		       EXISTS (SELECT 1 FROM messages x WHERE x.conversation_id = c.id)
		FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_a_id = $1::uuid OR m.user_b_id = $1::uuid)
		  AND ($3::timestamptz IS NULL OR (m.created_at, m.id) < ($3::timestamptz, $4::uuid))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2`, userID, limit, afterAt, nullUUID(afterID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MatchRow
	for rows.Next() {
		var m MatchRow
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherID, &m.CreatedAt, &m.HasMessages); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var other string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a_id = $1 OR user_b_id = $1)
		RETURNING CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNoMatch
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications WHERE type = 'match' AND data->>'matchId' = $1::text`, matchID); err != nil {
		return "", err
	}
	return other, tx.Commit(ctx)
}

func selectCandidates(selectList string) string { return fmt.Sprintf(candidateSQL, selectList) }
