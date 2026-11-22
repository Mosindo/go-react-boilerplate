package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, int, error)
	Swipe(ctx context.Context, swiperID, targetID, action string) (swipeResult, error)
	LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error)
	Matches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error)
	MatchWith(ctx context.Context, userID, otherID string) (storedMatch, error)
	Unmatch(ctx context.Context, userID, otherID string) error
	ResetPasses(ctx context.Context, userID string) (int, error)
	TouchActivity(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// PairKey is the canonical, order-independent key for two users. It is shared
// with chat (conversations.direct_key).
func PairKey(a, b string) string {
	lo, hi := OrderedPair(a, b)
	return lo + ":" + hi
}

// OrderedPair returns (lower, higher); canonical lowercase UUID strings sort
// the same way as PostgreSQL uuid values.
func OrderedPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// Candidates returns the eligible discovery pool and the viewer's max
// distance. Every rule is mutual: gender preferences, age ranges and distance
// must suit both members. Already-swiped, blocked, paused, incomplete,
// moderated and cross-workspace profiles are excluded in SQL.
func (r *PGRepository) Candidates(ctx context.Context, userID string, poolSize int) ([]Candidate, int, error) {
	rows, err := r.db.Query(ctx, `
		WITH me AS (
			SELECT p.user_id, p.gender, p.birthdate, p.latitude AS lat, p.longitude AS lng,
			       pr.interested_in, pr.min_age, pr.max_age, pr.max_distance_km, u.organization_id,
			       pr.max_distance_km / 111.0 AS dlat,
			       pr.max_distance_km / (111.0 * GREATEST(cos(radians(p.latitude)), 0.01)) AS dlng
			FROM profiles p
			JOIN preferences pr ON pr.user_id = p.user_id
			JOIN users u ON u.id = p.user_id
			WHERE p.user_id = $1 AND p.latitude IS NOT NULL AND p.gender IS NOT NULL AND p.birthdate IS NOT NULL
		)
		SELECT c.user_id,
		       geo_distance_km(me.lat, me.lng, c.latitude, c.longitude) AS distance_km,
		       cu.last_active_at,
		       (SELECT COUNT(*) FROM user_interests mi
		          JOIN user_interests ci ON ci.interest_id = mi.interest_id AND ci.user_id = c.user_id
		         WHERE mi.user_id = me.user_id) AS shared,
		       EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = c.user_id AND l.target_id = me.user_id AND l.action = 'like') AS liked_me,
		       me.max_distance_km
		FROM me
		JOIN profiles c
		  ON c.user_id <> me.user_id
		 AND c.discoverable = TRUE
		 AND c.moderation_status = 'active'
		 AND c.latitude IS NOT NULL
		 AND c.latitude BETWEEN me.lat - me.dlat AND me.lat + me.dlat
		 AND abs(mod((c.longitude - me.lng + 540)::numeric, 360) - 180) <= me.dlng
		JOIN preferences cp ON cp.user_id = c.user_id
		JOIN users cu ON cu.id = c.user_id AND cu.organization_id = me.organization_id
		WHERE c.gender = ANY(me.interested_in)
		  AND me.gender = ANY(cp.interested_in)
		  AND c.first_name IS NOT NULL
		  AND c.birthdate <= CURRENT_DATE - make_interval(years => me.min_age)
		  AND c.birthdate >  CURRENT_DATE - make_interval(years => me.max_age + 1)
		  AND me.birthdate <= CURRENT_DATE - make_interval(years => cp.min_age)
		  AND me.birthdate >  CURRENT_DATE - make_interval(years => cp.max_age + 1)
		  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = c.user_id)
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = me.user_id AND s.target_id = c.user_id)
		  AND NOT `+profiles.SQLBlockedBetween("me.user_id", "c.user_id")+`
		  AND geo_distance_km(me.lat, me.lng, c.latitude, c.longitude) <= LEAST(me.max_distance_km, cp.max_distance_km)
		ORDER BY cu.last_active_at DESC, c.user_id
		LIMIT $2
	`, userID, poolSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Candidate, 0, poolSize)
	maxDistance := 0
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.UserID, &c.DistanceKm, &c.LastActiveAt, &c.SharedInterests, &c.LikedViewer, &maxDistance); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, maxDistance, rows.Err()
}

func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (swipeResult, error) {
	var res swipeResult
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Serialise decisions of the same pair so simultaneous mutual likes
		// always produce exactly one match.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, PairKey(swiperID, targetID)); err != nil {
			return err
		}

		var swiperReady bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM profiles p JOIN preferences pr ON pr.user_id = p.user_id
				WHERE p.user_id = $1 AND `+profiles.SQLProfileComplete("p", "pr")+`
			)
		`, swiperID).Scan(&swiperReady); err != nil {
			return err
		}
		if !swiperReady {
			return ErrProfileIncomplete
		}

		var eligible bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM profiles t
				JOIN preferences tp ON tp.user_id = t.user_id
				JOIN users tu ON tu.id = t.user_id
				JOIN users su ON su.id = $1
				JOIN profiles sp ON sp.user_id = su.id
				JOIN preferences spr ON spr.user_id = su.id
				WHERE t.user_id = $2
				  AND tu.organization_id = su.organization_id
				  AND t.discoverable = TRUE
				  AND `+profiles.SQLProfileComplete("t", "tp")+`
				  AND NOT `+profiles.SQLBlockedBetween("$1::uuid", "$2::uuid")+`
				  AND (
				    (t.gender = ANY(spr.interested_in) AND sp.gender = ANY(tp.interested_in))
				    OR EXISTS (SELECT 1 FROM swipes l WHERE l.swiper_id = $2 AND l.target_id = $1 AND l.action = 'like')
				  )
			)
		`, swiperID, targetID).Scan(&eligible); err != nil {
			return err
		}
		if !eligible {
			return ErrTargetUnavailable
		}

		var inserted string
		err := tx.QueryRow(ctx, `
			INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
			ON CONFLICT (swiper_id, target_id) DO NOTHING
			RETURNING action
		`, swiperID, targetID, action).Scan(&inserted)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			var existing string
			if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, swiperID, targetID).Scan(&existing); err != nil {
				return err
			}
			if existing != action {
				return ErrAlreadySwiped
			}
		case err != nil:
			return err
		default:
			res.NewLike = action == ActionLike
		}
		res.Action = action
		if action != ActionLike {
			return nil
		}

		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, swiperID).Scan(&reciprocal); err != nil {
			return err
		}
		if !reciprocal {
			return nil
		}

		lo, hi := OrderedPair(swiperID, targetID)
		m := storedMatch{OtherUserID: targetID}
		err = tx.QueryRow(ctx, `
			INSERT INTO matches (user_a_id, user_b_id) VALUES ($1, $2)
			ON CONFLICT (user_a_id, user_b_id) DO NOTHING
			RETURNING id, created_at
		`, lo, hi).Scan(&m.ID, &m.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			// Idempotent replay of an existing match.
			if err := tx.QueryRow(ctx, `
				SELECT id, conversation_id, created_at FROM matches WHERE user_a_id = $1 AND user_b_id = $2
			`, lo, hi).Scan(&m.ID, &m.ConversationID, &m.CreatedAt); err != nil {
				return err
			}
			res.Match = &m
			return nil
		}
		if err != nil {
			return err
		}

		var conversationID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO conversations (kind, direct_key, created_by_user_id)
			VALUES ('direct', $1, $2)
			ON CONFLICT (direct_key) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`, PairKey(swiperID, targetID), swiperID).Scan(&conversationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO conversation_participants (conversation_id, user_id)
			VALUES ($1, $2), ($1, $3)
			ON CONFLICT (conversation_id, user_id) DO NOTHING
		`, conversationID, swiperID, targetID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE matches SET conversation_id = $2 WHERE id = $1`, m.ID, conversationID); err != nil {
			return err
		}
		m.ConversationID = &conversationID
		res.Match = &m
		res.NewMatch = true
		return nil
	})
	return res, err
}

// LikesReceived lists members who liked userID and are still waiting for an
// answer. Because the product is free, this list is never hidden.
func (r *PGRepository) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.swiper_id
		FROM swipes l
		JOIN profiles p ON p.user_id = l.swiper_id
		JOIN preferences pr ON pr.user_id = p.user_id
		WHERE l.target_id = $1
		  AND l.action = 'like'
		  AND p.discoverable = TRUE
		  AND `+profiles.SQLProfileComplete("p", "pr")+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $1 AND s.target_id = l.swiper_id)
		  AND NOT `+profiles.SQLBlockedBetween("$1::uuid", "l.swiper_id")+`
		ORDER BY l.created_at DESC, l.swiper_id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *PGRepository) Matches(ctx context.Context, userID string, limit, offset int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, CASE WHEN m.user_a_id = $1 THEN m.user_b_id ELSE m.user_a_id END, m.conversation_id, m.created_at
		FROM matches m
		WHERE (m.user_a_id = $1 OR m.user_b_id = $1)
		ORDER BY m.created_at DESC, m.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedMatch, error) {
		var m storedMatch
		err := row.Scan(&m.ID, &m.OtherUserID, &m.ConversationID, &m.CreatedAt)
		return m, err
	})
}

func (r *PGRepository) MatchWith(ctx context.Context, userID, otherID string) (storedMatch, error) {
	lo, hi := OrderedPair(userID, otherID)
	m := storedMatch{OtherUserID: otherID}
	err := r.db.QueryRow(ctx, `
		SELECT id, conversation_id, created_at FROM matches WHERE user_a_id = $1 AND user_b_id = $2
	`, lo, hi).Scan(&m.ID, &m.ConversationID, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedMatch{}, ErrMatchNotFound
	}
	return m, err
}

// Unmatch deletes the match and its conversation (messages included) for
// both members. Swipes are kept so the profiles never resurface.
func (r *PGRepository) Unmatch(ctx context.Context, userID, otherID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		return DeleteMatchTx(ctx, tx, userID, otherID, true)
	})
}

// DeleteMatchTx is shared with the safety feature (blocking).
func DeleteMatchTx(ctx context.Context, tx pgx.Tx, userID, otherID string, mustExist bool) error {
	lo, hi := OrderedPair(userID, otherID)
	tag, err := tx.Exec(ctx, `DELETE FROM matches WHERE user_a_id = $1 AND user_b_id = $2`, lo, hi)
	if err != nil {
		return err
	}
	if mustExist && tag.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	_, err = tx.Exec(ctx, `DELETE FROM conversations WHERE direct_key = $1`, PairKey(userID, otherID))
	return err
}

func (r *PGRepository) ResetPasses(ctx context.Context, userID string) (int, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM swipes WHERE swiper_id = $1 AND action = 'pass'`, userID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// TouchActivity refreshes last_active_at at most every 5 minutes.
func (r *PGRepository) TouchActivity(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_active_at = NOW()
		WHERE id = $1 AND last_active_at < NOW() - INTERVAL '5 minutes'
	`, userID)
	return err
}
