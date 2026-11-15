package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Add(ctx context.Context, userID string, p Photo) (Photo, error)
	Replace(ctx context.Context, userID, photoID string, p Photo) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, photoIDs []string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// lockUser serialises photo mutations of one user (row lock on the profile).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPhotoNotFound
	}
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	return listPhotos(ctx, r.db, userID)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listPhotos(ctx context.Context, q querier, userID string) ([]Photo, error) {
	rows, err := q.Query(ctx, `
		SELECT id, user_id, storage_key, width, height, size_bytes, position, created_at
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) {
		var p Photo
		var pos int16
		err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes, &pos, &p.CreatedAt)
		p.Position = int(pos)
		return p, err
	})
}

func (r *PGRepository) Add(ctx context.Context, userID string, p Photo) (Photo, error) {
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
		return Photo{}, ErrTooManyPhotos
	}
	var pos int16
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, width, height, size_bytes, position)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
		RETURNING id, user_id, storage_key, width, height, size_bytes, position, created_at
	`, userID, p.StorageKey, p.Width, p.Height, p.SizeBytes, count).Scan(
		&p.ID, &p.UserID, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes, &pos, &p.CreatedAt)
	if err != nil {
		return Photo{}, err
	}
	p.Position = int(pos)
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p Photo) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&oldKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, photoID, userID, p.StorageKey, p.Width, p.Height, p.SizeBytes); err != nil {
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPhotoNotFound
		}
		return "", err
	}
	// Compact positions so the first remaining photo becomes the primary one.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) o
		WHERE p.id = o.id
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, photoIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	current, err := listPhotos(ctx, tx, userID)
	if err != nil {
		return err
	}
	if !SameSet(current, photoIDs) {
		return ErrInvalidOrder
	}
	// The (user_id, position) unique constraint is deferred to commit time.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, photoIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SameSet reports whether ids is exactly a permutation of the photos' ids.
func SameSet(photos []Photo, ids []string) bool {
	if len(photos) != len(ids) {
		return false
	}
	want := make(map[string]struct{}, len(photos))
	for _, p := range photos {
		want[p.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := want[id]; !ok {
			return false
		}
		delete(want, id)
	}
	return len(want) == 0
}
