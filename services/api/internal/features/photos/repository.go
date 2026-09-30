package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryFull     = errors.New("photo limit reached")
	ErrRepositoryMismatch = errors.New("photo set mismatch")
)

type Repository interface {
	Insert(ctx context.Context, userID, storageKey string, width, height, size, maxPhotos int) (string, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (oldKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	ListIDs(ctx context.Context, userID string) ([]string, error)
	ListKeys(ctx context.Context, userID string) ([]string, error)
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// lockUser serialises photo mutations for one user via the profile row.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size, maxPhotos int) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return "", err
	}
	if count >= maxPhotos {
		return "", ErrRepositoryFull
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, storageKey, count, width, height, size).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, photoID, userID, storageKey, width, height, size); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
}

// Delete removes a photo and closes the gap so positions stay 0..n-1.
func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var oldKey string
	var position int
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2
		RETURNING storage_key, position
	`, photoID, userID).Scan(&oldKey, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1
		WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
}

// Reorder applies a full ordering; orderedIDs must be exactly the user's set.
func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}

	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*)
		FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != len(orderedIDs) || total != len(orderedIDs) {
		return ErrRepositoryMismatch
	}

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p
		SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ListIDs(ctx context.Context, userID string) ([]string, error) {
	return r.listColumn(ctx, `SELECT id::text FROM photos WHERE user_id = $1 ORDER BY position`, userID)
}

func (r *PGRepository) ListKeys(ctx context.Context, userID string) ([]string, error) {
	return r.listColumn(ctx, `SELECT storage_key FROM photos WHERE user_id = $1 ORDER BY position`, userID)
}

func (r *PGRepository) listColumn(ctx context.Context, query, userID string) ([]string, error) {
	rows, err := r.dbPool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.dbPool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return key, err
}
