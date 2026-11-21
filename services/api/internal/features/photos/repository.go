package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrPhotoLimit    = errors.New("photo limit reached")
	ErrInvalidOrder  = errors.New("photo order must list each of your photos exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Insert(ctx context.Context, photo StoredPhoto, maxPhotos int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID string, next StoredPhoto) (StoredPhoto, StoredPhoto, error)
	Delete(ctx context.Context, userID, photoID string) (StoredPhoto, error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	GetByID(ctx context.Context, photoID string) (StoredPhoto, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const photoColumns = `id, user_id, storage_key, thumb_storage_key, position::int, width, height`

func scanPhoto(row pgx.Row) (StoredPhoto, error) {
	var p StoredPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.ThumbStorageKey, &p.Position, &p.Width, &p.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPhoto{}, ErrPhotoNotFound
	}
	return p, err
}

// lockUserPhotos serializes photo mutations of one user (positions stay compact).
func lockUserPhotos(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1, 0))`, userID)
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredPhoto{}
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Insert(ctx context.Context, photo StoredPhoto, maxPhotos int) (StoredPhoto, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, photo.UserID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, photo.UserID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= maxPhotos {
		return StoredPhoto{}, ErrPhotoLimit
	}
	stored, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, thumb_storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+photoColumns,
		photo.UserID, photo.StorageKey, photo.ThumbStorageKey, count, photo.Width, photo.Height, photo.SizeBytes))
	if err != nil {
		return StoredPhoto{}, err
	}
	return stored, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, next StoredPhoto) (StoredPhoto, StoredPhoto, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return StoredPhoto{}, StoredPhoto{}, err
	}
	old, err := scanPhoto(tx.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID))
	if err != nil {
		return StoredPhoto{}, StoredPhoto{}, err
	}
	updated, err := scanPhoto(tx.QueryRow(ctx, `
		UPDATE photos
		SET storage_key = $3, thumb_storage_key = $4, width = $5, height = $6, size_bytes = $7, created_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING `+photoColumns, photoID, userID, next.StorageKey, next.ThumbStorageKey, next.Width, next.Height, next.SizeBytes))
	if err != nil {
		return StoredPhoto{}, StoredPhoto{}, err
	}
	return old, updated, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (StoredPhoto, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	deleted, err := scanPhoto(tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING `+photoColumns, photoID, userID))
	if err != nil {
		return StoredPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p
		SET position = ordered.rn - 1
		FROM (
			SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn
			FROM photos WHERE user_id = $1
		) ordered
		WHERE p.id = ordered.id
	`, userID); err != nil {
		return StoredPhoto{}, err
	}
	return deleted, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return err
	}
	var total, matched int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE id = ANY($2::uuid[]))
		FROM photos WHERE user_id = $1
	`, userID, photoIDs).Scan(&total, &matched); err != nil {
		return err
	}
	if total != len(photoIDs) || matched != len(photoIDs) {
		return ErrInvalidOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos
		SET position = array_position($2::uuid[], id) - 1
		WHERE user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetByID(ctx context.Context, photoID string) (StoredPhoto, error) {
	return scanPhoto(r.dbPool.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, photoID))
}
