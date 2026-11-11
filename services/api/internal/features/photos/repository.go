package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimitReached  = errors.New("photo limit reached")
	ErrInvalidOrder  = errors.New("invalid photo order")
	ErrNoProfile     = errors.New("profile required")
	maxPhotosPerUser = 6
)

type Repository interface {
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
	ExistsByKey(ctx context.Context, storageKey string) (bool, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// lockUser serializes photo mutations of a single user inside a transaction.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (StoredPhoto, error) {
	var p StoredPhoto
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var hasProfile bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, userID).Scan(&hasProfile); err != nil {
			return err
		}
		if !hasProfile {
			return ErrNoProfile
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= maxPhotosPerUser {
			return ErrLimitReached
		}
		return tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, user_id, storage_key, position
		`, userID, storageKey, count, width, height, size).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	})
	return p, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, error) {
	var oldKey string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&oldKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
			WHERE id = $1 AND user_id = $2
		`, photoID, userID, storageKey, width, height, size)
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
			return ErrNotFound
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
		var total, matched int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*), COUNT(*) FILTER (WHERE id = ANY($2)) FROM photos WHERE user_id = $1
		`, userID, photoIDs).Scan(&total, &matched); err != nil {
			return err
		}
		if total != len(photoIDs) || matched != len(photoIDs) {
			return ErrInvalidOrder
		}
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos p SET position = o.ord - 1
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, photoIDs)
		return err
	})
}

func (r *PGRepository) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (r *PGRepository) ExistsByKey(ctx context.Context, storageKey string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE storage_key = $1)`, storageKey).Scan(&ok)
	return ok, err
}
