package safety

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// pairKey must stay identical to the matching feature's key: swipes and blocks on one pair are
// serialised through the same advisory lock.
func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return "pair:" + a + ":" + b
}

func userExists(ctx context.Context, q pgx.Tx, id string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

// blockTx inserts the block (idempotent) and deletes any match between the pair.
func blockTx(ctx context.Context, tx pgx.Tx, blocker, blocked string) (*removedMatch, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, pairKey(blocker, blocked)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)
		ON CONFLICT (blocker_id, blocked_id) DO NOTHING`, blocker, blocked); err != nil {
		return nil, err
	}
	var rm removedMatch
	err := tx.QueryRow(ctx, `
		DELETE FROM matches m
		WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
		RETURNING m.user_a, m.user_b, COALESCE((SELECT c.id FROM conversations c WHERE c.match_id = m.id), '00000000-0000-0000-0000-000000000000')`,
		blocker, blocked).Scan(&rm.UserA, &rm.UserB, &rm.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rm, nil
}

func (r *Repository) Block(ctx context.Context, blocker, blocked string) (*removedMatch, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := userExists(ctx, tx, blocked)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	rm, err := blockTx(ctx, tx, blocker, blocked)
	if err != nil {
		return nil, err
	}
	return rm, tx.Commit(ctx)
}

func (r *Repository) Unblock(ctx context.Context, blocker, blocked string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, blocker, blocked)
	return err
}

func (r *Repository) ListBlocks(ctx context.Context, blocker string) ([]BlockedUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), b.created_at
		FROM blocks b LEFT JOIN profiles p ON p.user_id = b.blocked_id
		WHERE b.blocker_id = $1
		ORDER BY b.created_at DESC, b.blocked_id
		LIMIT 500`, blocker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.FirstName, &b.BlockedAt); err != nil {
			return nil, err
		}
		b.BlockedAt = b.BlockedAt.UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

// CreateReport stores a report (or returns the caller's open report against the same user from the
// last 24 hours), optionally blocking the reported user in the same transaction.
func (r *Repository) CreateReport(ctx context.Context, reporter, reported, reason, details string, block bool) (string, *removedMatch, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ok, err := userExists(ctx, tx, reported)
	if err != nil {
		return "", nil, err
	}
	if !ok {
		return "", nil, ErrNotFound
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "report:"+reporter+":"+reported); err != nil {
		return "", nil, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		SELECT id FROM reports
		WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open' AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC LIMIT 1`, reporter, reported).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4)
			RETURNING id`, reporter, reported, reason, details).Scan(&id)
	}
	if err != nil {
		return "", nil, err
	}

	var rm *removedMatch
	if block {
		if rm, err = blockTx(ctx, tx, reporter, reported); err != nil {
			return "", nil, err
		}
	}
	return id, rm, tx.Commit(ctx)
}
