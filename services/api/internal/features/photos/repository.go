package photos

import (
	"context"
	"errors"

	"example.com/api/internal/platform/access"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Add(ctx context.Context, userID, key string, w, h, size int) (StoredPhoto, error)
	Remove(ctx context.Context, userID, photoID string) (key string, err error)
	Replace(ctx context.Context, userID, photoID, newKey string, w, h, size int) (oldKey string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	KeyForViewer(ctx context.Context, viewerID, photoID string) (string, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const cols = `id::text, storage_key, position, width, height, size_bytes`

func scanPhoto(row pgx.Row) (StoredPhoto, error) {
	var p StoredPhoto
	err := row.Scan(&p.ID, &p.StorageKey, &p.Position, &p.Width, &p.Height, &p.SizeBytes)
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	return listTx(ctx, r.pool, userID)
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listTx(ctx context.Context, q queryer, userID string) ([]StoredPhoto, error) {
	rows, err := q.Query(ctx, `SELECT `+cols+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StoredPhoto{}
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// lockUser serialises photo mutations per user so position math is race-free.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, w, h, size int) (StoredPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrLimitReached
	}
	p, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+cols, userID, key, count, w, h, size))
	if err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Remove(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, newKey string, w, h, size int) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var old string
	err = tx.QueryRow(ctx, `
		UPDATE photos n SET storage_key = $3, width = $4, height = $5, size_bytes = $6
		FROM photos o WHERE n.id = o.id AND n.id = $1 AND n.user_id = $2
		RETURNING o.storage_key`, photoID, userID, newKey, w, h, size).Scan(&old)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return old, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	existing, err := listTx(ctx, tx, userID)
	if err != nil {
		return err
	}
	if len(existing) != len(ids) {
		return ErrOrderMismatch
	}
	have := map[string]bool{}
	for _, p := range existing {
		have[p.ID] = true
	}
	for _, id := range ids {
		if !have[id] {
			return ErrOrderMismatch
		}
		delete(have, id)
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	for pos, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, pos); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) KeyForViewer(ctx context.Context, viewerID, photoID string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `
		SELECT ph.storage_key FROM photos ph
		WHERE ph.id = $2 AND `+access.CanView("$1", "ph.user_id"), viewerID, photoID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return key, nil
}
