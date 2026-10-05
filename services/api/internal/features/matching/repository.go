package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotEligible   = errors.New("profile is not available")
	ErrAlreadySwiped = errors.New("you already reacted to this profile")
	ErrMatchNotFound = errors.New("match not found")
	ErrSelfSwipe     = errors.New("you cannot react to yourself")
)

// storedMatch is a freshly created match with the data needed to notify both users.
type storedMatch struct {
	ID             string
	ConversationID string
	CreatedAt      time.Time
	Notifications  map[string]notifications.Notification // by user id
	Names          map[string]string                     // first names by user id
}

type matchRow struct {
	ID             string
	ConversationID string
	CreatedAt      time.Time
	OtherUserID    string
	Last           *MessagePreview
	Unread         int
}

type Repository interface {
	Discover(ctx context.Context, userID string, limit int) ([]string, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (*storedMatch, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]matchRow, error)
	GetMatch(ctx context.Context, userID, matchID string) (matchRow, error)
	DeleteMatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
	TouchActive(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// candidateFilter holds the rules deciding whether profile "c" may be shown to (and liked by) "me".
// It is shared by the discovery feed and by swipe validation so the server never trusts the client's choice.
// Parameters: $1 = viewer id. Aliases: me/mp (viewer profile/preferences), c/cp (candidate).
//
// Rules, all mutual: gender interest, age range, distance (the smaller of both limits), a visible
// discoverable profile with location and at least one photo, and no block in either direction.
const candidateFilter = `
	c.is_discoverable
	AND c.latitude IS NOT NULL
	AND c.gender = ANY(mp.interested_in)
	AND me.gender = ANY(cp.interested_in)
	AND date_part('year', age(CURRENT_DATE, c.birth_date)) BETWEEN mp.min_age AND mp.max_age
	AND date_part('year', age(CURRENT_DATE, me.birth_date)) BETWEEN cp.min_age AND cp.max_age
	AND c.latitude BETWEEN me.latitude - LEAST(mp.max_distance_km, cp.max_distance_km) / 111.0
	                   AND me.latitude + LEAST(mp.max_distance_km, cp.max_distance_km) / 111.0
	AND haversine_km(me.latitude, me.longitude, c.latitude, c.longitude) <= LEAST(mp.max_distance_km, cp.max_distance_km)
	AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	AND NOT EXISTS (
	    SELECT 1 FROM blocks b
	    WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
	       OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))
`

// Discover returns the ids of the best next profiles for userID, never including anyone already swiped.
//
// Ranking lives in the ORDER BY below and is the single place to evolve the recommendation algorithm:
// people who already liked the viewer first, then shared interests, then proximity, then recent activity.
func (r *PGRepository) Discover(ctx context.Context, userID string, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.user_id
		FROM profiles me
		JOIN preferences mp ON mp.user_id = me.user_id
		JOIN profiles c ON c.user_id <> me.user_id
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users cu ON cu.id = c.user_id
		WHERE me.user_id = $1
		  AND `+candidateFilter+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
		ORDER BY
		  EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like') DESC,
		  (SELECT count(*) FROM user_interests a JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = c.user_id
		   WHERE a.user_id = me.user_id) DESC,
		  haversine_km(me.latitude, me.longitude, c.latitude, c.longitude) ASC,
		  cu.last_active_at DESC,
		  c.user_id
		LIMIT $2
	`, userID, limit)
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

func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'
	`, userID)
	return err
}

// Swipe records a like or pass and, on a reciprocal like, creates the match, its conversation and both
// notifications in the same transaction. A per-pair advisory lock makes two simultaneous reciprocal likes
// see each other, so exactly one match is created.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (*storedMatch, error) {
	if swiperID == targetID {
		return nil, ErrSelfSwipe
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := orderPair(swiperID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, low+high); err != nil {
		return nil, err
	}

	var eligible bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		  FROM profiles me
		  JOIN preferences mp ON mp.user_id = me.user_id
		  JOIN profiles c ON c.user_id = $2
		  JOIN preferences cp ON cp.user_id = c.user_id
		  WHERE me.user_id = $1
		    AND (`+candidateFilter+`
		         -- someone who already liked the viewer stays likeable even if their settings changed since
		         OR (EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = c.user_id AND s.target_id = me.user_id AND s.action = 'like')
		             AND NOT EXISTS (SELECT 1 FROM blocks b
		                 WHERE (b.blocker_id = me.user_id AND b.blocked_id = c.user_id)
		                    OR (b.blocker_id = c.user_id AND b.blocked_id = me.user_id))))
		)
	`, swiperID, targetID).Scan(&eligible)
	if err != nil {
		return nil, err
	}
	if !eligible {
		return nil, ErrNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadySwiped
	}

	var match *storedMatch
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, swiperID).Scan(&reciprocal); err != nil {
			return nil, err
		}
		if reciprocal {
			match, err = createMatch(ctx, tx, low, high)
			if err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return match, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, low, high string) (*storedMatch, error) {
	m := &storedMatch{Notifications: map[string]notifications.Notification{}, Names: map[string]string{}}
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
		RETURNING id, created_at
	`, low, high).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, match_id, direct_key) VALUES ('match', $1, $2)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id
	`, m.ID, low+":"+high).Scan(&m.ConversationID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING
	`, m.ConversationID, low, high); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `SELECT user_id, first_name FROM profiles WHERE user_id IN ($1, $2)`, low, high)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, err
		}
		m.Names[id] = name
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, pair := range [][2]string{{low, high}, {high, low}} {
		self, other := pair[0], pair[1]
		n, err := notifications.InsertTx(ctx, tx, self, notifications.TypeMatch, "Nouveau match",
			"Vous avez un match avec "+m.Names[other]+" !",
			map[string]any{"matchId": m.ID, "conversationId": m.ConversationID, "userId": other})
		if err != nil {
			return nil, err
		}
		m.Notifications[self] = n
	}
	return m, nil
}

const matchSelect = `
	SELECT m.id, c.id, m.created_at,
	       CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END,
	       lm.content, lm.sender_user_id, lm.created_at,
	       (SELECT count(*) FROM messages un
	         WHERE un.conversation_id = c.id AND un.recipient_user_id = $1 AND un.read_at IS NULL)
	FROM matches m
	JOIN conversations c ON c.match_id = m.id
	LEFT JOIN LATERAL (
	    SELECT content, sender_user_id, created_at FROM messages
	    WHERE conversation_id = c.id ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE (m.user_a = $1 OR m.user_b = $1)
	  AND NOT EXISTS (
	      SELECT 1 FROM blocks b
	      WHERE (b.blocker_id = m.user_a AND b.blocked_id = m.user_b)
	         OR (b.blocker_id = m.user_b AND b.blocked_id = m.user_a))
`

func scanMatchRow(row pgx.Row) (matchRow, error) {
	var m matchRow
	var content, sender *string
	var sentAt *time.Time
	if err := row.Scan(&m.ID, &m.ConversationID, &m.CreatedAt, &m.OtherUserID, &content, &sender, &sentAt, &m.Unread); err != nil {
		return matchRow{}, err
	}
	if content != nil && sender != nil && sentAt != nil {
		m.Last = &MessagePreview{Content: *content, SenderUserID: *sender, CreatedAt: *sentAt}
	}
	return m, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]matchRow, error) {
	rows, err := r.db.Query(ctx, matchSelect+`
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []matchRow{}
	for rows.Next() {
		m, err := scanMatchRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (matchRow, error) {
	m, err := scanMatchRow(r.db.QueryRow(ctx, matchSelect+` AND m.id = $2`, userID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return matchRow{}, ErrMatchNotFound
	}
	return m, err
}

// DeleteMatch unmatches: the conversation and its messages cascade away, while the swipes remain so the
// two people never reappear in each other's feed.
func (r *PGRepository) DeleteMatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $2 AND (user_a = $1 OR user_b = $1)
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
