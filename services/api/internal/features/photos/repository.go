package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	ExistsByKey(ctx context.Context, storageKey string) (bool, error)
	IsDiscoverable(ctx context.Context, userID string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	return list(ctx, r.db, userID)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func list(ctx context.Context, q querier, userID string) ([]StoredPhoto, error) {
	rows, err := q.Query(ctx, `
		SELECT id, storage_key, position, width, height FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredPhoto, error) {
		var p StoredPhoto
		err := row.Scan(&p.ID, &p.StorageKey, &p.Position, &p.Width, &p.Height)
		return p, err
	})
}

// lockUser serialises photo mutations of a user so the 6-photo cap and positions stay consistent.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrLimitReached
	}
	p := StoredPhoto{StorageKey: storageKey, Position: count, Width: width, Height: height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, storageKey, count, width, height, size).Scan(&p.ID); err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, error) {
	var oldKey string
	err := r.db.QueryRow(ctx, `
		WITH old AS (SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE)
		UPDATE photos p SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		FROM old
		WHERE p.id = $1 AND p.user_id = $2
		RETURNING old.storage_key
	`, photoID, userID, storageKey, width, height, size).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return oldKey, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	var position int
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position
	`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	// Close the gap so positions stay contiguous (position 0 is the main photo).
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	current, err := list(ctx, tx, userID)
	if err != nil {
		return err
	}
	if len(current) != len(ids) {
		return ErrInvalidOrder
	}
	owned := make(map[string]bool, len(current))
	for _, p := range current {
		owned[p.ID] = true
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !owned[id] || seen[id] {
			return ErrInvalidOrder
		}
		seen[id] = true
	}
	// The (user_id, position) unique constraint is deferred until commit.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) ExistsByKey(ctx context.Context, storageKey string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE storage_key = $1)`, storageKey).Scan(&ok)
	return ok, err
}

func (r *PGRepository) IsDiscoverable(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT COALESCE((SELECT is_discoverable FROM preferences WHERE user_id = $1), FALSE)`, userID).Scan(&ok)
	return ok, err
}
