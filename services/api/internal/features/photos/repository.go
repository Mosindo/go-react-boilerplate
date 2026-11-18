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
	ErrRepositoryOrder    = errors.New("order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	GetStorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// lockUser serialises photo mutations of one member so positions stay dense.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, user_id, position, storage_key, width, height
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StoredPhoto, 0)
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrRepositoryFull
	}
	p := StoredPhoto{UserID: userID, Position: count, StorageKey: storageKey, Width: width, Height: height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, userID, count, storageKey, width, height, size).Scan(&p.ID); err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, error) {
	var oldKey string
	err := r.dbPool.QueryRow(ctx, `
		WITH old AS (
			SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE
		)
		UPDATE photos p
		SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		FROM old
		WHERE p.id = $1 AND p.user_id = $2
		RETURNING old.storage_key
	`, photoID, userID, storageKey, width, height, size).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return oldKey, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = ordered.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) ordered
		WHERE p.id = ordered.id AND p.position <> ordered.rn - 1
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var total, matching int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE id::text = ANY($2))
		FROM photos WHERE user_id = $1
	`, userID, photoIDs).Scan(&total, &matching); err != nil {
		return err
	}
	if total != len(photoIDs) || matching != len(photoIDs) {
		return ErrRepositoryOrder
	}
	// The (user_id, position) constraint is deferred, so the permutation can
	// be applied in a single statement.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetStorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.dbPool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return key, err
}
