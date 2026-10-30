package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("photo not found")
	ErrLimit      = errors.New("photo limit reached")
	ErrBadReorder = errors.New("photo order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Add(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID, key string, width, height, size int) (oldKey string, err error)
	Remove(ctx context.Context, userID, photoID string) (key string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	KeyByID(ctx context.Context, photoID string) (string, error)
	KeysByUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, storage_key, position, width, height
		FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.StorageKey, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// lockUser serializes photo mutations of one user for the transaction.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1, 0))`, userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotos {
		return storedPhoto{}, ErrLimit
	}

	p := storedPhoto{StorageKey: key, Position: count, Width: width, Height: height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, width, height, size_bytes, position)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
		RETURNING id::text`, userID, key, width, height, size, count).Scan(&p.ID); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, key string, width, height, size int) (string, error) {
	var oldKey string
	err := r.pool.QueryRow(ctx, `
		WITH old AS (SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE)
		UPDATE photos p SET storage_key = $3, width = $4, height = $5, size_bytes = $6
		FROM old WHERE p.id = $1 AND p.user_id = $2
		RETURNING old.storage_key`, photoID, userID, key, width, height, size).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return oldKey, err
}

func (r *PGRepository) Remove(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var key string
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}

	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*) FROM photos WHERE user_id = $1`,
		userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != total || total != len(orderedIDs) {
		return ErrBadReorder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) KeyByID(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

func (r *PGRepository) KeysByUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
