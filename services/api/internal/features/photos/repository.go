package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Get(ctx context.Context, photoID string) (Photo, error)
	// Add appends a photo, or when replaceID is set swaps it in at the same position.
	// It returns the new photo and, for a replacement, the storage key to delete.
	Add(ctx context.Context, p Photo, replaceID string) (Photo, string, error)
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

const photoColumns = `id, user_id, position, storage_key, content_type, size_bytes, width, height`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.ContentType, &p.SizeBytes, &p.Width, &p.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Photo
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (Photo, error) {
	return scanPhoto(r.pool.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, photoID))
}

// lockUser serialises photo mutations of one user for the transaction.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "photos:"+userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, p Photo, replaceID string) (Photo, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, p.UserID); err != nil {
		return Photo{}, "", err
	}

	position := 0
	oldKey := ""
	if replaceID != "" {
		err := tx.QueryRow(ctx, `
			DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key
		`, replaceID, p.UserID).Scan(&position, &oldKey)
		if errors.Is(err, pgx.ErrNoRows) {
			return Photo{}, "", ErrNotFound
		}
		if err != nil {
			return Photo{}, "", err
		}
	} else {
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&position); err != nil {
			return Photo{}, "", err
		}
		if position >= MaxPhotosPerUser {
			return Photo{}, "", ErrLimitReached
		}
	}

	created, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, content_type, size_bytes, width, height)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+photoColumns,
		p.UserID, position, p.StorageKey, p.ContentType, p.SizeBytes, p.Width, p.Height))
	if err != nil {
		return Photo{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, "", err
	}
	return created, oldKey, nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var position int
	var key string
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key
	`, photoID, userID).Scan(&position, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	// Keep positions contiguous (0..n-1) so position 0 is always the primary photo.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}

	var count int
	var matched int
	if err := tx.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE id = ANY($2::uuid[])) FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&count, &matched); err != nil {
		return err
	}
	if count != len(orderedIDs) || matched != count {
		return ErrInvalidOrder
	}

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p
		SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
