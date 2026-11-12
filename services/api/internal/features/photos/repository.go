package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID string, p newPhoto) error
	Replace(ctx context.Context, userID, photoID string, p newPhoto) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, storage_key, size_bytes, width, height, position
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedPhoto, error) {
		var p storedPhoto
		err := row.Scan(&p.ID, &p.StorageKey, &p.SizeBytes, &p.Width, &p.Height, &p.Position)
		return p, err
	})
}

// lockOwner serializes photo mutations for one user via the profile row.
func lockOwner(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProfileRequired
	}
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID string, p newPhoto) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockOwner(ctx, tx, userID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count >= MaxPhotos {
		return ErrTooManyPhotos
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (user_id, storage_key, size_bytes, width, height, position)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, p.StorageKey, p.SizeBytes, p.Width, p.Height, count); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p newPhoto) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET storage_key = $3, size_bytes = $4, width = $5, height = $6, created_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, photoID, userID, p.StorageKey, p.SizeBytes, p.Width, p.Height); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockOwner(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	// Compact positions so the first photo is always the primary one (position 0).
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) o
		WHERE p.id = o.id
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockOwner(ctx, tx, userID); err != nil {
		return err
	}
	var total, matched int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE id = ANY($2::uuid[])) FROM photos WHERE user_id = $1
	`, userID, photoIDs).Scan(&total, &matched); err != nil {
		return err
	}
	if total != len(photoIDs) || matched != len(photoIDs) {
		return ErrInvalidOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
