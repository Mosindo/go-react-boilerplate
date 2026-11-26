package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoLimit    = errors.New("photo limit reached")
	errRepoNotFound = errors.New("photo not found")
	errRepoMismatch = errors.New("photo set mismatch")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID string, p storedPhoto, sizeBytes int) error
	Replace(ctx context.Context, userID, photoID string, p storedPhoto, sizeBytes int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (key string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	return listPhotos(ctx, r.db, userID)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listPhotos(ctx context.Context, q querier, userID string) ([]storedPhoto, error) {
	rows, err := q.Query(ctx, `
		SELECT id, storage_key, width, height, position FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (storedPhoto, error) {
		var p storedPhoto
		err := row.Scan(&p.ID, &p.StorageKey, &p.Width, &p.Height, &p.Position)
		return p, err
	})
}

// lockUser serialises photo mutations of one user so positions and the limit stay consistent.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID string, p storedPhoto, sizeBytes int) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= MaxPhotos {
			return errRepoLimit
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO photos (user_id, storage_key, width, height, size_bytes, position)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, p.StorageKey, p.Width, p.Height, sizeBytes, count)
		return err
	})
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p storedPhoto, sizeBytes int) (string, error) {
	var oldKey string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&oldKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
			WHERE id = $1 AND user_id = $2
		`, photoID, userID, p.StorageKey, p.Width, p.Height, sizeBytes)
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
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoNotFound
		}
		if err != nil {
			return err
		}
		// Compact positions so the first remaining photo becomes the main one.
		_, err = tx.Exec(ctx, `
			UPDATE photos p SET position = o.new_position
			FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::smallint AS new_position FROM photos WHERE user_id = $1) o
			WHERE p.id = o.id AND p.position <> o.new_position
		`, userID)
		return err
	})
	return key, err
}

// Reorder applies a full permutation of the user's photos; position 0 is the main photo.
func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		current, err := listPhotos(ctx, tx, userID)
		if err != nil {
			return err
		}
		if len(current) != len(ids) {
			return errRepoMismatch
		}
		owned := make(map[string]struct{}, len(current))
		for _, p := range current {
			owned[p.ID] = struct{}{}
		}
		for _, id := range ids {
			if _, ok := owned[id]; !ok {
				return errRepoMismatch
			}
			delete(owned, id) // also rejects duplicates
		}
		// The (user_id, position) unique constraint is deferred until commit.
		_, err = tx.Exec(ctx, `
			UPDATE photos p SET position = (o.ord - 1)::smallint
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, ids)
		return err
	})
}
