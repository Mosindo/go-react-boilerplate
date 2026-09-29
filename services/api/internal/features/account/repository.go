package account

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("account not found")

type Repository interface {
	Me(ctx context.Context, userID string, now time.Time) (auth.MeResponse, error)
	PasswordHash(ctx context.Context, userID string) (string, error)
	// ChangePassword stores the new hash and revokes every session except keepSessionID.
	ChangePassword(ctx context.Context, userID, newHash, keepSessionID string) error
	// Delete removes the user (all dependent rows cascade) in one transaction and returns the
	// storage keys of the photos that were owned by the user.
	Delete(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Me(ctx context.Context, userID string, now time.Time) (auth.MeResponse, error) {
	me, err := auth.LoadMe(ctx, r.pool, userID, now)
	if errors.Is(err, auth.ErrUserNotFound) {
		return auth.MeResponse{}, ErrNotFound
	}
	return me, err
}

func (r *PGRepository) PasswordHash(ctx context.Context, userID string) (string, error) {
	var h string
	err := r.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return h, err
}

func (r *PGRepository) ChangePassword(ctx context.Context, userID, newHash, keepSessionID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`, userID, newHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL AND id::text <> $2`, userID, keepSessionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID string) ([]string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return keys, nil
}
