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
)

const MaxPhotos = 6

type Row struct {
	ID         string
	StorageKey string
	Position   int
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, userID string) ([]Row, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, storage_key, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.ID, &row.StorageKey, &row.Position); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) KeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Add appends a photo at the next free position, atomically enforcing the per-user cap.
func (r *Repository) Add(ctx context.Context, userID, key string, width, height, size int) (Row, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Row{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID); err != nil {
		return Row{}, err
	}
	var next int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM photos WHERE user_id = $1`, userID).Scan(&next); err != nil {
		return Row{}, err
	}
	if next >= MaxPhotos {
		return Row{}, ErrRepoLimit
	}
	row := Row{StorageKey: key, Position: next}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, key, next, width, height, size).Scan(&row.ID); err != nil {
		return Row{}, err
	}
	return row, tx.Commit(ctx)
}

// Replace swaps the stored image of an owned photo, keeping its position, and returns the old key.
func (r *Repository) Replace(ctx context.Context, userID, photoID, newKey string, width, height, size int) (Row, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Row{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID); err != nil {
		return Row{}, "", err
	}
	var oldKey string
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&oldKey, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, "", ErrRepoNotFound
	}
	if err != nil {
		return Row{}, "", err
	}
	row := Row{StorageKey: newKey, Position: position}
	// A fresh id keeps client caches (immutable URLs) correct.
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, newKey, position, width, height, size).Scan(&row.ID); err != nil {
		return Row{}, "", err
	}
	return row, oldKey, tx.Commit(ctx)
}

// Delete removes an owned photo and compacts positions so the main photo is always position 0.
func (r *Repository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	if err != nil {
		return "", err
	}
	if err := compact(ctx, tx, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func compact(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE photos p SET position = ranked.new_pos
		FROM (
			SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::int AS new_pos
			FROM photos WHERE user_id = $1
		) ranked
		WHERE p.id = ranked.id AND p.position <> ranked.new_pos
	`, userID)
	return err
}

// Reorder applies a complete ordering of the user's photos.
func (r *Repository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID); err != nil {
		return err
	}
	var owned int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, ids).Scan(&owned); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if owned != len(ids) || total != len(ids) {
		return ErrRepoNotFound
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) KeyByID(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return key, err
}
