package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryUserNotFound = errors.New("safety: user not found")

type Repository interface {
	Block(ctx context.Context, blockerID, blockedID string) (conversationID string, wasMatched bool, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) userExists(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) error {
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrRepositoryUserNotFound
	}
	return nil
}

// Block hides both people from each other and removes any match between them
// (its conversation and messages cascade).
func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, bool, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.userExists(ctx, tx, blockedID); err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", false, err
	}

	var conversationID string
	matched := true
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(c.id::text, '') FROM matches m
		LEFT JOIN conversations c ON c.match_id = m.id
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
	`, blockerID, blockedID).Scan(&conversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		matched = false
	} else if err != nil {
		return "", false, err
	}
	if matched {
		if _, err := tx.Exec(ctx, `
			DELETE FROM matches WHERE user_a = LEAST($1::uuid, $2::uuid) AND user_b = GREATEST($1::uuid, $2::uuid)
		`, blockerID, blockedID); err != nil {
			return "", false, err
		}
	}
	return conversationID, matched, tx.Commit(ctx)
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.dbPool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC, b.blocked_id
		LIMIT $2 OFFSET $3
	`, blockerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]BlockedUser, 0, limit)
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.BlockedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// CreateReport stores a report unless the same reporter already has an open
// one against the same person from the last 24 hours (idempotent for retries and double taps).
func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if err := r.userExists(ctx, r.dbPool, reportedID); err != nil {
		return err
	}
	_, err := r.dbPool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM reports
			WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'
			  AND created_at > NOW() - INTERVAL '24 hours'
		)
	`, reporterID, reportedID, reason, details)
	return err
}
