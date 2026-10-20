package photos

import (
	"context"
	"errors"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Add(ctx context.Context, userID string, p NewPhoto) (Photo, error)
	List(ctx context.Context, userID string) ([]Photo, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Get(ctx context.Context, photoID string) (StoredPhoto, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

var errNotFound = platformerrors.NotFound("photo not found")

// lockUser serializes photo mutations of one user (position bookkeeping).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Add(ctx context.Context, userID string, p NewPhoto) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotosPerUser {
		return Photo{}, platformerrors.Conflict("you can upload at most 6 photos")
	}

	photo := Photo{Position: count}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text
	`, userID, count, p.StorageKey, p.Width, p.Height, p.SizeBytes).Scan(&photo.ID); err != nil {
		return Photo{}, err
	}
	photo.URL = urlFor(photo.ID)
	return photo, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position); err != nil {
			return nil, err
		}
		p.URL = urlFor(p.ID)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
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
	existing := map[string]struct{}{}
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

	if len(ids) != len(existing) {
		return platformerrors.BadRequest("photoIds must contain exactly your photos")
	}
	seen := map[string]struct{}{}
	positions := make([]int32, len(ids))
	for i, id := range ids {
		if _, ok := existing[id]; !ok {
			return platformerrors.BadRequest("photoIds must contain exactly your photos")
		}
		if _, dup := seen[id]; dup {
			return platformerrors.BadRequest("photoIds must not contain duplicates")
		}
		seen[id] = struct{}{}
		positions[i] = int32(i)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = x.pos
		FROM unnest($1::uuid[], $2::int[]) AS x(id, pos)
		WHERE photos.id = x.id AND photos.user_id = $3
	`, ids, positions, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Delete removes the photo, compacts the remaining positions and returns the storage key.
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
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = r.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE photos.id = r.id
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (StoredPhoto, error) {
	var p StoredPhoto
	err := r.db.QueryRow(ctx, `SELECT id::text, user_id::text, storage_key FROM photos WHERE id = $1`, photoID).
		Scan(&p.ID, &p.OwnerID, &p.StorageKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPhoto{}, errNotFound
	}
	return p, err
}

func urlFor(id string) string { return "/photos/" + id + "/file" }
