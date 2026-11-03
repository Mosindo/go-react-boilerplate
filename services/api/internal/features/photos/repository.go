package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLimit    = errors.New("photo limit reached")
	ErrNotFound = errors.New("photo not found")
	ErrOrder    = errors.New("order must list every photo exactly once")
)

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 1))`, userID)
	return err
}

func (r *Repository) Add(ctx context.Context, userID, key string, w, h, size int) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrLimit
	}
	var p Photo
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, position`,
		userID, count, key, w, h, size).Scan(&p.ID, &p.Position); err != nil {
		return Photo{}, err
	}
	p.URL = profiles.PhotoURL(p.ID)
	return p, tx.Commit(ctx)
}

// Replace swaps the file behind a photo slot; the photo gets a new id so
// cached copies of the old image are never served for the new one.
func (r *Repository) Replace(ctx context.Context, userID, photoID, key string, w, h, size int) (Photo, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, "", err
	}
	var position int
	var oldKey string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key`, photoID, userID).Scan(&position, &oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, "", ErrNotFound
	}
	if err != nil {
		return Photo{}, "", err
	}
	p := Photo{Position: position}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, userID, position, key, w, h, size).Scan(&p.ID); err != nil {
		return Photo{}, "", err
	}
	p.URL = profiles.PhotoURL(p.ID)
	return p, oldKey, tx.Commit(ctx)
}

// Delete removes a photo and re-packs positions so the list stays 0..n-1.
func (r *Repository) Delete(ctx context.Context, userID, photoID string) (string, error) {
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
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *Repository) Reorder(ctx context.Context, userID string, ids []string) error {
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
	if count != len(ids) {
		return ErrOrder
	}
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrOrder
		}
	}
	// The (user_id, position) constraint is deferred, so duplicates in ids
	// surface here at commit.
	if err := tx.Commit(ctx); err != nil {
		return ErrOrder
	}
	return nil
}

// KeyForViewer returns the storage key if the viewer may see the photo.
func (r *Repository) KeyForViewer(ctx context.Context, viewerID, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `
		SELECT ph.storage_key
		FROM photos ph
		JOIN profiles p ON p.user_id = ph.user_id
		WHERE ph.id = $2 AND (p.user_id = $1 OR (`+profiles.VisibleToViewer+`))`, viewerID, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}
