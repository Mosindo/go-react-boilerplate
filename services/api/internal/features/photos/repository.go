package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("photo not found")
	ErrLimitReached    = errors.New("photo limit reached")
	ErrProfileRequired = errors.New("profile required")
	ErrInvalidOrder    = errors.New("order must list each of your photos exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, photo Photo, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const photoColumns = `id, user_id, storage_key, width, height, size_bytes, position, created_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes, &p.Position, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) { return scanPhoto(row) })
}

// lockProfile serializes photo mutations per user so the 6-photo limit and
// positions stay consistent under concurrent uploads.
func lockProfile(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProfileRequired
	}
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error) {
	var photo Photo
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockProfile(ctx, tx, userID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= MaxPhotosPerUser {
			return ErrLimitReached
		}
		var err error
		photo, err = scanPhoto(tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, storage_key, width, height, size_bytes, position)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING `+photoColumns, userID, storageKey, width, height, size, count))
		return err
	})
	return photo, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, Photo, error) {
	var oldKey string
	var photo Photo
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		photo, err = scanPhoto(tx.QueryRow(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
			WHERE id = $1 AND user_id = $2
			RETURNING `+photoColumns, photoID, userID, storageKey, width, height, size))
		return err
	})
	return oldKey, photo, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	var key string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockProfile(ctx, tx, userID); err != nil {
			return err
		}
		var position int
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Close the gap so positions stay contiguous (0 is the main photo).
		_, err = tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position)
		return err
	})
	return key, err
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockProfile(ctx, tx, userID); err != nil {
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
		if len(existing) != len(photoIDs) {
			return ErrInvalidOrder
		}
		owned := make(map[string]bool, len(existing))
		for _, id := range existing {
			owned[id] = true
		}
		seen := map[string]bool{}
		for _, id := range photoIDs {
			if !owned[id] || seen[id] {
				return ErrInvalidOrder
			}
			seen[id] = true
		}
		// The unique (user_id, position) constraint is deferred, so the
		// permutation can be applied in one statement.
		_, err = tx.Exec(ctx, `
			UPDATE photos p SET position = o.ord - 1
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
		return "", ErrNotFound
	}
	return key, err
}
