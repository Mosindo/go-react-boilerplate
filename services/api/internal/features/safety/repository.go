package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrNotBlocked   = errors.New("user is not blocked")
	ErrSelf         = errors.New("you cannot do this to yourself")
)

type Repository interface {
	// Block creates the block and ends any match between the users, returning
	// the id of the ended match (if any).
	Block(ctx context.Context, blockerID, blockedID string) (endedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	UpsertReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	var matchID string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Same pair lock as matching.RecordSwipe: serializes block vs. like.
		if _, err := tx.Exec(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended(LEAST($1::text, $2::text) || ':' || GREATEST($1::text, $2::text), 0))
		`, blockerID, blockedID); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, blockedID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrUserNotFound
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, blockerID, blockedID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			UPDATE matches SET unmatched_at = NOW()
			WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL
			RETURNING id
		`, blockerID, blockedID).Scan(&matchID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	return matchID, err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotBlocked
	}
	return nil
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var b BlockedUser
		err := row.Scan(&b.UserID, &b.FirstName, &b.BlockedAt)
		return b, err
	})
}

func (r *PGRepository) UpsertReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (reporter_id, reported_id) WHERE reporter_id IS NOT NULL AND reported_id IS NOT NULL
		DO UPDATE SET reason = EXCLUDED.reason, details = EXCLUDED.details, status = 'open', updated_at = NOW()
	`, reporterID, reportedID, reason, details)
	return err
}
