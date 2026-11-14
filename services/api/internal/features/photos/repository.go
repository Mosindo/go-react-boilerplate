package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrInvalidOrder = errors.New("invalid photo order")
	ErrLastPhoto    = errors.New("cannot delete last photo")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Insert(ctx context.Context, p Photo) (Photo, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string, keepAtLeastOne bool) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const photoColumns = `id, user_id, storage_key, size_bytes, width, height, position, created_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.SizeBytes, &p.Width, &p.Height, &p.Position, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) { return scanPhoto(row) })
}

// lockUser serialises photo mutations of one user (positions stay dense).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::TEXT, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, p Photo) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, p.UserID); err != nil {
		return Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrLimitReached
	}
	created, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
		RETURNING `+photoColumns, p.UserID, p.StorageKey, p.SizeBytes, p.Width, p.Height, count))
	if err != nil {
		return Photo{}, err
	}
	return created, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (string, error) {
	var oldKey string
	err := r.db.QueryRow(ctx, `
		WITH old AS (SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE)
		UPDATE photos p
		SET storage_key = $3, size_bytes = $4, width = $5, height = $6, created_at = NOW()
		FROM old
		WHERE p.id = $1 AND p.user_id = $2
		RETURNING old.storage_key
	`, photoID, userID, storageKey, size, width, height).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return oldKey, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string, keepAtLeastOne bool) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if keepAtLeastOne {
		var remaining int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&remaining); err != nil {
			return "", err
		}
		if remaining == 0 {
			return "", ErrLastPhoto
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
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
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(photoIDs) {
		return ErrInvalidOrder
	}
	// The (user_id, position) unique constraint is deferred, so positions
	// can be permuted freely inside the transaction.
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::UUID[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != count {
		return ErrInvalidOrder
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

func (r *PGRepository) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
