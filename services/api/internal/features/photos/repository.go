package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound   = errors.New("photo not found")
	ErrTooManyPhotos   = errors.New("too many photos")
	ErrProfileRequired = errors.New("profile required")
	ErrInvalidOrder    = errors.New("invalid photo order")
)

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]Photo, error)
	Insert(ctx context.Context, p Photo) (Photo, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	StorageKeyByID(ctx context.Context, photoID string) (string, error)
	StorageKeysByUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const photoColumns = `id, user_id, position, storage_key, width, height, size_bytes, created_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) ListByUser(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Photo, 0, MaxPhotosPerUser)
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// lockUserPhotos serialises photo mutations of one user so the max count
// and contiguous positions can never be violated by concurrent requests.
func lockUserPhotos(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, p Photo) (Photo, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockUserPhotos(ctx, tx, p.UserID); err != nil {
		return Photo{}, err
	}
	var hasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, p.UserID).Scan(&hasProfile); err != nil {
		return Photo{}, err
	}
	if !hasProfile {
		return Photo{}, ErrProfileRequired
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotosPerUser {
		return Photo{}, ErrTooManyPhotos
	}

	created, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+photoColumns, p.UserID, count, p.StorageKey, p.Width, p.Height, p.SizeBytes))
	if err != nil {
		return Photo{}, err
	}
	return created, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos
		SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, photoID, userID, storageKey, width, height, size); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
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
	var key string
	var position int
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2
		RETURNING storage_key, position
	`, photoID, userID).Scan(&key, &position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	// Keep positions contiguous so position 0 is always the main photo.
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1
		WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

// Reorder requires the exact set of the user's photo ids, in the new order.
func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockUserPhotos(ctx, tx, userID); err != nil {
		return err
	}
	var count, matching int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE id = ANY($2::uuid[]))
		FROM photos WHERE user_id = $1
	`, userID, photoIDs).Scan(&count, &matching); err != nil {
		return err
	}
	if count != len(photoIDs) || matching != len(photoIDs) {
		return ErrInvalidOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p
		SET position = o.idx - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, idx)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKeyByID(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.dbPool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	return key, nil
}

func (r *PGRepository) StorageKeysByUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0, MaxPhotosPerUser)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
