package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (*RemovedMatch, error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, block bool) (id string, removed *RemovedMatch, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// blockTx records the block and closes any active match between the two users.
func blockTx(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) (*RemovedMatch, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	var rm RemovedMatch
	var a, b string
	err := tx.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL
		RETURNING id, user_a, user_b, (SELECT id FROM conversations WHERE match_id = matches.id)
	`, blockerID, blockedID).Scan(&rm.MatchID, &a, &b, &rm.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rm.UserIDs = [2]string{a, b}
	return &rm, nil
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (*RemovedMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	rm, err := blockTx(ctx, tx, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	return rm, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.display_name, 'Unknown'), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 500
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.DisplayName, &u.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string, block bool) (string, *RemovedMatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, reportedID).Scan(&exists); err != nil {
		return "", nil, err
	}
	if !exists {
		return "", nil, ErrUserNotFound
	}

	// Re-submitting the same open report within a day is idempotent.
	var id string
	err = tx.QueryRow(ctx, `
		SELECT id FROM reports
		WHERE reporter_id = $1 AND reported_id = $2 AND reason = $3 AND status = 'open'
		  AND created_at > NOW() - INTERVAL '24 hours'
		LIMIT 1
	`, reporterID, reportedID, reason).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4) RETURNING id
		`, reporterID, reportedID, reason, details).Scan(&id)
	}
	if err != nil {
		return "", nil, err
	}

	var removed *RemovedMatch
	if block {
		if removed, err = blockTx(ctx, tx, reporterID, reportedID); err != nil {
			return "", nil, err
		}
	}
	return id, removed, tx.Commit(ctx)
}
