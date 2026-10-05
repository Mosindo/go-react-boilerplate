package photos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrBadOrder     = errors.New("photo ids must list every photo exactly once")
)

type storedPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
	CreatedAt  time.Time
}

type Repository interface {
	List(ctx context.Context, userID string) ([]storedPhoto, error)
	Insert(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error)
	Get(ctx context.Context, id string) (storedPhoto, error)
	Replace(ctx context.Context, userID, id, key string, width, height, size int) (oldKey string, p storedPhoto, err error)
	Delete(ctx context.Context, userID, id string) (key string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	Keys(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const photoColumns = `id, user_id, storage_key, position, width, height, created_at`

func scanPhoto(row pgx.Row) (storedPhoto, error) {
	var p storedPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedPhoto{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]storedPhoto, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// withUserLock serialises photo mutations of one user so position and limit checks cannot race.
func (r *PGRepository) withUserLock(ctx context.Context, userID string, fn func(tx pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 42))`, userID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Insert(ctx context.Context, userID, key string, width, height, size int) (storedPhoto, error) {
	var out storedPhoto
	err := r.withUserLock(ctx, userID, func(tx pgx.Tx) error {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count >= MaxPhotosPerUser {
			return ErrLimitReached
		}
		p, err := scanPhoto(tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, storage_key, position, content_type, width, height, size_bytes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING `+photoColumns, userID, key, count, storedMime, width, height, size))
		out = p
		return err
	})
	return out, err
}

func (r *PGRepository) Get(ctx context.Context, id string) (storedPhoto, error) {
	return scanPhoto(r.db.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, id))
}

func (r *PGRepository) Replace(ctx context.Context, userID, id, key string, width, height, size int) (string, storedPhoto, error) {
	var oldKey string
	var out storedPhoto
	err := r.withUserLock(ctx, userID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, id, userID).Scan(&oldKey); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		p, err := scanPhoto(tx.QueryRow(ctx, `
			UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6
			WHERE id = $1 AND user_id = $2
			RETURNING `+photoColumns, id, userID, key, width, height, size))
		out = p
		return err
	})
	return oldKey, out, err
}

// Delete removes a photo and compacts the remaining positions so the first photo stays the primary one.
func (r *PGRepository) Delete(ctx context.Context, userID, id string) (string, error) {
	var key string
	err := r.withUserLock(ctx, userID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, id, userID).Scan(&key); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE photos p SET position = ranked.rn
			FROM (SELECT id, (row_number() OVER (ORDER BY position) - 1)::smallint AS rn FROM photos WHERE user_id = $1) ranked
			WHERE p.id = ranked.id AND p.position <> ranked.rn
		`, userID)
		return err
	})
	return key, err
}

// Reorder sets positions from the given list (first id becomes the primary photo).
func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	return r.withUserLock(ctx, userID, func(tx pgx.Tx) error {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return ErrBadOrder
		}
		tag, err := tx.Exec(ctx, `
			UPDATE photos p SET position = (o.ord - 1)::smallint
			FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
			WHERE p.id = o.id AND p.user_id = $1
		`, userID, ids)
		if err != nil {
			return err
		}
		if int(tag.RowsAffected()) != len(ids) {
			return ErrBadOrder // unknown id or an id owned by someone else
		}
		return nil
	})
}

func (r *PGRepository) Keys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
