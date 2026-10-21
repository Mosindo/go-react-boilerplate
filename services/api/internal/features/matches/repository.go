package matches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTargetUnavailable = errors.New("matches: target unavailable")
	ErrNoProfile         = errors.New("matches: caller has no profile")
	ErrNotFound          = errors.New("matches: not found")
)

// storedMatch is a match row joined with the other member's identity.
type storedMatch struct {
	ID             string
	ConversationID string
	CreatedAt      time.Time
	OtherID        string
	OtherName      string
	OtherPhotoID   *string
}

// swipeOutcome is what the transaction decided.
type swipeOutcome struct {
	Action    string
	Match     *storedMatch // set when the pair is matched (new or pre-existing)
	NewMatch  bool         // true only when this call created the match
	MyName    string       // identity of the caller, for the other member's notification
	MyPhotoID *string
}

type Repository interface {
	Swipe(ctx context.Context, me, target, action string) (swipeOutcome, error)
	ResetPasses(ctx context.Context, me string) (int64, error)
	List(ctx context.Context, me string, before *time.Time, limit int) ([]storedMatch, error)
	Unmatch(ctx context.Context, me, matchID string) (otherID, conversationID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Swipe records the decision and creates the match when interest is mutual, atomically. A
// per-pair advisory lock serialises two members liking each other at the same instant, so exactly
// one of them observes (and creates) the match.
func (r *PGRepository) Swipe(ctx context.Context, me, target, action string) (swipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return swipeOutcome{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))`, me, target); err != nil {
		return swipeOutcome{}, err
	}

	var myName string
	var myPhoto *string
	err = tx.QueryRow(ctx, `
		SELECT p.first_name, (SELECT id FROM photos WHERE user_id = p.user_id AND position = 0)
		FROM profiles p WHERE p.user_id = $1
	`, me).Scan(&myName, &myPhoto)
	if errors.Is(err, pgx.ErrNoRows) {
		return swipeOutcome{}, ErrNoProfile
	}
	if err != nil {
		return swipeOutcome{}, err
	}

	var available bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM profiles p
			JOIN users u ON u.id = p.user_id AND u.status = 'active'
			WHERE p.user_id = $2 AND p.user_id <> $1
			  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = $2) OR (b.blocker_id = $2 AND b.blocked_id = $1))
		)`, me, target).Scan(&available)
	if err != nil {
		return swipeOutcome{}, err
	}
	if !available {
		return swipeOutcome{}, ErrTargetUnavailable
	}

	out := swipeOutcome{Action: action, MyName: myName, MyPhotoID: myPhoto}

	tag, err := tx.Exec(ctx, `INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, me, target, action)
	if err != nil {
		return swipeOutcome{}, err
	}
	if tag.RowsAffected() == 0 {
		// Already decided earlier: idempotent answer, nothing is written.
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, me, target).Scan(&out.Action); err != nil {
			return swipeOutcome{}, err
		}
		out.Match, err = findMatch(ctx, tx, me, target)
		if err != nil {
			return swipeOutcome{}, err
		}
		return out, tx.Commit(ctx)
	}

	if action == "like" {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')`, me, target).Scan(&reciprocal); err != nil {
			return swipeOutcome{}, err
		}
		if reciprocal {
			var matchID, convID string
			var createdAt time.Time
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id, created_at
			`, me, target).Scan(&matchID, &createdAt)
			if err == nil {
				if err := tx.QueryRow(ctx, `INSERT INTO conversations (match_id) VALUES ($1) RETURNING id`, matchID).Scan(&convID); err != nil {
					return swipeOutcome{}, err
				}
				if _, err := tx.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2), ($1, $3)`, convID, me, target); err != nil {
					return swipeOutcome{}, err
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)
					ON CONFLICT DO NOTHING
				`, me, target, matchID); err != nil {
					return swipeOutcome{}, err
				}
				out.NewMatch = true
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return swipeOutcome{}, err
			}
			out.Match, err = findMatch(ctx, tx, me, target)
			if err != nil {
				return swipeOutcome{}, err
			}
		}
	}
	return out, tx.Commit(ctx)
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func findMatch(ctx context.Context, q rowQuerier, me, other string) (*storedMatch, error) {
	var m storedMatch
	err := q.QueryRow(ctx, `
		SELECT m.id, c.id, m.created_at, p.user_id, p.first_name, (SELECT id FROM photos WHERE user_id = p.user_id AND position = 0)
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = $2
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
	`, me, other).Scan(&m.ID, &m.ConversationID, &m.CreatedAt, &m.OtherID, &m.OtherName, &m.OtherPhotoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &m, err
}

func (r *PGRepository) ResetPasses(ctx context.Context, me string) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM swipes WHERE swiper_id = $1 AND action = 'pass'`, me)
	return tag.RowsAffected(), err
}

func (r *PGRepository) List(ctx context.Context, me string, before *time.Time, limit int) ([]storedMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, m.created_at, p.user_id, p.first_name,
		       (SELECT id FROM photos WHERE user_id = p.user_id AND position = 0)
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND ($2::timestamptz IS NULL OR m.created_at < $2)
		ORDER BY m.created_at DESC
		LIMIT $3
	`, me, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedMatch
	for rows.Next() {
		var m storedMatch
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.CreatedAt, &m.OtherID, &m.OtherName, &m.OtherPhotoID); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, me, matchID string) (string, string, error) {
	var other, conv string
	err := r.db.QueryRow(ctx, `
		WITH gone AS (
			DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
			RETURNING id, CASE WHEN user_a = $1 THEN user_b ELSE user_a END AS other
		)
		SELECT gone.other, (SELECT id FROM conversations WHERE match_id = gone.id) FROM gone
	`, me, matchID).Scan(&other, &conv)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return other, conv, err
}
