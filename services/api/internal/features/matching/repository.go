package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotReady    = errors.New("viewer profile is incomplete")
	ErrRepositoryUnavailable = errors.New("profile unavailable")
	ErrRepositoryNotFound    = errors.New("match not found")
)

// eligibleCandidates selects the users a viewer ($1) may be shown / may swipe:
// active, visible, located, with a photo, matching both sides' gender, age and
// distance preferences, not blocked in either direction. ($2 = optional target).
// This single predicate backs both discovery and swipe authorisation, so a
// client can never act on a profile the server would not have proposed.
const eligibleCandidates = `
	FROM profiles v
	JOIN user_preferences vp ON vp.user_id = v.user_id
	JOIN profiles p ON p.user_id <> v.user_id
	JOIN user_preferences cp ON cp.user_id = p.user_id
	JOIN users u ON u.id = p.user_id AND u.status = 'active'
	WHERE v.user_id = $1
	  AND ($2::uuid IS NULL OR p.user_id = $2::uuid)
	  AND v.latitude IS NOT NULL
	  AND p.is_visible
	  AND p.latitude IS NOT NULL
	  AND p.latitude BETWEEN v.latitude - vp.max_distance_km / 111.0 AND v.latitude + vp.max_distance_km / 111.0
	  AND p.gender = ANY (vp.interested_in)
	  AND v.gender = ANY (cp.interested_in)
	  AND date_part('year', age(p.birth_date)) BETWEEN vp.min_age AND vp.max_age
	  AND date_part('year', age(v.birth_date)) BETWEEN cp.min_age AND cp.max_age
	  AND haversine_km(v.latitude, v.longitude, p.latitude, p.longitude) <= LEAST(vp.max_distance_km, cp.max_distance_km)
	  AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
	  AND NOT EXISTS (SELECT 1 FROM blocks b
	       WHERE (b.blocker_id = v.user_id AND b.blocked_id = p.user_id)
	          OR (b.blocker_id = p.user_id AND b.blocked_id = v.user_id))
`

type SwipeOutcome struct {
	AlreadySwiped bool
	MatchID       string
	MatchedAt     time.Time
}

type Repository interface {
	ViewerReady(ctx context.Context, userID string) (bool, error)
	TouchActive(ctx context.Context, userID string) error
	Candidates(ctx context.Context, viewerID string, limit int) ([]string, error)
	Swipe(ctx context.Context, from, to, action string) (SwipeOutcome, error)
	GetMatch(ctx context.Context, userID, matchID string) (otherUserID string, createdAt time.Time, err error)
	Unmatch(ctx context.Context, userID, matchID string) (otherUserID string, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) ViewerReady(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM profiles p
		  JOIN user_preferences up ON up.user_id = p.user_id
		  WHERE p.user_id = $1 AND p.latitude IS NOT NULL
		    AND EXISTS (SELECT 1 FROM photos ph WHERE ph.user_id = p.user_id)
		)`, userID).Scan(&ok)
	return ok, err
}

func (r *PGRepository) TouchActive(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE profiles SET last_active_at = NOW()
		WHERE user_id = $1 AND last_active_at < NOW() - INTERVAL '1 minute'`, userID)
	return err
}

func (r *PGRepository) Candidates(ctx context.Context, viewerID string, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.user_id `+eligibleCandidates+`
		  AND NOT EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = v.user_id AND s.to_user_id = p.user_id)
		ORDER BY EXISTS (SELECT 1 FROM swipes s WHERE s.from_user_id = p.user_id AND s.to_user_id = v.user_id AND s.action = 'like') DESC,
		         p.last_active_at DESC, p.user_id
		LIMIT $3
	`, viewerID, nil, limit)
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

// Swipe records the decision and creates the match when interest is mutual.
// An advisory lock on the pair makes concurrent reciprocal likes safe: exactly
// one of them observes the other and creates the single match row.
func (r *PGRepository) Swipe(ctx context.Context, from, to, action string) (SwipeOutcome, error) {
	var out SwipeOutcome
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(least($1::text, $2::text) || ':' || greatest($1::text, $2::text), 0))`, from, to); err != nil {
		return out, err
	}

	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 `+eligibleCandidates+`)`, from, to).Scan(&eligible); err != nil {
		return out, mapUUID(err)
	}
	if !eligible {
		return out, ErrRepositoryUnavailable
	}

	var already bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2)`, from, to).Scan(&already); err != nil {
		return out, err
	}
	if already {
		out.AlreadySwiped = true
		err := tx.QueryRow(ctx, `
			SELECT id, created_at FROM matches WHERE user_a = least($1::uuid, $2::uuid) AND user_b = greatest($1::uuid, $2::uuid) AND ended_at IS NULL
		`, from, to).Scan(&out.MatchID, &out.MatchedAt)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		return out, tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, $3)`, from, to, action); err != nil {
		return out, err
	}

	if action == ActionLike {
		var mutual bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM swipes WHERE from_user_id = $1 AND to_user_id = $2 AND action = 'like')`, to, from).Scan(&mutual); err != nil {
			return out, err
		}
		if mutual {
			err := tx.QueryRow(ctx, `
				INSERT INTO matches (user_a, user_b) VALUES (least($1::uuid, $2::uuid), greatest($1::uuid, $2::uuid))
				ON CONFLICT (user_a, user_b) DO NOTHING
				RETURNING id, created_at
			`, from, to).Scan(&out.MatchID, &out.MatchedAt)
			if err != nil {
				return out, err
			}
			for _, pair := range [][2]string{{from, to}, {to, from}} {
				if err := notifications.Create(ctx, tx, notifications.New{
					UserID: pair[0], Type: notifications.TypeMatch, ActorID: pair[1], MatchID: out.MatchID,
				}); err != nil {
					return out, err
				}
			}
		}
	}
	return out, tx.Commit(ctx)
}

func (r *PGRepository) GetMatch(ctx context.Context, userID, matchID string) (string, time.Time, error) {
	var other string
	var created time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT CASE WHEN user_a = $1 THEN user_b ELSE user_a END, created_at
		FROM matches WHERE id = $2 AND ended_at IS NULL AND ($1 = user_a OR $1 = user_b)
	`, userID, matchID).Scan(&other, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, ErrRepositoryNotFound
	}
	return other, created, mapNotFound(err)
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var other string
	err = tx.QueryRow(ctx, `
		UPDATE matches SET ended_at = NOW()
		WHERE id = $2 AND ended_at IS NULL AND ($1 = user_a OR $1 = user_b)
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END
	`, userID, matchID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", mapNotFound(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE match_id = $1`, matchID); err != nil {
		return "", err
	}
	return other, tx.Commit(ctx)
}

func mapUUID(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrRepositoryUnavailable
	}
	return err
}

func mapNotFound(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrRepositoryNotFound
	}
	return err
}
