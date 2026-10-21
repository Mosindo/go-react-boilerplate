package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("photos: not found")
	ErrLimit    = errors.New("photos: limit reached")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID, id string, p storedPhoto, width, height, size int) (storedPhoto, error)
	Delete(ctx context.Context, userID, photoID string) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID string, key, thumbKey string, width, height, size int) (old storedPhoto, updated storedPhoto, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	Keys(ctx context.Context, photoID string) (key, thumbKey string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	return listPhotos(ctx, r.db, userID)
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listPhotos(ctx context.Context, q queryer, userID string) ([]storedPhoto, error) {
	rows, err := q.Query(ctx, `SELECT id, position, storage_key, thumb_key FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []storedPhoto{}
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.Position, &p.Key, &p.ThumbKey); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Insert appends a photo at the next free position. A per-user advisory lock makes the count check
// and the insert atomic.
func (r *PGRepository) Insert(ctx context.Context, userID, id string, p storedPhoto, width, height, size int) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		return storedPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotosPerUser {
		return storedPhoto{}, ErrLimit
	}
	p.ID, p.Position = id, count
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (id, user_id, position, storage_key, thumb_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, userID, count, p.Key, p.ThumbKey, width, height, size); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

// Delete removes a photo and closes the position gap.
func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		return storedPhoto{}, err
	}
	var p storedPhoto
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING id, position, storage_key, thumb_key
	`, photoID, userID).Scan(&p.ID, &p.Position, &p.Key, &p.ThumbKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedPhoto{}, ErrNotFound
	}
	if err != nil {
		return storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, p.Position); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, key, thumbKey string, width, height, size int) (storedPhoto, storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var old storedPhoto
	err = tx.QueryRow(ctx, `SELECT id, position, storage_key, thumb_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`,
		photoID, userID).Scan(&old.ID, &old.Position, &old.Key, &old.ThumbKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedPhoto{}, storedPhoto{}, ErrNotFound
	}
	if err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET storage_key = $3, thumb_key = $4, width = $5, height = $6, size_bytes = $7 WHERE id = $1 AND user_id = $2
	`, photoID, userID, key, thumbKey, width, height, size); err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	return old, storedPhoto{ID: old.ID, Position: old.Position, Key: key, ThumbKey: thumbKey}, nil
}

// Reorder applies a new order; orderedIDs must be exactly the caller's photos.
func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		return err
	}
	current, err := listPhotos(ctx, tx, userID)
	if err != nil {
		return err
	}
	if len(current) != len(orderedIDs) {
		return ErrNotFound
	}
	owned := map[string]bool{}
	for _, p := range current {
		owned[p.ID] = true
	}
	for i, id := range orderedIDs {
		if !owned[id] {
			return ErrNotFound
		}
		delete(owned, id)
		// The (user_id, position) unique constraint is deferred, so swaps are safe inside the transaction.
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Keys(ctx context.Context, photoID string) (string, string, error) {
	var key, thumb string
	err := r.db.QueryRow(ctx, `SELECT storage_key, thumb_key FROM photos WHERE id = $1`, photoID).Scan(&key, &thumb)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return key, thumb, err
}
