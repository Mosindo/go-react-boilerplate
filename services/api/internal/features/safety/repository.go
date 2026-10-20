package safety

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errUserNotFound = platformerrors.NotFound("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*removedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]profiles.UserSummary, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Block is one transaction: record the block, dissolve any match (cascading to
// conversation and messages) and drop notifications linking the two users.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*removedMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, errUserNotFound
	}

	if _, err := tx.Exec(ctx, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		return nil, err
	}

	var removed *removedMatch
	rm := removedMatch{}
	err = tx.QueryRow(ctx, `
		WITH gone AS (
		  DELETE FROM matches
		  WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
		  RETURNING id
		)
		SELECT g.id::text, COALESCE((SELECT c.id::text FROM conversations c WHERE c.match_id = g.id), '') FROM gone g
	`, blockerID, blockedID).Scan(&rm.MatchID, &rm.ConversationID)
	switch {
	case err == nil:
		removed = &rm
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE (user_id = $1 AND actor_id = $2) OR (user_id = $2 AND actor_id = $1)`, blockerID, blockedID); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]profiles.UserSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id::text, COALESCE(pf.first_name, ''),
		       CASE WHEN ph.id IS NULL THEN NULL ELSE '/photos/' || ph.id::text || '/file' END
		FROM blocks b
		LEFT JOIN profiles pf ON pf.user_id = b.blocked_id
		LEFT JOIN photos ph ON ph.user_id = b.blocked_id AND ph.position = 0
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC, b.blocked_id
		LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []profiles.UserSummary{}
	for rows.Next() {
		var u profiles.UserSummary
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.PhotoURL); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2
	`, reporterID, reportedID, reason, details)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errUserNotFound
	}
	return nil
}
