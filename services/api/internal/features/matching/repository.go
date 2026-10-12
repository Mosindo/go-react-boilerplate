package matching

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("match not found")

type Repository interface {
	Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error)
	ListMatches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error)
	// Unmatch deletes the match and returns the other participant.
	Unmatch(ctx context.Context, matchID, userID string) (otherUserID string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Swipe records a decision atomically. A transaction-scoped advisory lock on the
// unordered pair makes two simultaneous mutual likes serialise, so exactly one
// of them observes the other's like and creates the (single) match.
func (r *PGRepository) Swipe(ctx context.Context, swiperID, targetID, action string) (SwipeOutcome, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return SwipeOutcome{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::uuid, $2::uuid)::text || GREATEST($1::uuid, $2::uuid)::text, 0))
	`, swiperID, targetID); err != nil {
		return SwipeOutcome{}, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (swiper_id, target_id) DO NOTHING
	`, swiperID, targetID, action)
	if err != nil {
		return SwipeOutcome{}, err
	}

	out := SwipeOutcome{AlreadyProcessed: tag.RowsAffected() == 0}
	if out.AlreadyProcessed || action != "like" {
		if out.AlreadyProcessed {
			out.MatchID, err = existingMatch(ctx, tx, swiperID, targetID)
			if err != nil {
				return SwipeOutcome{}, err
			}
		}
		return out, tx.Commit(ctx)
	}

	var reciprocal bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM swipes WHERE swiper_id = $2 AND target_id = $1 AND action = 'like')
	`, swiperID, targetID).Scan(&reciprocal); err != nil {
		return SwipeOutcome{}, err
	}
	if !reciprocal {
		return out, tx.Commit(ctx)
	}

	var matchID string
	err = tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b)
		VALUES (LEAST($1::uuid, $2::uuid), GREATEST($1::uuid, $2::uuid))
		ON CONFLICT (user_a, user_b) DO NOTHING
		RETURNING id
	`, swiperID, targetID).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		matchID, err = existingMatch(ctx, tx, swiperID, targetID)
		if err != nil {
			return SwipeOutcome{}, err
		}
		out.MatchID = matchID
		return out, tx.Commit(ctx)
	}
	if err != nil {
		return SwipeOutcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO match_members (match_id, user_id) VALUES ($1, $2), ($1, $3)
	`, matchID, swiperID, targetID); err != nil {
		return SwipeOutcome{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_notifications (user_id, kind, actor_id, match_id)
		VALUES ($1, 'match', $2, $3), ($2, 'match', $1, $3)
	`, swiperID, targetID, matchID); err != nil {
		return SwipeOutcome{}, err
	}
	out.MatchID = matchID
	return out, tx.Commit(ctx)
}

func existingMatch(ctx context.Context, q pgx.Tx, a, b string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		SELECT id FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
	`, a, b).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit, offset int) ([]StoredMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id,
		       CASE WHEN m.user_a = $1::uuid THEN m.user_b ELSE m.user_a END AS other_id,
		       m.created_at,
		       COALESCE(lm.created_at, m.created_at) AS activity,
		       lm.id, lm.sender_id, lm.body, lm.created_at,
		       (SELECT COUNT(*) FROM chat_messages cm
		         WHERE cm.match_id = m.id AND cm.sender_id <> $1::uuid
		           AND cm.created_at > mm.last_read_at
		           AND (mm.cleared_at IS NULL OR cm.created_at > mm.cleared_at))::int AS unread
		FROM matches m
		JOIN match_members mm ON mm.match_id = m.id AND mm.user_id = $1::uuid
		JOIN users ou ON ou.id = CASE WHEN m.user_a = $1::uuid THEN m.user_b ELSE m.user_a END
		LEFT JOIN LATERAL (
			SELECT cm.id, cm.sender_id, cm.body, cm.created_at
			FROM chat_messages cm
			WHERE cm.match_id = m.id AND (mm.cleared_at IS NULL OR cm.created_at > mm.cleared_at)
			ORDER BY cm.created_at DESC, cm.id DESC LIMIT 1
		) lm ON TRUE
		WHERE (m.user_a = $1::uuid OR m.user_b = $1::uuid) AND ou.disabled_at IS NULL
		ORDER BY activity DESC, m.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredMatch{}
	for rows.Next() {
		var m StoredMatch
		var lmID, lmSender, lmBody *string
		var lmAt *time.Time
		if err := rows.Scan(&m.ID, &m.OtherUserID, &m.CreatedAt, &m.LastActivityAt, &lmID, &lmSender, &lmBody, &lmAt, &m.UnreadCount); err != nil {
			return nil, err
		}
		if lmID != nil {
			m.LastMessage = &MessagePreview{ID: *lmID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, matchID, userID string) (string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches
		WHERE id = $1::uuid AND (user_a = $2::uuid OR user_b = $2::uuid)
		RETURNING CASE WHEN user_a = $2::uuid THEN user_b ELSE user_a END
	`, matchID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return other, err
}
