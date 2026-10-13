package matches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAlreadyActed = errors.New("you already liked or passed this profile")
	ErrNotEligible  = errors.New("profile is not available")
	ErrNotFound     = errors.New("match not found")
)

// Repository persists swipes and matches. Match creation is atomic with the
// swipe: pair-level advisory locking guarantees that two simultaneous likes
// produce exactly one match.
type Repository interface {
	HasActed(ctx context.Context, fromID, toID string) (bool, error)
	Swipe(ctx context.Context, fromID, toID, action string) (LikeResult, error)
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredMatch, error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) HasActed(ctx context.Context, fromID, toID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM likes WHERE from_user_id = $1 AND to_user_id = $2)
	`, fromID, toID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Swipe(ctx context.Context, fromID, toID, action string) (LikeResult, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return LikeResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	a, b := fromID, toID
	if a > b {
		a, b = b, a
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, a+":"+b); err != nil {
		return LikeResult{}, err
	}

	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)`, fromID, toID).Scan(&blocked); err != nil {
		return LikeResult{}, err
	}
	if blocked {
		return LikeResult{}, ErrNotEligible
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO likes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)
		ON CONFLICT (from_user_id, to_user_id) DO NOTHING
	`, fromID, toID, action)
	if err != nil {
		return LikeResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return LikeResult{}, ErrAlreadyActed
	}

	result := LikeResult{}
	if action == ActionLike {
		var reciprocal bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM likes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')
		`, toID, fromID).Scan(&reciprocal); err != nil {
			return LikeResult{}, err
		}
		if reciprocal {
			ref, err := createMatch(ctx, tx, a, b)
			if err != nil {
				return LikeResult{}, err
			}
			result = LikeResult{Matched: true, Match: &ref}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return LikeResult{}, err
	}
	return result, nil
}

func createMatch(ctx context.Context, tx pgx.Tx, a, b string) (MatchRef, error) {
	var ref MatchRef
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (user_a, user_b) VALUES ($1, $2)
		ON CONFLICT (user_a, user_b) DO UPDATE SET user_a = EXCLUDED.user_a
		RETURNING id
	`, a, b).Scan(&ref.ID)
	if err != nil {
		return MatchRef{}, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO conversations (match_id) VALUES ($1)
		ON CONFLICT (match_id) DO UPDATE SET match_id = EXCLUDED.match_id
		RETURNING id
	`, ref.ID).Scan(&ref.ConversationID)
	if err != nil {
		return MatchRef{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO conversation_participants (conversation_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING
	`, ref.ConversationID, a, b); err != nil {
		return MatchRef{}, err
	}
	// One notification per participant, naming the other person.
	if _, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		SELECT x.user_id, 'match', 'Nouveau match', 'Vous avez un match avec ' || p.first_name,
		       jsonb_build_object('matchId', $1::text, 'conversationId', $2::text, 'userId', x.other_id::text, 'name', p.first_name)
		FROM (VALUES ($3::uuid, $4::uuid), ($4::uuid, $3::uuid)) AS x(user_id, other_id)
		JOIN profiles p ON p.user_id = x.other_id
	`, ref.ID, ref.ConversationID, a, b); err != nil {
		return MatchRef{}, err
	}
	return ref, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]StoredMatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, c.id, CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END, m.created_at, c.last_message_at
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_a = $1 OR m.user_b = $1)
		  AND m.unmatched_at IS NULL
		  AND ($2::timestamptz IS NULL OR m.created_at < $2)
		  AND NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = $1 AND b.blocked_id IN (m.user_a, m.user_b))
			   OR (b.blocked_id = $1 AND b.blocker_id IN (m.user_a, m.user_b))
		  )
		ORDER BY m.created_at DESC, m.id
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredMatch{}
	for rows.Next() {
		var m StoredMatch
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.OtherUserID, &m.CreatedAt, &m.LastMessageAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var a, b string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $2
		WHERE id = $1 AND (user_a = $2 OR user_b = $2) AND unmatched_at IS NULL
		RETURNING user_a, user_b
	`, matchID, userID).Scan(&a, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if a == userID {
		return b, nil
	}
	return a, nil
}
