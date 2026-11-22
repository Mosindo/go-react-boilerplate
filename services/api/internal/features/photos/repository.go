package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID, storageKey string, size, width, height int) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, storage_key, position, width, height
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPhoto)
}

func scanPhoto(row pgx.CollectableRow) (storedPhoto, error) {
	var p storedPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height)
	return p, err
}

// lockUser serialises photo mutations of one account so positions stay dense.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPhotoNotFound
	}
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, size, width, height int) (storedPhoto, error) {
	var p storedPhoto
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= profiles.MaxPhotos {
			return ErrTooManyPhotos
		}
		return tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
			VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
			RETURNING id, user_id, storage_key, position, width, height
		`, userID, storageKey, size, width, height, count).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height)
	})
	return p, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (string, error) {
	var oldKey string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPhotoNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE photos SET storage_key = $3, size_bytes = $4, width = $5, height = $6, updated_at = NOW()
			WHERE id = $1 AND user_id = $2
		`, photoID, userID, storageKey, size, width, height)
		return err
	})
	return oldKey, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	var key string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var position int
		err := tx.QueryRow(ctx, `
			DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position
		`, photoID, userID).Scan(&key, &position)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPhotoNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position)
		return err
	})
	return key, err
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var owned int
		var total int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*)
			FROM photos WHERE user_id = $1
		`, userID, photoIDs).Scan(&owned, &total); err != nil {
			return err
		}
		if owned != len(photoIDs) || total != len(photoIDs) {
			return ErrInvalidOrder
		}
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos p SET position = o.ord - 1, updated_at = NOW()
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, photoIDs)
		return err
	})
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	return key, err
}
