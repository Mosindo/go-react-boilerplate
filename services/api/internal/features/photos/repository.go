package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxPhotos = 6

var ErrNotFound = errors.New("photo not found")

type stored struct {
	ID       string
	UserID   string
	Key      string
	Position int
}

type Repository interface {
	Add(ctx context.Context, userID, key string, w, h, size int) (string, error)
	Get(ctx context.Context, photoID string) (stored, error)
	Replace(ctx context.Context, userID, photoID, newKey string, w, h, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (key string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	List(ctx context.Context, userID string) ([]profiles.Photo, error)
	Keys(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// lockUser serialises photo mutations per user (limit checks, renumbering).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, w, h, size int) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return "", err
	}
	if count >= MaxPhotos {
		return "", errMaxPhotos
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, userID, count, key, w, h, size).Scan(&id); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (stored, error) {
	var s stored
	err := r.db.QueryRow(ctx, `SELECT id, user_id, storage_key, position FROM photos WHERE id = $1`, photoID).
		Scan(&s.ID, &s.UserID, &s.Key, &s.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, ErrNotFound
	}
	return s, err
}

// Replace swaps the image in place under a new id so cached copies of the old id never go stale.
func (r *PGRepository) Replace(ctx context.Context, userID, photoID, newKey string, w, h, size int) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var oldKey string
	var pos int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&oldKey, &pos)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, pos, newKey, w, h, size); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
}

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
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = n.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position, created_at) AS rn FROM photos WHERE user_id = $1) n
		WHERE p.id = n.id`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
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
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, ids).Scan(&count); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if count != len(ids) || total != len(ids) {
		return errBadOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]profiles.Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position, created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []profiles.Photo{}
	for rows.Next() {
		var id string
		var pos int
		if err := rows.Scan(&id, &pos); err != nil {
			return nil, err
		}
		out = append(out, profiles.Photo{ID: id, Position: pos, URL: "/photos/" + id, ThumbURL: "/photos/" + id + "?size=thumb"})
	}
	return out, rows.Err()
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
