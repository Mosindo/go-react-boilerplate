package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// lockUser serialises photo mutations per user so counts and positions stay consistent.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *Repository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position); err != nil {
			return nil, err
		}
		p.URL, p.ThumbURL = "/photos/"+p.ID+"/image", "/photos/"+p.ID+"/thumb"
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Add(ctx context.Context, userID, fullKey, thumbKey string, w, h, size int) (Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var count, pos int
	if err := tx.QueryRow(ctx, `SELECT count(*), COALESCE(max(position) + 1, 0) FROM photos WHERE user_id = $1`, userID).Scan(&count, &pos); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrLimitReached
	}
	p := Photo{Position: pos}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, thumb_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, userID, fullKey, thumbKey, pos, w, h, size).Scan(&p.ID); err != nil {
		return Photo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, err
	}
	p.URL, p.ThumbURL = "/photos/"+p.ID+"/image", "/photos/"+p.ID+"/thumb"
	return p, nil
}

// Replace swaps the file behind a photo slot. It creates a new photo id so that immutable
// client caches keyed by URL never serve the old picture. It returns the removed keys.
func (r *Repository) Replace(ctx context.Context, userID, photoID, fullKey, thumbKey string, w, h, size int) (Photo, storedPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, storedPhoto{}, err
	}
	var old storedPhoto
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING id, user_id, storage_key, thumb_key, position`, photoID, userID).
		Scan(&old.ID, &old.UserID, &old.StorageKey, &old.ThumbKey, &old.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Photo{}, storedPhoto{}, ErrNotFound
		}
		return Photo{}, storedPhoto{}, err
	}
	p := Photo{Position: old.Position}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, thumb_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`, userID, fullKey, thumbKey, old.Position, w, h, size).Scan(&p.ID); err != nil {
		return Photo{}, storedPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, storedPhoto{}, err
	}
	p.URL, p.ThumbURL = "/photos/"+p.ID+"/image", "/photos/"+p.ID+"/thumb"
	return p, old, nil
}

// Delete removes a photo and closes the gap so positions stay 0..n-1 (0 is the primary photo).
func (r *Repository) Delete(ctx context.Context, userID, photoID string) (storedPhoto, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}
	var old storedPhoto
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING id, user_id, storage_key, thumb_key, position`, photoID, userID).
		Scan(&old.ID, &old.UserID, &old.StorageKey, &old.ThumbKey, &old.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return storedPhoto{}, ErrNotFound
		}
		return storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, old.Position); err != nil {
		return storedPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return storedPhoto{}, err
	}
	return old, nil
}

// Reorder sets positions from the given id order (first id becomes the primary photo).
func (r *Repository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrInvalidOrder
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.pos - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, pos)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrInvalidOrder
		}
		return err
	}
	if int(tag.RowsAffected()) != len(ids) {
		return ErrInvalidOrder
	}
	return tx.Commit(ctx)
}

// Viewable returns the storage keys of a photo when viewerID may see it: the owner always can;
// others only when the owner is a visible profile and no block exists in either direction.
func (r *Repository) Viewable(ctx context.Context, viewerID, photoID string) (storedPhoto, error) {
	var p storedPhoto
	err := r.pool.QueryRow(ctx, `
		SELECT ph.id, ph.user_id, ph.storage_key, ph.thumb_key, ph.position
		FROM photos ph
		WHERE ph.id = $2
		  AND (
		    ph.user_id = $1
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id)
		           OR (b.blocker_id = ph.user_id AND b.blocked_id = $1)
		      )
		      AND (
		        EXISTS (SELECT 1 FROM profiles pr WHERE pr.user_id = ph.user_id AND pr.is_visible)
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.user_a_id = LEAST($1::uuid, ph.user_id) AND m.user_b_id = GREATEST($1::uuid, ph.user_id)
		        )
		      )
		    )
		  )
	`, viewerID, photoID).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.ThumbKey, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return storedPhoto{}, ErrNotFound
		}
		return storedPhoto{}, err
	}
	return p, nil
}

func (r *Repository) KeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT storage_key, thumb_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil {
			return nil, err
		}
		keys = append(keys, a, b)
	}
	return keys, rows.Err()
}
