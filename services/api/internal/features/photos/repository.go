package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("photo not found")
	ErrLimitReached    = errors.New("photo limit reached")
	ErrNotAPermutation = errors.New("photoIds must contain each of your photos exactly once")
)

type Repository interface {
	Count(ctx context.Context, userID string) (int, error)
	// Add appends a photo atomically, enforcing the per-user maximum.
	Add(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error)
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	// Remove deletes the row, re-packs positions and returns the storage key.
	Remove(ctx context.Context, userID, photoID string) (string, error)
	// File returns the storage key and byte size of a photo.
	File(ctx context.Context, photoID string) (key string, size int64, err error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) Count(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

// lockUser serialises photo mutations of one user (upload/reorder/delete) inside tx.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}
	var count, next int
	if err := tx.QueryRow(ctx, `SELECT count(*), COALESCE(max(position) + 1, 0) FROM photos WHERE user_id = $1`, userID).Scan(&count, &next); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotosPerUser {
		return storedPhoto{}, ErrLimitReached
	}
	var p storedPhoto
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, position`, userID, key, next, width, height, size).Scan(&p.ID, &p.Position)
	if err != nil {
		return storedPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return storedPhoto{}, err
	}
	return p, nil
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedPhoto{}
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	// Exact permutation check under the lock: same cardinality and every id belongs to the user.
	var owned, matched int
	err = tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM photos WHERE user_id = $1),
		       (SELECT count(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[]))`, userID, ids).Scan(&owned, &matched)
	if err != nil {
		return err
	}
	if owned != len(ids) || matched != len(ids) {
		return ErrNotAPermutation
	}
	// The unique(user_id, position) constraint is deferred, so intermediate states are fine.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Remove(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = s.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) s
		WHERE p.id = s.id`, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) File(ctx context.Context, photoID string) (string, int64, error) {
	var key string
	var size int64
	err := r.pool.QueryRow(ctx, `SELECT storage_key, size_bytes FROM photos WHERE id = $1`, photoID).Scan(&key, &size)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrNotFound
	}
	return key, size, err
}
