package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("photo not found")
	ErrLimit    = errors.New("photo limit reached")
)

type Repository interface {
	Add(ctx context.Context, userID, key string, size, w, h int) (storedPhoto, error)
	Get(ctx context.Context, id string) (storedPhoto, error)
	GetOwned(ctx context.Context, userID, id string) (storedPhoto, error)
	Replace(ctx context.Context, userID, id, newKey string, size, w, h int) (oldKey string, err error)
	Delete(ctx context.Context, userID, id string) (key string, err error)
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Reorder(ctx context.Context, userID string, ids []string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, size, w, h int) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}
	var count, next int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*), COALESCE(MAX(position) + 1, 0) FROM photos WHERE user_id = $1`, userID).Scan(&count, &next); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotos {
		return storedPhoto{}, ErrLimit
	}
	p := storedPhoto{UserID: userID, StorageKey: key}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6) RETURNING id, position
	`, userID, key, size, w, h, next).Scan(&p.ID, &p.Position); err != nil {
		return storedPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, id string) (storedPhoto, error) {
	var p storedPhoto
	err := r.db.QueryRow(ctx, `SELECT id, user_id, storage_key, position FROM photos WHERE id = $1`, id).
		Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedPhoto{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) GetOwned(ctx context.Context, userID, id string) (storedPhoto, error) {
	p, err := r.Get(ctx, id)
	if err != nil {
		return storedPhoto{}, err
	}
	if p.UserID != userID {
		return storedPhoto{}, ErrNotFound
	}
	return p, nil
}

func (r *PGRepository) Replace(ctx context.Context, userID, id, newKey string, size, w, h int) (string, error) {
	var old string
	err := r.db.QueryRow(ctx, `
		UPDATE photos p SET storage_key = $3, size_bytes = $4, width = $5, height = $6
		FROM (SELECT id, storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE) prev
		WHERE p.id = prev.id RETURNING prev.storage_key
	`, id, userID, newKey, size, w, h).Scan(&old)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return old, err
}

// Delete removes the photo and closes the gap in positions.
func (r *PGRepository) Delete(ctx context.Context, userID, id string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	var pos int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, id, userID).Scan(&key, &pos)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_unique DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, pos); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `SELECT id, user_id, storage_key, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		var p storedPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Reorder applies the exact permutation `ids` (all of the user's photos).
func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, ids).Scan(&count); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if count != len(ids) || total != len(ids) {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_unique DEFERRED`); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
