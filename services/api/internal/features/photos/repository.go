package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotFound = errors.New("photo not found")
	ErrRepoLimit    = errors.New("photo limit reached")
	ErrRepoMismatch = errors.New("photo set mismatch")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Insert(ctx context.Context, userID, key string, width, height, size int) (Photo, error)
	Replace(ctx context.Context, userID, photoID, key string, width, height, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (key string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func lockUserPhotos(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, position, storage_key, width, height FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) {
		var p Photo
		var pos int16
		err := row.Scan(&p.ID, &pos, &p.StorageKey, &p.Width, &p.Height)
		p.Position = int(pos)
		return p, err
	})
}

func (r *PGRepository) Insert(ctx context.Context, userID, key string, width, height, size int) (Photo, error) {
	var p Photo
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := lockUserPhotos(ctx, tx, userID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= MaxPhotos {
			return ErrRepoLimit
		}
		var pos int16
		err := tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, position, storage_key, width, height
		`, userID, count, key, width, height, size).Scan(&p.ID, &pos, &p.StorageKey, &p.Width, &p.Height)
		p.Position = int(pos)
		return err
	})
	return p, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, key string, width, height, size int) (string, error) {
	var old string
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&old); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRepoNotFound
			}
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
			WHERE id = $1 AND user_id = $2
		`, photoID, userID, key, width, height, size)
		return err
	})
	return old, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	var key string
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := lockUserPhotos(ctx, tx, userID); err != nil {
			return err
		}
		var pos int16
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &pos)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepoNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, pos)
		return err
	})
	return key, err
}

// Reorder requires the exact set of the user's photo ids; index 0 becomes the
// main photo.
func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := lockUserPhotos(ctx, tx, userID); err != nil {
			return err
		}
		var matching, total int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*) FROM photos WHERE user_id = $1
		`, userID, ids).Scan(&matching, &total); err != nil {
			return err
		}
		if matching != len(ids) || total != len(ids) {
			return ErrRepoMismatch
		}
		if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos p SET position = o.ord - 1
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, ids)
		return err
	})
}
