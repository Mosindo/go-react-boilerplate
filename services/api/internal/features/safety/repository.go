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
	Block(ctx context.Context, blockerID, blockedID string) (matchIDs []string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Block records the block and ends any match between the two people (their
// conversation disappears with it) in one transaction.
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) ([]string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		DELETE FROM matches
		WHERE user_a = least($1::uuid, $2::uuid) AND user_b = greatest($1::uuid, $2::uuid)
		RETURNING id
	`, blockerID, blockedID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return ids, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC
		LIMIT 200
	`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateReport stores a report. A second open report from the same person about
// the same user within 24 hours returns the first one instead of piling up.
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT id FROM reports
		WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open' AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC LIMIT 1
	`, reporterID, reportedID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	err = r.db.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4) RETURNING id
	`, reporterID, reportedID, reason, details).Scan(&id)
	if isForeignKeyViolation(err) {
		return "", ErrUserNotFound
	}
	return id, err
}
