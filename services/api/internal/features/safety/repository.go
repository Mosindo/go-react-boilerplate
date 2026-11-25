package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepoUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (endedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// Block records the block and ends any active match between the two users,
// which immediately closes their conversation for both sides.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	var endedMatchID string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Same pair lock as matching.Swipe so a concurrent like cannot create
		// a match that survives this block.
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtext('pair:' || LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text)))
		`, blockerID, blockedID); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errRepoUserNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, blockerID, blockedID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			UPDATE matches SET ended_at = NOW(), ended_by = $1
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid) AND ended_at IS NULL
			RETURNING id
		`, blockerID, blockedID).Scan(&endedMatchID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	return endedMatchID, err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, u.id, $3, $4 FROM users u WHERE u.id = $2
		RETURNING id
	`, reporterID, reportedID, reason, details).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errRepoUserNotFound
	}
	return id, err
}
