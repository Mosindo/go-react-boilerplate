package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	// Add appends a photo at the next position (ErrLimit when full).
	Add(ctx context.Context, userID, key string, size, width, height int) (stored, error)
	// Replace swaps the image of an owned photo and returns the previous storage key.
	Replace(ctx context.Context, userID, photoID, key string, size, width, height int) (stored, string, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	// Delete removes an owned photo, compacts positions and returns the removed storage key.
	Delete(ctx context.Context, userID, photoID string) (string, error)
	// VisibleKey returns the storage key when viewer may see the photo.
	VisibleKey(ctx context.Context, viewerID, photoID string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Add(ctx context.Context, userID, key string, size, width, height int) (stored, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return stored{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Serialize concurrent uploads of the same user so the count check cannot be raced.
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return stored{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return stored{}, err
	}
	if count >= MaxPhotos {
		return stored{}, ErrLimit
	}
	p := stored{UserID: userID, Position: count, StorageKey: key, Width: width, Height: height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING id
	`, userID, count, key, size, width, height).Scan(&p.ID); err != nil {
		return stored{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, key string, size, width, height int) (stored, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return stored{}, "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var oldKey string
	var position int
	err = tx.QueryRow(ctx, `SELECT storage_key, position FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, "", ErrNotFound
	}
	if err != nil {
		return stored{}, "", err
	}
	// A new row (new id) keeps URLs immutable, which makes aggressive client caching safe.
	if _, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1`, photoID); err != nil {
		return stored{}, "", err
	}
	p := stored{UserID: userID, Position: position, StorageKey: key, Width: width, Height: height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING id
	`, userID, position, key, size, width, height).Scan(&p.ID); err != nil {
		return stored{}, "", err
	}
	return p, oldKey, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 FOR UPDATE`, userID)
	if err != nil {
		return err
	}
	current := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		current[id] = struct{}{}
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if len(ids) != len(current) {
		return ErrBadOrder
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := current[id]; !ok {
			return ErrBadOrder
		}
		if _, dup := seen[id]; dup {
			return ErrBadOrder
		}
		seen[id] = struct{}{}
	}
	// (user_id, position) is DEFERRABLE, so swaps are valid inside the transaction.
	for pos, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $2 WHERE id = $1 AND user_id = $3`, id, pos, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) VisibleKey(ctx context.Context, viewerID, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `
		SELECT ph.storage_key
		FROM photos ph
		JOIN users u ON u.id = ph.user_id AND u.status = 'active'
		JOIN profiles p ON p.user_id = ph.user_id
		WHERE ph.id = $1
		  AND (
		    ph.user_id = $2
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $2 AND b.blocked_id = ph.user_id)
		           OR (b.blocker_id = ph.user_id AND b.blocked_id = $2)
		      )
		      AND (
		        p.discoverable
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.unmatched_at IS NULL
		            AND m.user_low = LEAST($2::uuid, ph.user_id) AND m.user_high = GREATEST($2::uuid, ph.user_id)
		        )
		      )
		    )
		  )
	`, photoID, viewerID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}
