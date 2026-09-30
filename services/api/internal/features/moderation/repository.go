package moderation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound  = errors.New("not found")
	ErrRepositoryProtected = errors.New("target is a moderator")
)

type Repository interface {
	Role(ctx context.Context, userID string) (string, error)
	ListReports(ctx context.Context, status string, limit, offset int) ([]reportRow, error)
	ResolveReport(ctx context.Context, reportID, moderatorID, status string) error
	SetSuspended(ctx context.Context, userID, moderatorID string, suspended bool) error
	SetRoleByEmail(ctx context.Context, email, role string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) Role(ctx context.Context, userID string) (string, error) {
	var role string
	err := r.dbPool.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 AND suspended_at IS NULL`, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return role, err
}

func (r *PGRepository) ListReports(ctx context.Context, status string, limit, offset int) ([]reportRow, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT r.id, r.reason, r.details, r.status, r.created_at, r.reviewed_at, r.reported_id,
		       (SELECT COUNT(DISTINCT o.reporter_id) FROM reports o WHERE o.reported_id = r.reported_id AND o.status = 'open'),
		       COALESCE(u.suspended_at IS NOT NULL, FALSE)
		FROM reports r
		LEFT JOIN users u ON u.id = r.reported_id
		WHERE r.status = $1
		ORDER BY r.created_at ASC, r.id
		LIMIT $2 OFFSET $3
	`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []reportRow{}
	for rows.Next() {
		var row reportRow
		if err := rows.Scan(&row.ID, &row.Reason, &row.Details, &row.Status, &row.CreatedAt, &row.ReviewedAt, &row.ReportedID, &row.OpenReports, &row.Suspended); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *PGRepository) ResolveReport(ctx context.Context, reportID, moderatorID, status string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE reports SET status = $3, reviewed_at = NOW(), reviewed_by = $2
		WHERE id = $1 AND status = 'open'
	`, reportID, moderatorID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

// SetSuspended suspends (revoking every session and push token and closing
// the member's open reports as reviewed) or reinstates a member. Moderators cannot be suspended
// through the API.
func (r *PGRepository) SetSuspended(ctx context.Context, userID, moderatorID string, suspended bool) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var role string
	err = tx.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if role == RoleModerator {
		return ErrRepositoryProtected
	}

	if suspended {
		if _, err := tx.Exec(ctx, `UPDATE users SET suspended_at = COALESCE(suspended_at, NOW()) WHERE id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM push_tokens WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE reports SET status = 'reviewed', reviewed_at = NOW(), reviewed_by = $2
			WHERE reported_id = $1 AND status = 'open'
		`, userID, moderatorID); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `UPDATE users SET suspended_at = NULL WHERE id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) SetRoleByEmail(ctx context.Context, email, role string) error {
	tag, err := r.dbPool.Exec(ctx, `UPDATE users SET role = $2 WHERE email = $1`, email, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}
