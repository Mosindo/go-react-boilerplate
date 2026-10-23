package matching

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/api/internal/features/photos"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryTargetUnavailable = errors.New("swipe target unavailable")
	ErrRepositoryNotFound          = errors.New("match not found")
)

type Repository interface {
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error)
	Summary(ctx context.Context, viewerID, matchID string) (MatchSummary, error)
	List(ctx context.Context, viewerID string, limit int, beforeTime *time.Time, beforeID string) ([]MatchSummary, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherID string, err error)
	Hide(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Swipe records a decision atomically. A per-pair advisory lock makes the
// "both liked" check race-free: if A and B like each other at the same moment,
// exactly one of the two transactions observes the other's like and creates the match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	a, b := orderedPair(swiperID, targetID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, a+"|"+b); err != nil {
		return SwipeOutcome{}, err
	}

	var available bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM users u JOIN profiles p ON p.user_id = u.id
		  WHERE u.id = $2 AND u.status = 'active'
		    AND NOT EXISTS (
		      SELECT 1 FROM blocks bl
		      WHERE (bl.blocker_id = $1 AND bl.blocked_id = $2) OR (bl.blocker_id = $2 AND bl.blocked_id = $1)
		    )
		)
	`, swiperID, targetID).Scan(&available); err != nil {
		return SwipeOutcome{}, err
	}
	if !available {
		return SwipeOutcome{}, ErrRepositoryTargetUnavailable
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return SwipeOutcome{}, err
	}

	if tag.RowsAffected() == 0 {
		// Decisions are final: report the stored one without side effects.
		out := SwipeOutcome{AlreadyProcessed: true}
		if err := tx.QueryRow(ctx, `SELECT action FROM swipes WHERE swiper_id = $1 AND target_id = $2`, swiperID, targetID).Scan(&out.Action); err != nil {
			return SwipeOutcome{}, err
		}
		out.MatchID, err = activeMatchID(ctx, tx, a, b)
		if err != nil {
			return SwipeOutcome{}, err
		}
		return out, tx.Commit(ctx)
	}

	out := SwipeOutcome{Action: action}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $1 AND target_id = $2 AND action = 'like')
		`, targetID, swiperID).Scan(&reciprocal); err != nil {
			return SwipeOutcome{}, err
		}
		if reciprocal {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id
			`, a, b).Scan(&out.MatchID)
			switch {
			case err == nil:
				out.NewMatch = true
			case errors.Is(err, pgx.ErrNoRows):
				if out.MatchID, err = activeMatchID(ctx, tx, a, b); err != nil {
					return SwipeOutcome{}, err
				}
			default:
				return SwipeOutcome{}, err
			}
		}
	}
	return out, tx.Commit(ctx)
}

func activeMatchID(ctx context.Context, tx pgx.Tx, a, b string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM matches WHERE user_a = $1 AND user_b = $2 AND unmatched_at IS NULL`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func orderedPair(x, y string) (string, string) {
	if x < y {
		return x, y
	}
	return y, x
}

// summarySelect renders a match from one participant's point of view ($1 = viewer).
const summarySelect = `
	SELECT m.id, m.created_at, m.last_activity_at,
	       o.user_id, o.first_name,
	       ph.id, ph.updated_at,
	       (SELECT COUNT(*) FROM messages x
	          WHERE x.match_id = m.id AND x.sender_id <> $1 AND x.read_at IS NULL
	            AND x.created_at > COALESCE(v.hidden_at, '-infinity')),
	       lm.content, lm.sender_id, lm.created_at
	FROM matches m
	CROSS JOIN LATERAL (
	  SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS other_id,
	         CASE WHEN m.user_a = $1 THEN m.a_hidden_at ELSE m.b_hidden_at END AS hidden_at
	) v
	JOIN profiles o ON o.user_id = v.other_id
	JOIN users ou ON ou.id = v.other_id AND ou.status = 'active'
	LEFT JOIN LATERAL (
	  SELECT p.id, p.updated_at FROM photos p WHERE p.user_id = v.other_id ORDER BY p.position LIMIT 1
	) ph ON TRUE
	LEFT JOIN LATERAL (
	  SELECT content, sender_id, created_at FROM messages x
	  WHERE x.match_id = m.id AND x.created_at > COALESCE(v.hidden_at, '-infinity')
	  ORDER BY x.created_at DESC, x.id DESC LIMIT 1
	) lm ON TRUE
	WHERE (m.user_a = $1 OR m.user_b = $1) AND m.unmatched_at IS NULL`

func scanSummary(row pgx.Row) (MatchSummary, error) {
	var (
		s          MatchSummary
		photoID    *string
		photoStamp *time.Time
		lmContent  *string
		lmSender   *string
		lmAt       *time.Time
	)
	if err := row.Scan(&s.ID, &s.CreatedAt, &s.LastActivityAt, &s.User.UserID, &s.User.FirstName,
		&photoID, &photoStamp, &s.UnreadCount, &lmContent, &lmSender, &lmAt); err != nil {
		return MatchSummary{}, err
	}
	if photoID != nil && photoStamp != nil {
		thumb := photos.NewPhoto(*photoID, 0, 1, 1, *photoStamp).ThumbURL
		s.User.ThumbURL = &thumb
	}
	if lmContent != nil && lmSender != nil && lmAt != nil {
		s.LastMessage = &LastMessage{Content: *lmContent, SenderID: *lmSender, CreatedAt: *lmAt}
	}
	return s, nil
}

func (r *PGRepository) Summary(ctx context.Context, viewerID, matchID string) (MatchSummary, error) {
	s, err := scanSummary(r.db.QueryRow(ctx, summarySelect+` AND m.id = $2`, viewerID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return MatchSummary{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) List(ctx context.Context, viewerID string, limit int, beforeTime *time.Time, beforeID string) ([]MatchSummary, error) {
	query := summarySelect + `
	  AND (v.hidden_at IS NULL OR m.last_activity_at > v.hidden_at)`
	args := []any{viewerID}
	if beforeTime != nil {
		query += ` AND (m.last_activity_at, m.id) < ($2, $3::uuid)`
		args = append(args, *beforeTime, beforeID)
	}
	query += fmt.Sprintf(` ORDER BY m.last_activity_at DESC, m.id DESC LIMIT %d`, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MatchSummary{}
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW()
		WHERE id = $1 AND (user_a = $2 OR user_b = $2) AND unmatched_at IS NULL
		RETURNING CASE WHEN user_a = $2 THEN user_b ELSE user_a END
	`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return other, err
}

func (r *PGRepository) Hide(ctx context.Context, userID, matchID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE matches SET
		  a_hidden_at = CASE WHEN user_a = $2 THEN NOW() ELSE a_hidden_at END,
		  b_hidden_at = CASE WHEN user_b = $2 THEN NOW() ELSE b_hidden_at END
		WHERE id = $1 AND (user_a = $2 OR user_b = $2) AND unmatched_at IS NULL
	`, matchID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}
