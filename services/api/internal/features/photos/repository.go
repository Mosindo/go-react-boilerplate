package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("repository: not found")
	ErrRepositoryFull     = errors.New("repository: photo limit reached")
	ErrRepositoryMismatch = errors.New("repository: ids do not match user's photos")
)

type Repository interface {
	Add(ctx context.Context, userID, key string, w, h, size, maxPhotos int) (stored, error)
	Get(ctx context.Context, id string) (stored, error)
	Replace(ctx context.Context, userID, id, newKey string, w, h, size int) (oldKey string, err error)
	Delete(ctx context.Context, userID, id string) (key string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	List(ctx context.Context, userID string) ([]Photo, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Add(ctx context.Context, userID, key string, w, h, size, maxPhotos int) (stored, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return stored{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialise concurrent uploads of the same user so the limit and positions stay consistent.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return stored{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return stored{}, err
	}
	if count >= maxPhotos {
		return stored{}, ErrRepositoryFull
	}
	s := stored{UserID: userID, StorageKey: key, Position: count}
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, userID, key, count, w, h, size).Scan(&s.ID)
	if err != nil {
		return stored{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, id string) (stored, error) {
	var s stored
	err := r.db.QueryRow(ctx, `SELECT id, user_id, storage_key, position FROM photos WHERE id = $1`, id).
		Scan(&s.ID, &s.UserID, &s.StorageKey, &s.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, ErrRepositoryNotFound
	}
	return s, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, id, newKey string, w, h, size int) (string, error) {
	var old string
	err := r.db.QueryRow(ctx, `
		UPDATE photos p SET storage_key = $3, width = $4, height = $5, size_bytes = $6
		FROM (SELECT id, storage_key FROM photos WHERE id = $2 AND user_id = $1 FOR UPDATE) prev
		WHERE p.id = prev.id
		RETURNING prev.storage_key`, userID, id, newKey, w, h, size).Scan(&old)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	return old, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, id string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, id, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn - 1`, userID); err != nil {
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
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])`, userID, ids).Scan(&count); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if count != len(ids) || total != len(ids) {
		return ErrRepositoryMismatch
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position, substr(storage_key, 1, 8) FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		var v string
		if err := rows.Scan(&p.ID, &p.Position, &v); err != nil {
			return nil, err
		}
		p.URL = "/photos/" + p.ID + "/file?v=" + v
		out = append(out, p)
	}
	return out, rows.Err()
}
