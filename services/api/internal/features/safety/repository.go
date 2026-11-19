package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*Closed, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]StoredBlock, error)
	Report(ctx context.Context, reporterID, reportedID, reason, details string) (*Closed, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*Closed, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	closed, err := block(ctx, tx, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	return closed, tx.Commit(ctx)
}

// block inserts the block, ends any match between the two users and removes the
// notifications they generated for each other.
func block(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (*Closed, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isFKViolation(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	var closed Closed
	err := tx.QueryRow(ctx, `
		UPDATE matches m SET unmatched_at = NOW(), unmatched_by = $1
		FROM conversations c
		WHERE c.match_id = m.id
		  AND m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
		  AND m.unmatched_at IS NULL
		RETURNING m.id, c.id
	`, blockerID, blockedID).Scan(&closed.MatchID, &closed.ConversationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE (user_id = $1 AND actor_id = $2) OR (user_id = $2 AND actor_id = $1)
	`, blockerID, blockedID); err != nil {
		return nil, err
	}

	if closed.MatchID == "" {
		return nil, nil
	}
	return &closed, nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]StoredBlock, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, p.first_name, b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredBlock, error) {
		var s StoredBlock
		err := row.Scan(&s.UserID, &s.FirstName, &s.CreatedAt)
		return s, err
	})
}

// Report stores a moderation report and blocks the reported user in the same transaction.
func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, reason, details string) (*Closed, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
	`, reporterID, reportedID, reason, details); err != nil {
		if isFKViolation(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	closed, err := block(ctx, tx, reporterID, reportedID)
	if err != nil {
		return nil, err
	}
	return closed, tx.Commit(ctx)
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
