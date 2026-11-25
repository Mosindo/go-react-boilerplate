package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errRepoNotFound = errors.New("photo not found")
	errRepoLimit    = errors.New("photo limit reached")
	errRepoMismatch = errors.New("photo set mismatch")
)

type Repository interface {
	ListByUser(ctx context.Context, userID string) ([]storedPhoto, error)
	ListByUsers(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, p storedPhoto, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const photoColumns = `id, user_id, storage_key, position, width, height, created_at`

func scanPhoto(row pgx.CollectableRow) (storedPhoto, error) {
	var p storedPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) ListByUser(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPhoto)
}

// ListByUsers loads the photos of many users in a single query (no N+1).
func (r *PGRepository) ListByUsers(ctx context.Context, userIDs []string) (map[string][]storedPhoto, error) {
	result := make(map[string][]storedPhoto, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, scanPhoto)
	if err != nil {
		return nil, err
	}
	for _, p := range list {
		result[p.UserID] = append(result[p.UserID], p)
	}
	return result, nil
}

// lockUser serialises photo mutations of one user inside a transaction.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (storedPhoto, error) {
	var p storedPhoto
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= MaxPhotosPerUser {
			return errRepoLimit
		}
		row, err := tx.Query(ctx, `
			INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes, mime_type)
			VALUES ($1, $2, $3, $4, $5, $6, 'image/jpeg')
			RETURNING `+photoColumns, userID, storageKey, count, width, height, size)
		if err != nil {
			return err
		}
		p, err = pgx.CollectExactlyOneRow(row, scanPhoto)
		return err
	})
	return p, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, storedPhoto, error) {
	var oldKey string
	var p storedPhoto
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errRepoNotFound
			}
			return err
		}
		row, err := tx.Query(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
			WHERE id = $1 AND user_id = $2
			RETURNING `+photoColumns, photoID, userID, storageKey, width, height, size)
		if err != nil {
			return err
		}
		p, err = pgx.CollectExactlyOneRow(row, scanPhoto)
		return err
	})
	return oldKey, p, err
}

// Delete removes a photo and compacts positions so they stay 0..n-1.
func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	var key string
	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var position int
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
		if errors.Is(err, pgx.ErrNoRows) {
			return errRepoNotFound
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position)
		return err
	})
	return key, err
}

// Reorder applies a full ordering; orderedIDs must contain exactly the
// user's photos. The unique (user_id, position) constraint is deferred.
func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var owned, total int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*)
			FROM photos WHERE user_id = $1
		`, userID, orderedIDs).Scan(&owned, &total); err != nil {
			return err
		}
		if owned != len(orderedIDs) || total != len(orderedIDs) {
			return errRepoMismatch
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos p SET position = o.ord - 1
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, orderedIDs)
		return err
	})
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errRepoNotFound
	}
	return key, err
}
