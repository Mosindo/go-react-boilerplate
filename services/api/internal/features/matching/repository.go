package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ViewerContext(ctx context.Context, userID string) (viewerContext, error)
	Discover(ctx context.Context, q discoverQuery) ([]string, error)
	Swipe(ctx context.Context, actorID, organizationID, targetID, decision string) (swipeResult, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) ViewerContext(ctx context.Context, userID string) (viewerContext, error) {
	var v viewerContext
	err := r.db.QueryRow(ctx, `
		SELECT u.organization_id, p.gender, p.birthdate, p.latitude, p.longitude,
		       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km
		FROM users u
		JOIN profiles p ON p.user_id = u.id
		JOIN preferences pr ON pr.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&v.OrganizationID, &v.Gender, &v.Birthdate, &v.Latitude, &v.Longitude,
		&v.InterestedIn, &v.MinAge, &v.MaxAge, &v.MaxDistanceKm)
	if errors.Is(err, pgx.ErrNoRows) {
		return viewerContext{}, ErrProfileIncomplete
	}
	return v, err
}

// Discover returns candidate ids ranked for the viewer. Hard filters are
// mutual (both sides' gender, age and distance preferences must accept the
// other); the score favours people who already liked the viewer, shared
// interests and recent activity. Swapping this ranking for a smarter model
// only requires changing the ORDER BY score (or replacing this method).
func (r *PGRepository) Discover(ctx context.Context, q discoverQuery) ([]string, error) {
	distance := profiles.HaversineSQL("$9::float8", "$10::float8", "p.latitude", "p.longitude")
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id
		FROM profiles p
		JOIN users u ON u.id = p.user_id AND u.organization_id = $2
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE p.user_id <> $1
		  AND p.discoverable
		  AND p.gender = ANY($5::text[])
		  AND $3 = ANY(pr.interested_in)
		  AND p.birthdate BETWEEN $6 AND $7
		  AND $4 BETWEEN pr.min_age AND pr.max_age
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.actor_user_id = $1 AND s.target_user_id = p.user_id)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = p.user_id)
			   OR (b.blocker_user_id = p.user_id AND b.blocked_user_id = $1)
		  )
		  AND NOT (p.user_id = ANY($14::uuid[]))
		  AND (
			$9::float8 IS NULL
			OR (
				p.latitude IS NOT NULL
				AND p.latitude BETWEEN $9::float8 - $11::float8 AND $9::float8 + $11::float8
				AND ($12::float8 IS NULL OR p.longitude BETWEEN $10::float8 - $12::float8 AND $10::float8 + $12::float8)
				AND `+distance+` <= LEAST($13::int, pr.max_distance_km)
			)
		  )
		ORDER BY (
			CASE WHEN EXISTS (
				SELECT 1 FROM swipes l
				WHERE l.actor_user_id = p.user_id AND l.target_user_id = $1 AND l.decision = 'like'
			) THEN 3 ELSE 0 END
			+ LEAST(5, (
				SELECT COUNT(*) FROM user_interests mine
				JOIN user_interests theirs ON theirs.interest_id = mine.interest_id AND theirs.user_id = p.user_id
				WHERE mine.user_id = $1
			))
			+ CASE
				WHEN p.last_active_at > NOW() - INTERVAL '3 days' THEN 2
				WHEN p.last_active_at > NOW() - INTERVAL '14 days' THEN 1
				ELSE 0
			  END
		) DESC, p.last_active_at DESC, p.user_id
		LIMIT $8
	`, q.ViewerID, q.OrganizationID, q.Gender, q.Age, q.InterestedIn, q.Earliest, q.Latest, q.Limit,
		q.Latitude, q.Longitude, q.DeltaLat, q.DeltaLng, q.MaxDistanceKm, q.Exclude)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func orderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func (r *PGRepository) Swipe(ctx context.Context, actorID, organizationID, targetID, decision string) (swipeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	low, high := orderedPair(actorID, targetID)
	// Serialize the two members of a pair so simultaneous mutual likes always produce exactly one match.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, low+":"+high); err != nil {
		return swipeResult{}, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.organization_id = $2
			WHERE p.user_id = $3
			  AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.blocker_user_id = $1 AND b.blocked_user_id = $3)
				   OR (b.blocker_user_id = $3 AND b.blocked_user_id = $1)
			  )
		)
	`, actorID, organizationID, targetID).Scan(&available); err != nil {
		return swipeResult{}, err
	}
	if !available {
		return swipeResult{}, ErrTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (actor_user_id, target_user_id, decision)
		VALUES ($1, $2, $3)
		ON CONFLICT (actor_user_id, target_user_id) DO NOTHING
	`, actorID, targetID, decision)
	if err != nil {
		return swipeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return swipeResult{}, ErrAlreadySwiped
	}

	result := swipeResult{}
	if decision == DecisionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE actor_user_id = $1 AND target_user_id = $2 AND decision = 'like')
		`, targetID, actorID).Scan(&reciprocal); err != nil {
			return swipeResult{}, err
		}
		if reciprocal {
			if err := createMatch(ctx, tx, low, high, actorID, &result); err != nil {
				return swipeResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return swipeResult{}, err
	}
	return result, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, low, high, createdBy string, result *swipeResult) error {
	var conversationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO conversations (kind, direct_key, created_by_user_id)
		VALUES ('direct', $1, $2)
		ON CONFLICT (direct_key) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, low+":"+high, createdBy).Scan(&conversationID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT (conversation_id, user_id) DO UPDATE SET hidden_at = NULL
	`, conversationID, low, high); err != nil {
		return err
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_low_id, user_high_id, conversation_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_low_id, user_high_id) DO UPDATE SET conversation_id = EXCLUDED.conversation_id
		RETURNING id, created_at
	`, low, high, conversationID).Scan(&result.MatchID, &result.CreatedAt)
	if err != nil {
		return err
	}
	result.Matched = true
	result.ConversationID = conversationID
	return nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id,
		       CASE WHEN m.user_low_id = $1 THEN m.user_high_id ELSE m.user_low_id END AS other_id,
		       m.created_at,
		       (SELECT MAX(msg.created_at) FROM messages msg WHERE msg.conversation_id = m.conversation_id) AS last_message_at
		FROM matches m
		WHERE m.user_low_id = $1 OR m.user_high_id = $1
		ORDER BY m.created_at DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedMatch, error) {
		var m storedMatch
		err := row.Scan(&m.ID, &m.OtherUserID, &m.CreatedAt, &m.LastMessageAt)
		return m, err
	})
}

// Unmatch removes the match and its conversation for both members, and turns
// the caller's like into a pass so the pair never resurfaces in discovery.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var otherID string
	var conversationID *string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $1 AND (user_low_id = $2 OR user_high_id = $2)
		RETURNING CASE WHEN user_low_id = $2 THEN user_high_id ELSE user_low_id END, conversation_id
	`, matchID, userID).Scan(&otherID, &conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrMatchNotFound
	}
	if err != nil {
		return "", err
	}
	if err := removePairState(ctx, tx, userID, otherID, conversationID); err != nil {
		return "", err
	}
	return otherID, tx.Commit(ctx)
}

func removePairState(ctx context.Context, tx pgx.Tx, userID, otherID string, conversationID *string) error {
	if conversationID != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, *conversationID); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
		UPDATE swipes SET decision = 'pass', updated_at = $3
		WHERE actor_user_id = $1 AND target_user_id = $2
	`, userID, otherID, time.Now())
	return err
}
