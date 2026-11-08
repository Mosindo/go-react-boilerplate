package matches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryTargetUnavailable = errors.New("repository: target unavailable")
	ErrRepositoryConflict          = errors.New("repository: different swipe already recorded")
	ErrRepositoryNotFound          = errors.New("repository: not found")
)

type swipeOutcome struct {
	Match *storedMatch // non-nil if the two users are matched after this swipe
	// Created is true only when this call created the match (so notifications are sent once).
	Created bool
}

type Repository interface {
	Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error)
	ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]storedMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Swipe records a like/pass atomically and creates the match (and its conversation) when the
// interest is mutual. A transaction-level advisory lock on the user pair serialises two people
// liking each other at the same instant, which would otherwise both miss the other's like.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))`, swiperID, targetID); err != nil {
		return swipeOutcome{}, err
	}

	var available bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles t
		  WHERE t.user_id = $2
		    AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		    AND (t.discoverable
		         OR EXISTS (SELECT 1 FROM swipes s WHERE s.swiper_id = $2 AND s.target_id = $1 AND s.action = 'like')
		         OR EXISTS (SELECT 1 FROM matches m WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid))))`,
		swiperID, targetID).Scan(&available)
	if err != nil {
		return swipeOutcome{}, err
	}
	if !available {
		return swipeOutcome{}, ErrRepositoryTargetUnavailable
	}

	var existing string
	err = tx.QueryRow(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
		RETURNING action`, swiperID, targetID, action).Scan(&existing)
	inserted := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		// Repeated swipe: idempotent when identical, rejected when it contradicts the first one.
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, swiperID, targetID).Scan(&existing); err != nil {
			return swipeOutcome{}, err
		}
		if existing != action {
			return swipeOutcome{}, ErrRepositoryConflict
		}
	} else if err != nil {
		return swipeOutcome{}, err
	}

	out := swipeOutcome{}
	if action == "like" {
		var mutual bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')`, swiperID, targetID).Scan(&mutual); err != nil {
			return swipeOutcome{}, err
		}
		if mutual {
			m := storedMatch{OtherUserID: targetID}
			var matchID string
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING RETURNING id`, swiperID, targetID).Scan(&matchID)
			switch {
			case err == nil:
				out.Created = inserted
				if _, err := tx.Exec(ctx, `INSERT INTO conversations (match_id) VALUES ($1)`, matchID); err != nil {
					return swipeOutcome{}, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO conversation_participants (conversation_id, user_id)
					SELECT c.id, u FROM conversations c, unnest(ARRAY[$2::uuid, $3::uuid]) AS u WHERE c.match_id = $1`, matchID, swiperID, targetID); err != nil {
					return swipeOutcome{}, err
				}
			case errors.Is(err, pgx.ErrNoRows):
				// match already existed
			default:
				return swipeOutcome{}, err
			}
			err = tx.QueryRow(ctx, `
				SELECT m.id, c.id, m.created_at FROM matches m JOIN conversations c ON c.match_id = m.id
				WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)`, swiperID, targetID).
				Scan(&m.ID, &m.ConversationID, &m.CreatedAt)
			if err != nil {
				return swipeOutcome{}, err
			}
			out.Match = &m
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return swipeOutcome{}, err
	}
	return out, nil
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, before *time.Time, limit int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS other_id, m.created_at,
		       EXISTS (SELECT 1 FROM messages x WHERE x.conversation_id = c.id) AS has_messages
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($2::timestamptz IS NULL OR m.created_at < $2)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END)
		           OR (b.blocked_id = $1 AND b.blocker_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END))
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedMatch
	for rows.Next() {
		var m storedMatch
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherUserID, &m.CreatedAt, &m.HasMessages); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $1 AND (user_a = $2 OR user_b = $2)
		RETURNING CASE WHEN user_a = $2 THEN user_b ELSE user_a END`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return other, err
}
