package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLimitReached = errors.New("photo limit reached")
	ErrLastPhoto    = errors.New("cannot delete the last photo")
	ErrNotFound     = errors.New("photo not found")
	ErrBadOrder     = errors.New("order must list every photo exactly once")
)

type Repository interface {
	Add(ctx context.Context, p Photo) (Photo, error)
	List(ctx context.Context, userID string) ([]Photo, error)
	// Remove deletes an owned photo, compacting positions, and returns its storage key.
	Remove(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	// GetForViewer returns a photo only when viewerID may see it (owner, or a
	// non-blocked user for a visible profile / match).
	GetForViewer(ctx context.Context, viewerID, photoID string) (Photo, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Add(ctx context.Context, p Photo) (Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Serialise concurrent uploads of the same user so positions stay dense.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, p.UserID); err != nil {
		return Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotosPerUser {
		return Photo{}, ErrLimitReached
	}
	p.Position = count
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, content_type, size_bytes, width, height, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, p.UserID, p.StorageKey, p.ContentType, p.SizeBytes, p.Width, p.Height, p.Position).Scan(&p.ID)
	if err != nil {
		return Photo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, err
	}
	return p, nil
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, storage_key, content_type, size_bytes, width, height, position
		FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scanPhoto)
}

func scanPhoto(row pgx.CollectableRow) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.ContentType, &p.SizeBytes, &p.Width, &p.Height, &p.Position)
	return p, err
}

func (r *PGRepository) Remove(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return "", err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if count <= 1 {
		return "", ErrLastPhoto
	}
	if _, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1`, photoID); err != nil {
		return "", err
	}
	// Constraint is deferred, so renumbering in one statement is safe.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.n
		FROM (SELECT id, (row_number() OVER (ORDER BY position) - 1)::smallint AS n FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.n
	`, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if len(existing) != len(orderedIDs) {
		return ErrBadOrder
	}
	set := make(map[string]struct{}, len(existing))
	for _, id := range existing {
		set[id] = struct{}{}
	}
	for _, id := range orderedIDs {
		if _, ok := set[id]; !ok {
			return ErrBadOrder
		}
		delete(set, id) // duplicates fail on the second occurrence
	}
	if len(set) != 0 {
		return ErrBadOrder
	}

	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.n::smallint - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, n)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetForViewer(ctx context.Context, viewerID, photoID string) (Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ph.id, ph.user_id, ph.storage_key, ph.content_type, ph.size_bytes, ph.width, ph.height, ph.position
		FROM photos ph
		JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $2
		  AND (ph.user_id = $1 OR (
		        NOT EXISTS (SELECT 1 FROM blocks b
		                    WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id)
		                       OR (b.blocker_id = ph.user_id AND b.blocked_id = $1))
		        AND (pr.is_visible OR EXISTS (
		              SELECT 1 FROM matches m
		              WHERE m.user_low_id = LEAST($1::uuid, ph.user_id) AND m.user_high_id = GREATEST($1::uuid, ph.user_id)))))
	`, viewerID, photoID)
	if err != nil {
		return Photo{}, err
	}
	photos, err := pgx.CollectRows(rows, scanPhoto)
	if err != nil {
		return Photo{}, err
	}
	if len(photos) == 0 {
		return Photo{}, ErrNotFound
	}
	return photos[0], nil
}
