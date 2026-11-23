package matching

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrViewerNotFound     = errors.New("viewer profile not found")
	ErrTargetNotAvailable = errors.New("profile not available")
	ErrAlreadySwiped      = errors.New("profile already swiped")
	ErrMatchNotFound      = errors.New("match not found")
)

type Repository interface {
	GetViewer(ctx context.Context, userID string) (Viewer, bool, error)
	TouchActivity(ctx context.Context, userID string) error
	FindCandidates(ctx context.Context, q DiscoveryQuery) ([]string, error)
	Swipe(ctx context.Context, q DiscoveryQuery, targetID, action string) (*StoredMatch, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// GetViewer loads the browsing user's criteria. ready=false means the profile
// is incomplete (no location, gender, birthdate, preferences or photo).
func (r *PGRepository) GetViewer(ctx context.Context, userID string) (Viewer, bool, error) {
	var v Viewer
	var gender *string
	var birthdate *time.Time
	var lat, lon *float64
	var photoCount int
	err := r.db.QueryRow(ctx, `
		SELECT p.user_id, p.gender, p.birthdate, p.latitude, p.longitude,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km,
		       (SELECT COUNT(*) FROM photos ph WHERE ph.user_id = p.user_id)
		FROM profiles p
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id = $1
	`, userID).Scan(&v.UserID, &gender, &birthdate, &lat, &lon, &v.InterestedIn, &v.MinAge, &v.MaxAge, &v.MaxDistanceKm, &photoCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Viewer{}, false, ErrViewerNotFound
	}
	if err != nil {
		return Viewer{}, false, err
	}
	if gender == nil || birthdate == nil || lat == nil || lon == nil || len(v.InterestedIn) == 0 || photoCount == 0 {
		return v, false, nil
	}
	v.Gender, v.Birthdate, v.Latitude, v.Longitude = *gender, *birthdate, *lat, *lon
	return v, true, nil
}

func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}

// candidateSQL selects user ids eligible for the viewer. Eligibility is
// mutual: each side must match the other's gender and age preferences and be
// within both distance limits. Already swiped, blocked (either direction),
// hidden, suspended and incomplete profiles are excluded.
//
// Ranking is a transparent, tunable score: shared interests, proximity and
// recent activity. Replace the ORDER BY with a learned ranker later without
// touching the eligibility rules.
const candidateSQL = `
	SELECT c.user_id
	FROM profiles c
	JOIN preferences cp ON cp.user_id = c.user_id
	JOIN users u ON u.id = c.user_id
	CROSS JOIN LATERAL (
		SELECT km_between($2, $3, c.latitude, c.longitude) AS km
	) d
	WHERE c.user_id <> $1
	  AND ($11::uuid IS NULL OR c.user_id = $11::uuid)
	  AND c.is_discoverable
	  AND u.suspended_at IS NULL
	  AND c.latitude IS NOT NULL
	  AND c.latitude BETWEEN $2 - $10 AND $2 + $10
	  AND c.first_name IS NOT NULL
	  AND c.gender = ANY($4::text[])
	  AND $5 = ANY(cp.interested_in)
	  AND c.birthdate BETWEEN $6 AND $7
	  AND $8 BETWEEN cp.min_age AND cp.max_age
	  AND d.km <= LEAST($9, cp.max_distance_km)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
	  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = c.user_id)
	  AND NOT EXISTS (
		SELECT 1 FROM blocks b
		WHERE (b.blocker_id = $1 AND b.blocked_id = c.user_id)
		   OR (b.blocker_id = c.user_id AND b.blocked_id = $1)
	  )
	ORDER BY (
		(SELECT COUNT(*) FROM user_interests a
		 JOIN user_interests b ON b.interest_id = a.interest_id AND b.user_id = c.user_id
		 WHERE a.user_id = $1) * 1.0
		+ (1 - d.km / GREATEST($9, 1)) * 2.0
		+ CASE WHEN c.last_active_at > NOW() - INTERVAL '1 day' THEN 2.0
		       WHEN c.last_active_at > NOW() - INTERVAL '7 days' THEN 1.0
		       ELSE 0 END
	) DESC, c.last_active_at DESC, c.user_id
	LIMIT $12
`

func candidateArgs(q DiscoveryQuery) []any {
	var only any
	if q.OnlyUserID != "" {
		only = q.OnlyUserID
	}
	return []any{
		q.UserID, q.Latitude, q.Longitude, q.InterestedIn, q.Gender,
		q.OldestBirthdate, q.YoungestBirth, q.ViewerAge, q.MaxDistanceKm, q.LatDelta,
		only, q.Limit,
	}
}

func (r *PGRepository) FindCandidates(ctx context.Context, q DiscoveryQuery) ([]string, error) {
	rows, err := r.db.Query(ctx, candidateSQL, candidateArgs(q)...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func pairKey(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Swipe records a decision and creates a match on mutual likes. A per-pair
// advisory lock makes simultaneous reciprocal likes produce exactly one match.
func (r *PGRepository) Swipe(ctx context.Context, q DiscoveryQuery, targetID, action string) (*StoredMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := pairKey(q.UserID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, low+":"+high); err != nil {
		return nil, err
	}

	// The target must currently be eligible for this viewer: clients cannot
	// like hidden, blocked, out-of-preference or already swiped profiles.
	q.OnlyUserID = targetID
	q.Limit = 1
	var eligible string
	err = tx.QueryRow(ctx, candidateSQL, candidateArgs(q)...).Scan(&eligible)
	if errors.Is(err, pgx.ErrNoRows) {
		var swiped bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2)`, q.UserID, targetID).Scan(&swiped); err != nil {
			return nil, err
		}
		if swiped {
			return nil, ErrAlreadySwiped
		}
		return nil, ErrTargetNotAvailable
	}
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, q.UserID, targetID, action)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadySwiped
	}

	if action != ActionLike {
		return nil, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')
	`, targetID, q.UserID).Scan(&reciprocal); err != nil {
		return nil, err
	}
	if !reciprocal {
		return nil, tx.Commit(ctx)
	}

	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, created_by_user_id)
		VALUES ('match', $1, $2)
		ON CONFLICT (direct_key) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, low+":"+high, q.UserID).Scan(&conversationID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, low, high); err != nil {
		return nil, err
	}
	match := &StoredMatch{ConversationID: conversationID, OtherUserID: targetID}
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a_id, user_b_id, conversation_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET conversation_id = matches.conversation_id
		RETURNING id, conversation_id, created_at
	`, low, high, conversationID).Scan(&match.ID, &match.ConversationID, &match.CreatedAt)
	if err != nil {
		return nil, err
	}
	return match, tx.Commit(ctx)
}

// ListMatches returns the user's matches with last message and unread count,
// most recent activity first. Single query with lateral joins (no N+1).
func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.conversation_id,
		       CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END,
		       m.created_at,
		       lm.content, lm.sender_user_id, lm.created_at,
		       COALESCE(unread.n, 0)
		FROM matches m
		JOIN conversation_participants me
		  ON me.conversation_id = m.conversation_id AND me.user_id = $1
		LEFT JOIN LATERAL (
			SELECT content, sender_user_id, created_at
			FROM messages
			WHERE conversation_id = m.conversation_id
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS n
			FROM messages
			WHERE conversation_id = m.conversation_id
			  AND sender_user_id <> $1
			  AND created_at > COALESCE(me.last_read_at, 'epoch'::timestamptz)
		) unread ON TRUE
		WHERE m.user_a_id = $1 OR m.user_b_id = $1
		ORDER BY COALESCE(lm.created_at, m.created_at) DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredMatch, error) {
		var m StoredMatch
		var content, sender *string
		var sentAt *time.Time
		err := row.Scan(&m.ID, &m.ConversationID, &m.OtherUserID, &m.CreatedAt, &content, &sender, &sentAt, &m.UnreadCount)
		if err == nil && content != nil && sender != nil && sentAt != nil {
			m.LastMessage = &LastMessage{Content: *content, SenderUserID: *sender, CreatedAt: *sentAt}
		}
		return m, err
	})
}

// Unmatch deletes the shared conversation; the match and every message
// cascade. Swipes are kept so the two people are not shown to each other again.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		WITH target AS (
			SELECT conversation_id,
			       CASE WHEN user_a_id = $2 THEN user_b_id ELSE user_a_id END AS other
			FROM matches
			WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
		), deleted AS (
			DELETE FROM conversations WHERE id IN (SELECT conversation_id FROM target)
		)
		SELECT other FROM target
	`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	return other, err
}
