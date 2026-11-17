package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound  = errors.New("photo not found")
	ErrTooManyPhotos  = errors.New("too many photos")
	ErrInvalidOrder   = errors.New("invalid photo order")
	ErrRepoUserAbsent = errors.New("user not found")
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

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, user_id, storage_key, position, width, height
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredPhoto{}
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// lockUser serialises photo mutations of one user (row lock on the profile).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepoUserAbsent
	}
	return err
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
		return StoredPhoto{}, ErrTooManyPhotos
	}
	var p StoredPhoto
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes, mime_type)
		VALUES ($1, $2, $3, $4, $5, $6, 'image/jpeg')
		RETURNING id, user_id, storage_key, position, width, height
	`, userID, storageKey, count, width, height, size).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height)
	if err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
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
		return "", ErrPhotoNotFound
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
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	// Compact positions so the first photo is always the primary one.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = ordered.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position, created_at) AS rn FROM photos WHERE user_id = $1) ordered
		WHERE p.id = ordered.id
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
	rows, err := tx.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if !samePhotoSet(existing, photoIDs) {
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

func (r *PGRepository) GetStorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.dbPool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	return key, err
}

func samePhotoSet(existing, requested []string) bool {
	if len(existing) != len(requested) {
		return false
	}
	set := make(map[string]bool, len(existing))
	for _, id := range existing {
		set[id] = true
	}
	for _, id := range requested {
		if !set[id] {
			return false
		}
		delete(set, id)
	}
	return len(set) == 0
}
