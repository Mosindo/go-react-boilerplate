package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound   = errors.New("photo not found")
	ErrTooManyPhotos   = errors.New("a profile can hold at most 6 photos")
	ErrInvalidOrdering = errors.New("photo order must list each of your photos exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	ListStorageKeys(ctx context.Context, userID string) ([]string, error)
	Create(ctx context.Context, userID, storageKey string, img ProcessedImage) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, img ProcessedImage) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	GetStorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const photoColumns = `id, user_id, position, storage_key, size_bytes, width, height, created_at`

func scanPhoto(row pgx.CollectableRow) (StoredPhoto, error) {
	var p StoredPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.SizeBytes, &p.Width, &p.Height, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPhoto)
}

func (r *PGRepository) ListStorageKeys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// lockOwner serializes photo mutations of a user so position bookkeeping is
// race-free.
func lockOwner(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPhotoNotFound
	}
	return err
}

func (r *PGRepository) Create(ctx context.Context, userID, storageKey string, img ProcessedImage) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockOwner(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrTooManyPhotos
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, count, storageKey, len(img.Data), img.Width, img.Height)
	if err != nil {
		return StoredPhoto{}, err
	}
	photo, err := pgx.CollectExactlyOneRow(rows, scanPhoto)
	if err != nil {
		return StoredPhoto{}, err
	}
	return photo, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, img ProcessedImage) (string, error) {
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
	`, photoID, userID, storageKey, len(img.Data), img.Width, img.Height); err != nil {
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
	var position int
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position
	`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
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
	rows, err := tx.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if !samePermutation(existing, photoIDs) {
		return ErrInvalidOrdering
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = array_position($2::uuid[], id) - 1 WHERE user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetStorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	return key, err
}

func samePermutation(existing, requested []string) bool {
	if len(existing) != len(requested) {
		return false
	}
	set := make(map[string]int, len(existing))
	for _, id := range existing {
		set[id]++
	}
	for _, id := range requested {
		if set[id] != 1 {
			return false
		}
		set[id]--
	}
	return true
}
