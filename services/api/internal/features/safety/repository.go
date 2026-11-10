package safety

import (
	"context"
	"errors"

	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (removedMatchID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) (removedMatchID string, err error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (string, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING
	`, blockerID, blockedID)
	if db.IsForeignKeyViolation(err) {
		return "", ErrRepoUserNotFound
	}
	if err != nil {
		return "", err
	}
	var matchID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches
		WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		RETURNING id
	`, blockerID, blockedID).Scan(&matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return matchID, err
}

// Block hides both people from each other everywhere and ends any match
// (which deletes the conversation).
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, error) {
	var matchID string
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		matchID, err = blockTx(ctx, tx, blockerID, blockedID)
		return err
	})
	return matchID, err
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BlockedUser, error) {
		var u BlockedUser
		err := row.Scan(&u.UserID, &u.FirstName, &u.CreatedAt)
		return u, err
	})
}

// Report stores the report, blocks the reported user for the reporter, and
// puts the reported profile under review once enough distinct people with a
// real profile reported it.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string, threshold int) (string, error) {
	var matchID string
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
			ON CONFLICT (reporter_id, reported_id) DO UPDATE SET reason = EXCLUDED.reason, details = EXCLUDED.details
		`, reporterID, reportedID, reason, details)
		if db.IsForeignKeyViolation(err) {
			return ErrRepoUserNotFound
		}
		if err != nil {
			return err
		}
		if matchID, err = blockTx(ctx, tx, reporterID, reportedID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE profiles SET moderation_status = 'under_review', updated_at = NOW()
			WHERE user_id = $1 AND moderation_status = 'active'
			  AND (
				SELECT COUNT(DISTINCT r.reporter_id)
				FROM reports r JOIN profiles rp ON rp.user_id = r.reporter_id
				WHERE r.reported_id = $1 AND r.status = 'open'
			  ) >= $2
		`, reportedID, threshold)
		return err
	})
	return matchID, err
}
