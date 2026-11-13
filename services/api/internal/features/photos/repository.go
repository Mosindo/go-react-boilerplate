package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound   = errors.New("photo not found")
	ErrTooManyPhotos   = errors.New("a profile can have at most 6 photos")
	ErrInvalidOrdering = errors.New("photo order must list each of your photos exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, photo storedPhoto, err error)
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

func scanPhotos(rows pgx.Rows) ([]storedPhoto, error) {
	defer rows.Close()
	out := make([]storedPhoto, 0, MaxPhotosPerUser)
	for rows.Next() {
		var p storedPhoto
		var position int16
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &position, &p.Width, &p.Height, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Position = int(position)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, storage_key, position, width, height, created_at
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return scanPhotos(rows)
}

// lockUser serializes photo mutations of one user via the profile row.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPhotoNotFound
	}
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotosPerUser {
		return storedPhoto{}, ErrTooManyPhotos
	}

	var p storedPhoto
	var position int16
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, storage_key, position, width, height, created_at
	`, userID, storageKey, count, width, height, size).Scan(&p.ID, &p.UserID, &p.StorageKey, &position, &p.Width, &p.Height, &p.CreatedAt)
	if err != nil {
		return storedPhoto{}, err
	}
	p.Position = int(position)
	if _, err := tx.Exec(ctx, `SELECT refresh_profile_completeness($1)`, userID); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", storedPhoto{}, err
	}

	var oldKey string
	if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&oldKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", storedPhoto{}, ErrPhotoNotFound
		}
		return "", storedPhoto{}, err
	}

	var p storedPhoto
	var position int16
	err = tx.QueryRow(ctx, `
		UPDATE photos
		SET id = gen_random_uuid(), storage_key = $3, width = $4, height = $5, size_bytes = $6, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, storage_key, position, width, height, created_at
	`, photoID, userID, storageKey, width, height, size).Scan(&p.ID, &p.UserID, &p.StorageKey, &position, &p.Width, &p.Height, &p.CreatedAt)
	if err != nil {
		return "", storedPhoto{}, err
	}
	p.Position = int(position)
	return oldKey, p, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var key string
	var position int16
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
		UPDATE photos SET position = position - 1, updated_at = NOW()
		WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SELECT refresh_profile_completeness($1)`, userID); err != nil {
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
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(existing) != len(photoIDs) {
		return ErrInvalidOrdering
	}
	seen := make(map[string]struct{}, len(photoIDs))
	for _, id := range photoIDs {
		if _, ok := existing[id]; !ok {
			return ErrInvalidOrdering
		}
		if _, dup := seen[id]; dup {
			return ErrInvalidOrdering
		}
		seen[id] = struct{}{}
	}

	// The unique (user_id, position) constraint is deferred, so positions can
	// be permuted freely inside the transaction.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p
		SET position = o.ord - 1, updated_at = NOW()
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	return key, nil
}
