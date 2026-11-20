package photos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryLimit    = errors.New("photo limit reached")
	ErrRepositoryLast     = errors.New("cannot remove the last photo")
	ErrRepositoryMismatch = errors.New("photo ids do not match")
)

type StoredPhoto struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
	CreatedAt  time.Time
}

type NewPhoto struct {
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
}

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Insert(ctx context.Context, userID string, photo NewPhoto, maxPhotos int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID string, photo NewPhoto) (newPhoto StoredPhoto, oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func scanPhotos(rows pgx.Rows) ([]StoredPhoto, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (StoredPhoto, error) {
		var p StoredPhoto
		var position int16
		err := row.Scan(&p.ID, &p.UserID, &position, &p.StorageKey, &p.Width, &p.Height, &p.CreatedAt)
		p.Position = int(position)
		return p, err
	})
}

const photoColumns = `id, user_id, position, storage_key, width, height, created_at`

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return scanPhotos(rows)
}

// lockUserPhotos serialises photo mutations of one user inside a transaction.
func lockUserPhotos(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID string, photo NewPhoto, maxPhotos int) (StoredPhoto, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= maxPhotos {
		return StoredPhoto{}, ErrRepositoryLimit
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, width, height, size_bytes)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, count, photo.StorageKey, photo.Width, photo.Height, photo.SizeBytes)
	if err != nil {
		return StoredPhoto{}, err
	}
	inserted, err := scanPhotos(rows)
	if err != nil {
		return StoredPhoto{}, err
	}
	return inserted[0], tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, photo NewPhoto) (StoredPhoto, string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return StoredPhoto{}, "", err
	}
	var position int16
	var oldKey string
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key
	`, photoID, userID).Scan(&position, &oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPhoto{}, "", ErrRepositoryNotFound
	}
	if err != nil {
		return StoredPhoto{}, "", err
	}
	// A replaced photo gets a new id so cached URLs of the old image die.
	rows, err := tx.Query(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, width, height, size_bytes)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, position, photo.StorageKey, photo.Width, photo.Height, photo.SizeBytes)
	if err != nil {
		return StoredPhoto{}, "", err
	}
	inserted, err := scanPhotos(rows)
	if err != nil {
		return StoredPhoto{}, "", err
	}
	return inserted[0], oldKey, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return "", err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return "", err
	}
	var position int16
	var key string
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key
	`, photoID, userID).Scan(&position, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if count <= 1 {
		return "", ErrRepositoryLast
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return err
	}
	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*) FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != len(orderedIDs) || total != len(orderedIDs) {
		return ErrRepositoryMismatch
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.dbPool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return key, err
}
