package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// lockUser serializes photo mutations of one user.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *Repository) add(ctx context.Context, userID string, p processed) (profiles.Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return profiles.Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return profiles.Photo{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return profiles.Photo{}, err
	}
	if count >= MaxPhotosPerUser {
		return profiles.Photo{}, ErrPhotoLimit
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, content, thumb) VALUES ($1, $2, $3, $4) RETURNING id
	`, userID, count, p.Full, p.Thumb).Scan(&id); err != nil {
		return profiles.Photo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return profiles.Photo{}, err
	}
	return profiles.PhotoURLs(id, count), nil
}

// replace swaps the content of a photo while keeping its position. The id changes so cached images
// can be treated as immutable.
func (r *Repository) replace(ctx context.Context, userID, photoID string, p processed) (profiles.Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return profiles.Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return profiles.Photo{}, err
	}

	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return profiles.Photo{}, ErrNotFound
	}
	if err != nil {
		return profiles.Photo{}, err
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, content, thumb) VALUES ($1, $2, $3, $4) RETURNING id
	`, userID, position, p.Full, p.Thumb).Scan(&id); err != nil {
		return profiles.Photo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return profiles.Photo{}, err
	}
	return profiles.PhotoURLs(id, position), nil
}

func (r *Repository) remove(ctx context.Context, userID, photoID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	// Compact positions (the unique constraint is deferred).
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = ranked.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) ranked
		WHERE photos.id = ranked.id
	`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	owned := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		owned[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) != len(owned) {
		return nil, ErrInvalidOrder
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := owned[id]; !ok {
			return nil, ErrInvalidOrder
		}
		if _, dup := seen[id]; dup {
			return nil, ErrInvalidOrder
		}
		seen[id] = struct{}{}
	}

	positions := make([]int32, len(ids))
	for i := range ids {
		positions[i] = int32(i)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = v.pos, updated_at = NOW()
		FROM UNNEST($1::uuid[], $2::int[]) AS v(id, pos)
		WHERE photos.id = v.id AND photos.user_id = $3
	`, ids, positions, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	out := make([]profiles.Photo, len(ids))
	for i, id := range ids {
		out[i] = profiles.PhotoURLs(id, i)
	}
	return out, nil
}

func (r *Repository) ownerOf(ctx context.Context, photoID string) (string, error) {
	var owner string
	err := r.pool.QueryRow(ctx, `SELECT user_id FROM photos WHERE id = $1`, photoID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return owner, err
}

func (r *Repository) content(ctx context.Context, photoID string, thumb bool) ([]byte, error) {
	column := "content"
	if thumb {
		column = "thumb"
	}
	var data []byte
	err := r.pool.QueryRow(ctx, `SELECT `+column+` FROM photos WHERE id = $1`, photoID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}
