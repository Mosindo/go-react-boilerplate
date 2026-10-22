package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	// Block records the block and removes any match (and its conversation) between the pair.
	// It returns the removed match/conversation IDs, empty when there was none.
	Block(ctx context.Context, blockerID, blockedID string) (matchID, conversationID string, err error)
	Unblock(ctx context.Context, blockerID, blockedID string) error
	ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) userExists(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

func (r *PGRepository) Block(ctx context.Context, blockerID, blockedID string) (string, string, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ok, err := r.userExists(ctx, tx, blockedID)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", ErrUserNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", "", err
	}

	var matchID, conversationID string
	err = tx.QueryRow(ctx, `
		DELETE FROM matches m
		USING conversations c
		WHERE c.match_id = m.id
		  AND m.user_a_id = LEAST($1::uuid, $2::uuid) AND m.user_b_id = GREATEST($1::uuid, $2::uuid)
		RETURNING m.id, c.id
	`, blockerID, blockedID).Scan(&matchID, &conversationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return matchID, conversationID, nil
}

func (r *PGRepository) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blockerID, blockedID)
	return err
}

func (r *PGRepository) ListBlocked(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
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
	out := make([]BlockedUser, 0)
	for rows.Next() {
		var u BlockedUser
		if err := rows.Scan(&u.UserID, &u.FirstName, &u.BlockedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *PGRepository) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	ok, err := r.userExists(ctx, r.pool, reportedID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	// One open report per reporter/reported pair: repeating it is a no-op, not a spam vector.
	_, err = r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1 FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'
		)
	`, reporterID, reportedID, reason, details)
	return err
}
