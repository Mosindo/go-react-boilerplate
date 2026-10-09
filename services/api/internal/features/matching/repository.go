package matching

import (
	"context"
	"errors"
	"fmt"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("matching: not found")
	ErrProfileIncomplet = errors.New("matching: profile incomplete")
	ErrSelfSwipe        = errors.New("matching: cannot swipe yourself")
)

// Ranker decides the order in which candidates are proposed. Swap it (or
// precompute a candidate queue) to evolve the recommendation algorithm
// without touching the filtering rules in the query.
type Ranker interface {
	// OrderBy returns a SQL ORDER BY expression list. Available aliases:
	// p (candidate profile), u (candidate user), v (viewer profile) and
	// lm.liked_me (whether the candidate already liked the viewer).
	OrderBy() string
}

// ReciprocalFirstRanker proposes people who already liked the viewer first
// (an instant match is the best outcome), then recently active members.
type ReciprocalFirstRanker struct{}

func (ReciprocalFirstRanker) OrderBy() string {
	return "lm.liked_me DESC, u.last_active_at DESC, p.user_id"
}

type Repository interface {
	Discover(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error)
	Swipe(ctx context.Context, userID, targetID string, action Action) (SwipeResult, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]*profiles.PublicProfile, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchItem, error)
	Unmatch(ctx context.Context, userID, matchID string) (RemovedMatch, error)
}

type PGRepository struct {
	db     *pgxpool.Pool
	ranker Ranker
}

func NewPGRepository(db *pgxpool.Pool, ranker Ranker) *PGRepository {
	if ranker == nil {
		ranker = ReciprocalFirstRanker{}
	}
	return &PGRepository{db: db, ranker: ranker}
}

const notBlocked = `NOT EXISTS (
	SELECT 1 FROM blocks b
	WHERE (b.blocker_id = v.user_id AND b.blocked_id = p.user_id)
	   OR (b.blocker_id = p.user_id AND b.blocked_id = v.user_id))`

// Discover returns compatible, not-yet-swiped candidates. Compatibility is
// mutual: gender, age range and distance must satisfy *both* members' preferences.
func (r *PGRepository) Discover(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error) {
	if err := r.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	query := `
		SELECT ` + profiles.PublicSelect + `
		FROM profiles v
		JOIN preferences vp ON vp.user_id = v.user_id
		JOIN profiles p ON p.user_id <> v.user_id
		JOIN preferences tp ON tp.user_id = p.user_id
		JOIN users u ON u.id = p.user_id
		CROSS JOIN LATERAL (
			SELECT EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = p.user_id AND s.target_id = v.user_id AND s.action = 'like') AS liked_me
		) lm
		WHERE v.user_id = $1
		  AND p.discoverable
		  -- latitude window first: it can use idx_profiles_discovery before the exact haversine test
		  AND p.latitude BETWEEN v.latitude - vp.max_distance_km / 111.0 AND v.latitude + vp.max_distance_km / 111.0
		  AND p.gender = ANY (vp.interested_in)
		  AND v.gender = ANY (tp.interested_in)
		  AND date_part('year', age(p.birth_date)) BETWEEN vp.min_age AND vp.max_age
		  AND date_part('year', age(v.birth_date)) BETWEEN tp.min_age AND tp.max_age
		  AND geo_distance_km(v.latitude, v.longitude, p.latitude, p.longitude) <= LEAST(vp.max_distance_km, tp.max_distance_km)
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = v.user_id AND s.target_id = p.user_id)
		  AND ` + notBlocked + `
		ORDER BY ` + r.ranker.OrderBy() + `
		LIMIT $2`
	return r.queryPublic(ctx, query, userID, limit)
}

func (r *PGRepository) queryPublic(ctx context.Context, query string, args ...any) ([]*profiles.PublicProfile, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	out := []*profiles.PublicProfile{}
	for rows.Next() {
		p := &profiles.PublicProfile{}
		if err := profiles.ScanPublic(rows, p); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := profiles.Hydrate(ctx, r.db, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PGRepository) requireComplete(ctx context.Context, userID string) error {
	var complete bool
	err := r.db.QueryRow(ctx, completeSQL, userID).Scan(&complete)
	if err != nil {
		return err
	}
	if !complete {
		return ErrProfileIncomplet
	}
	return nil
}

const completeSQL = `
	SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)
	   AND EXISTS (SELECT 1 FROM preferences WHERE user_id = $1)
	   AND EXISTS (SELECT 1 FROM photos WHERE user_id = $1)`

func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]*profiles.PublicProfile, error) {
	if err := r.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	return r.queryPublic(ctx, `
		SELECT `+profiles.PublicSelect+`
		FROM swipes s
		JOIN profiles p ON p.user_id = s.user_id
		JOIN profiles v ON v.user_id = $1
		WHERE s.target_id = $1 AND s.action = 'like'
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.user_id = $1 AND mine.target_id = p.user_id)
		  AND `+notBlocked+`
		ORDER BY s.created_at DESC, p.user_id
		LIMIT $2 OFFSET $3`, userID, limit, offset)
}

// Swipe records the decision and creates the match when interest is mutual.
// An advisory lock on the unordered pair serialises two people liking each
// other at the same instant, so exactly one of them creates the match.
func (r *PGRepository) Swipe(ctx context.Context, userID, targetID string, action Action) (SwipeResult, error) {
	if userID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, userID, targetID); err != nil {
		if isInvalidUUID(err) {
			return SwipeResult{}, ErrNotFound
		}
		return SwipeResult{}, err
	}

	var complete bool
	if err := tx.QueryRow(ctx, completeSQL, userID).Scan(&complete); err != nil {
		return SwipeResult{}, err
	}
	if !complete {
		return SwipeResult{}, ErrProfileIncomplet
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $2)
		   AND NOT EXISTS (SELECT 1 FROM blocks b
		       WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
	`, userID, targetID).Scan(&available); err != nil {
		return SwipeResult{}, err
	}
	if !available {
		return SwipeResult{}, ErrNotFound
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (user_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, target_id) DO NOTHING
	`, userID, targetID, string(action))
	if err != nil {
		return SwipeResult{}, err
	}

	res := SwipeResult{}
	if tag.RowsAffected() == 0 {
		// Repeated swipe: never duplicate, report the existing state.
		res.AlreadySwiped = true
		ref, err := activeMatch(ctx, tx, userID, targetID)
		if err != nil {
			return SwipeResult{}, err
		}
		if ref != nil {
			res.Matched, res.Match = true, ref
		}
		if err := tx.Commit(ctx); err != nil {
			return SwipeResult{}, err
		}
		return res, nil
	}

	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE user_id = $2 AND target_id = $1 AND action = 'like')
		`, userID, targetID).Scan(&reciprocal); err != nil {
			return SwipeResult{}, err
		}
		if reciprocal {
			created, ref, err := createMatch(ctx, tx, userID, targetID)
			if err != nil {
				return SwipeResult{}, err
			}
			res.Matched, res.Match, res.Created = true, ref, created
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SwipeResult{}, err
	}
	return res, nil
}

func activeMatch(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, a, b string) (*MatchRef, error) {
	var ref MatchRef
	err := q.QueryRow(ctx, `
		SELECT m.id::text, c.id::text FROM matches m JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid) AND m.unmatched_at IS NULL
	`, a, b).Scan(&ref.ID, &ref.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, userID, targetID string) ([]MatchedUser, *MatchRef, error) {
	var ref MatchRef
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id::text
	`, userID, targetID).Scan(&ref.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		// The pair matched before (e.g. unmatched later): never revive it.
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id::text`, ref.ID).Scan(&ref.ConversationID); err != nil {
		return nil, nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)
	`, ref.ConversationID, userID, targetID); err != nil {
		return nil, nil, err
	}

	var created []MatchedUser
	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		var other profiles.ProfileSummary
		if err := profiles.ScanSummary(tx.QueryRow(ctx, `SELECT `+profiles.SummarySelect+` FROM profiles p WHERE p.user_id = $1`, pair[1]), &other); err != nil {
			return nil, nil, err
		}
		n, err := notifications.Create(ctx, tx, notifications.NewNotification{
			UserID: pair[0],
			Type:   notifications.TypeMatch,
			Title:  "C'est un match !",
			Body:   fmt.Sprintf("Vous et %s vous plaisez mutuellement. Dites bonjour !", other.FirstName),
			Data:   map[string]string{"matchId": ref.ID, "conversationId": ref.ConversationID, "userId": pair[1]},
		})
		if err != nil {
			return nil, nil, err
		}
		created = append(created, MatchedUser{UserID: pair[0], Other: other, Notification: n})
	}
	return created, &ref, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text, m.created_at, c.id::text, `+profiles.SummarySelect+`
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND m.unmatched_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		      WHERE (b.blocker_id = $1 AND b.blocked_id = p.user_id) OR (b.blocker_id = p.user_id AND b.blocked_id = $1))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MatchItem{}
	for rows.Next() {
		var it MatchItem
		if err := rows.Scan(&it.ID, &it.CreatedAt, &it.ConversationID, &it.User.ID, &it.User.FirstName, &it.User.Age, &it.User.MainPhotoURL); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (RemovedMatch, error) {
	var rm RemovedMatch
	err := r.db.QueryRow(ctx, `
		UPDATE matches m SET unmatched_at = NOW()
		WHERE m.id = $1 AND (m.user_a = $2 OR m.user_b = $2) AND m.unmatched_at IS NULL
		RETURNING m.id::text, m.user_a::text, m.user_b::text, (SELECT c.id::text FROM conversations c WHERE c.match_id = m.id)
	`, matchID, userID).Scan(&rm.MatchID, &rm.UserIDs[0], &rm.UserIDs[1], &rm.ConversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return RemovedMatch{}, ErrNotFound
		}
		return RemovedMatch{}, err
	}
	return rm, nil
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
