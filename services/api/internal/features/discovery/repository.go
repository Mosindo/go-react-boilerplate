package discovery

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotOnboarded      = errors.New("viewer profile incomplete")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrConflictingSwipe  = errors.New("already swiped differently")
	ErrMatchNotFound     = errors.New("match not found")
)

// SwipeResult is the repository-level outcome of a swipe.
type SwipeResult struct {
	Action           string
	AlreadyProcessed bool
	Matched          bool
	MatchID          string
	ConversationID   string
	// For a new match: what each user is shown about the other one, and the
	// notification created for each of them.
	Cards         map[string]MatchUser
	Notifications map[string]notifications.Notification
}

type Unmatched struct {
	MatchID        string
	ConversationID string
	OtherUserID    string
}

type Repository interface {
	Candidates(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error)
	Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error)
	Unmatch(ctx context.Context, userID, matchID string) (Unmatched, error)
	UnmatchByConversation(ctx context.Context, userID, conversationID string) (Unmatched, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// candidateQuery selects profiles that are mutually compatible with the viewer
// (genders, age ranges, distance), have not been swiped yet, are not blocked in
// either direction and own at least one photo. All filtering happens in SQL so a
// page costs one query (plus two to load photos/interests for the page).
const candidateQuery = `
WITH me AS (
	SELECT p.user_id, p.gender, p.birth_date, p.latitude, p.longitude, p.last_active_at,
	       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
	FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
	WHERE p.user_id = $1 AND p.latitude IS NOT NULL
), cand AS (
	SELECT t.user_id, t.first_name, t.birth_date, t.gender, t.bio, t.city, t.show_distance, t.last_active_at,
	       haversine_km(me.latitude, me.longitude, t.latitude, t.longitude) AS dist,
	       LEAST(me.max_distance_km, tp.max_distance_km)::float8 AS max_dist,
	       (SELECT COUNT(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id
	         WHERE a.user_id = me.user_id AND b.user_id = t.user_id)::float8 AS shared,
	       EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = t.user_id AND s.to_user_id = me.user_id AND s.action = 'like') AS liked_me,
	       GREATEST(EXTRACT(EPOCH FROM (NOW() - t.last_active_at)) / 86400.0, 0) AS idle_days
	FROM me
	JOIN profiles t ON t.user_id <> me.user_id AND t.is_visible AND t.latitude IS NOT NULL
	JOIN preferences tp ON tp.user_id = t.user_id
	WHERE t.gender = ANY (me.interested_in)
	  AND me.gender = ANY (tp.interested_in)
	  -- candidate age within my range, my age within theirs (age = whole years since birth_date)
	  AND t.birth_date <= (CURRENT_DATE - make_interval(years => me.min_age))::date
	  AND t.birth_date >  (CURRENT_DATE - make_interval(years => me.max_age + 1))::date
	  AND me.birth_date <= (CURRENT_DATE - make_interval(years => tp.min_age))::date
	  AND me.birth_date >  (CURRENT_DATE - make_interval(years => tp.max_age + 1))::date
	  -- cheap latitude window first (indexed), exact great-circle distance after
	  AND t.latitude BETWEEN me.latitude - me.max_distance_km / 111.0 AND me.latitude + me.max_distance_km / 111.0
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = me.user_id AND s.to_user_id = t.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	                  WHERE (b.blocker_id = me.user_id AND b.blocked_id = t.user_id)
	                     OR (b.blocker_id = t.user_id AND b.blocked_id = me.user_id))
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
)
SELECT user_id::text, first_name, birth_date, gender, bio, city, show_distance, dist
FROM cand
WHERE dist <= max_dist
ORDER BY ` + rankExpression + `
LIMIT $2`

func (r *PGRepository) Candidates(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error) {
	rows, err := r.pool.Query(ctx, candidateQuery, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	var out []*profiles.PublicProfile
	for rows.Next() {
		var (
			p        profiles.PublicProfile
			birth    time.Time
			showDist bool
			dist     float64
		)
		if err := rows.Scan(&p.ID, &p.FirstName, &birth, &p.Gender, &p.Bio, &p.City, &showDist, &dist); err != nil {
			return nil, err
		}
		p.Age = profiles.AgeAt(birth, now)
		if showDist {
			d := profiles.ApproxDistance(dist)
			p.DistanceKm = &d
		}
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if err := profiles.Hydrate(ctx, r.pool, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Swipe records a like/pass and creates the match when the interest is mutual.
// An advisory lock on the user pair serialises two people liking each other at
// the same moment, so exactly one match (and conversation) is ever created.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	low, high := orderPair(userID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, low+":"+high); err != nil {
		return SwipeResult{}, err
	}

	var onboarded bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
		   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`, userID).Scan(&onboarded); err != nil {
		return SwipeResult{}, err
	}
	if !onboarded {
		return SwipeResult{}, ErrNotOnboarded
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks b
		               WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))`,
		userID, targetID).Scan(&blocked); err != nil {
		return SwipeResult{}, err
	}
	if blocked {
		return SwipeResult{}, ErrTargetUnavailable
	}

	result := SwipeResult{Action: action}

	var existingAction string
	err = tx.QueryRow(ctx, `SELECT action FROM swipes WHERE from_user_id = $1 AND to_user_id = $2`, userID, targetID).Scan(&existingAction)
	switch {
	case err == nil:
		if existingAction != action {
			return SwipeResult{}, ErrConflictingSwipe
		}
		result.AlreadyProcessed = true
		matchID, convID, err := findMatch(ctx, tx, low, high)
		if err != nil {
			return SwipeResult{}, err
		}
		result.Matched, result.MatchID, result.ConversationID = matchID != "", matchID, convID
		if result.Matched {
			card, err := cardFor(ctx, tx, targetID)
			if err != nil {
				return SwipeResult{}, err
			}
			result.Cards = map[string]MatchUser{userID: card}
		}
		return result, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return SwipeResult{}, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles t
			WHERE t.user_id = $2
			  AND NOT EXISTS (SELECT 1 FROM blocks b
			                  WHERE (b.blocker_id = $1 AND b.blocked_id = t.user_id)
			                     OR (b.blocker_id = t.user_id AND b.blocked_id = $1))
			  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = t.user_id)
			  AND (t.is_visible OR EXISTS (
			        SELECT 1 FROM swipes s WHERE s.from_user_id = t.user_id AND s.to_user_id = $1 AND s.action = 'like')))
	`, userID, targetID).Scan(&available); err != nil {
		return SwipeResult{}, err
	}
	if !available {
		return SwipeResult{}, ErrTargetUnavailable
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, userID, targetID, action); err != nil {
		return SwipeResult{}, err
	}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')
		`, targetID, userID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			if err := createMatch(ctx, tx, &result, userID, targetID, low, high); err != nil {
				return SwipeResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return result, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, result *SwipeResult, userID, targetID, low, high string) error {
	var matchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_low_id, user_high_id) VALUES ($1, $2)
		ON CONFLICT (user_low_id, user_high_id) DO UPDATE SET user_low_id = matches.user_low_id
		RETURNING id`, low, high).Scan(&matchID); err != nil {
		return err
	}
	var convID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = conversations.match_id
		RETURNING id`, matchID).Scan(&convID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, convID, userID, targetID); err != nil {
		return err
	}

	cards := map[string]MatchUser{}
	for viewer, other := range map[string]string{userID: targetID, targetID: userID} {
		card, err := cardFor(ctx, tx, other)
		if err != nil {
			return err
		}
		cards[viewer] = card
	}

	notifs := map[string]notifications.Notification{}
	for viewer, card := range cards {
		n, err := notifications.Insert(ctx, tx, viewer, notifications.TypeMatch, map[string]any{
			"matchId":        matchID,
			"conversationId": convID,
			"userId":         card.ID,
			"firstName":      card.FirstName,
		})
		if err != nil {
			return err
		}
		notifs[viewer] = n
	}

	result.Matched, result.MatchID, result.ConversationID = true, matchID, convID
	result.Cards, result.Notifications = cards, notifs
	return nil
}

func findMatch(ctx context.Context, tx pgx.Tx, low, high string) (matchID, convID string, err error) {
	err = tx.QueryRow(ctx, `
		SELECT m.id::text, COALESCE(c.id::text, '') FROM matches m
		LEFT JOIN conversations c ON c.match_id = m.id
		WHERE m.user_low_id = $1 AND m.user_high_id = $2`, low, high).Scan(&matchID, &convID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	return matchID, convID, err
}

func cardFor(ctx context.Context, tx pgx.Tx, userID string) (MatchUser, error) {
	card := MatchUser{ID: userID}
	var photoID *string
	err := tx.QueryRow(ctx, `
		SELECT p.first_name, (SELECT ph.id::text FROM photos ph WHERE ph.user_id = p.user_id ORDER BY ph.position LIMIT 1)
		FROM profiles p WHERE p.user_id = $1`, userID).Scan(&card.FirstName, &photoID)
	if err != nil {
		return MatchUser{}, err
	}
	if photoID != nil {
		u := profiles.PhotoURL(*photoID)
		card.PhotoURL = &u
	}
	return card, nil
}

// Unmatch deletes a match the user belongs to; the conversation and its
// messages disappear with it (foreign key cascade).
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (Unmatched, error) {
	var u Unmatched
	u.MatchID = matchID
	err := r.pool.QueryRow(ctx, `
		WITH conv AS (SELECT id FROM conversations WHERE match_id = $2),
		del AS (
			DELETE FROM matches WHERE id = $2 AND (user_low_id = $1 OR user_high_id = $1)
			RETURNING CASE WHEN user_low_id = $1 THEN user_high_id ELSE user_low_id END AS other
		)
		SELECT del.other::text, COALESCE((SELECT id::text FROM conv), '') FROM del
	`, userID, matchID).Scan(&u.OtherUserID, &u.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Unmatched{}, ErrMatchNotFound
	}
	return u, err
}

// UnmatchByConversation ends the match behind a conversation the user belongs to.
func (r *PGRepository) UnmatchByConversation(ctx context.Context, userID, conversationID string) (Unmatched, error) {
	var matchID string
	err := r.pool.QueryRow(ctx, `
		SELECT m.id::text FROM conversations c JOIN matches m ON m.id = c.match_id
		WHERE c.id = $2 AND (m.user_low_id = $1 OR m.user_high_id = $1)
	`, userID, conversationID).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Unmatched{}, ErrMatchNotFound
	}
	if err != nil {
		return Unmatched{}, err
	}
	return r.Unmatch(ctx, userID, matchID)
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
