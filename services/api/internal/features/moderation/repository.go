package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryUserNotFound = errors.New("user not found")

type Repository interface {
	// Block records the block and unmatches the pair; it returns the ids of matches that were closed.
	Block(ctx context.Context, blockerID, blockedID string) (closedMatchIDs []string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	// Report stores (or refreshes) a report and reports whether the account was suspended by it.
	Report(ctx context.Context, reporterID, reportedID, matchID, reason, details string, threshold int) (suspended bool, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func userExists(ctx context.Context, tx pgx.Tx, id string) (bool, error) {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

func block(ctx context.Context, tx pgx.Tx, blockerID, blockedID string) ([]string, error) {
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		UPDATE matches SET unmatched_at = NOW()
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL
		RETURNING id
	`, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := userExists(ctx, tx, blockedID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrRepositoryUserNotFound
	}
	ids, err := block(ctx, tx, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	return ids, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC, b.blocked_id
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) Report(ctx context.Context, reporterID, reportedID, matchID, reason, details string, threshold int) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := userExists(ctx, tx, reportedID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrRepositoryUserNotFound
	}
	// Evidence link only if the reporter really shares that match with the reported user.
	var match any
	if matchID != "" {
		var valid bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM matches
			  WHERE id = $1 AND user_a = LEAST($2::uuid, $3::uuid) AND user_b = GREATEST($2::uuid, $3::uuid))
		`, matchID, reporterID, reportedID).Scan(&valid); err != nil {
			return false, err
		}
		if valid {
			match = matchID
		}
	}
	// One report per (reporter, reported): repeating refreshes it, so a single person cannot inflate the count.
	if _, err := tx.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, match_id, reason, details)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (reporter_id, reported_id) DO UPDATE SET
		  reason = EXCLUDED.reason, details = EXCLUDED.details,
		  match_id = COALESCE(EXCLUDED.match_id, reports.match_id), status = 'open', created_at = NOW()
	`, reporterID, reportedID, match, reason, details); err != nil {
		return false, err
	}
	var suspended bool
	if err := tx.QueryRow(ctx, `
		WITH n AS (SELECT COUNT(DISTINCT reporter_id) AS c FROM reports WHERE reported_id = $1 AND status = 'open'),
		     upd AS (
		       UPDATE users SET status = 'suspended', updated_at = NOW()
		       WHERE id = $1 AND status = 'active' AND (SELECT c FROM n) >= $2
		       RETURNING id
		     ),
		     rev AS (
		       UPDATE sessions SET revoked_at = NOW()
		       WHERE user_id IN (SELECT id FROM upd) AND revoked_at IS NULL
		       RETURNING id
		     )
		SELECT EXISTS (SELECT 1 FROM upd)
	`, reportedID, threshold).Scan(&suspended); err != nil {
		return false, err
	}
	return suspended, tx.Commit(ctx)
}
