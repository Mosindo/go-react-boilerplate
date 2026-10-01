package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrReportNotFound = errors.New("report not found")
)

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (Report, error)
	ListReports(ctx context.Context, status string, limit, offset int) ([]Report, error)
	ResolveReport(ctx context.Context, reportID, status string, suspend bool) (Report, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Block records the block and closes any active match with that user, atomically.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, blockerID, blockedID); err != nil {
		if isFKViolation(err) {
			return ErrUserNotFound
		}
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
		WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid) AND unmatched_at IS NULL`,
		blockerID, blockedID); err != nil {
		return err
	}
	// A blocked profile must never resurface in discovery: record a pass.
	if _, err := tx.Exec(ctx, `
		INSERT INTO swipes (from_user_id, to_user_id, action) VALUES ($1, $2, 'pass') ON CONFLICT DO NOTHING`,
		blockerID, blockedID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1 ORDER BY b.created_at DESC LIMIT $2 OFFSET $3`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockedUser, 0, 16)
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

const reportColumns = `r.id, r.reporter_id, r.reported_user_id, COALESCE(p.first_name, ''), r.reason, r.details, r.status, r.created_at, r.resolved_at`

func scanReport(row pgx.Row) (Report, error) {
	var rep Report
	err := row.Scan(&rep.ID, &rep.ReporterID, &rep.ReportedUserID, &rep.ReportedName, &rep.Reason, &rep.Details, &rep.Status, &rep.CreatedAt, &rep.ResolvedAt)
	return rep, err
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (Report, error) {
	rep, err := scanReport(r.pool.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO reports (reporter_id, reported_user_id, reason, details)
			VALUES ($1, $2, $3, $4) RETURNING *
		)
		SELECT `+reportColumns+` FROM ins r LEFT JOIN profiles p ON p.user_id = r.reported_user_id`,
		reporterID, reportedID, reason, details))
	if err != nil && isFKViolation(err) {
		return Report{}, ErrUserNotFound
	}
	return rep, err
}

func (r *PGRepository) ListReports(ctx context.Context, status string, limit, offset int) ([]Report, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+reportColumns+` FROM reports r LEFT JOIN profiles p ON p.user_id = r.reported_user_id
		WHERE ($1 = '' OR r.status = $1) ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Report, 0, 20)
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

func (r *PGRepository) ResolveReport(ctx context.Context, reportID, status string, suspend bool) (Report, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var reportedID string
	err = tx.QueryRow(ctx, `
		UPDATE reports SET status = $2, resolved_at = NOW() WHERE id = $1 RETURNING reported_user_id`, reportID, status).Scan(&reportedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	if err != nil {
		return Report{}, err
	}
	if suspend {
		if _, err := tx.Exec(ctx, `UPDATE users SET status = 'suspended', updated_at = NOW() WHERE id = $1 AND role <> 'admin'`, reportedID); err != nil {
			return Report{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, reportedID); err != nil {
			return Report{}, err
		}
	}
	rep, err := scanReport(tx.QueryRow(ctx, `
		SELECT `+reportColumns+` FROM reports r LEFT JOIN profiles p ON p.user_id = r.reported_user_id WHERE r.id = $1`, reportID))
	if err != nil {
		return Report{}, err
	}
	return rep, tx.Commit(ctx)
}
