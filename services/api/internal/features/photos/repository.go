package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type row struct {
	ID       string
	Position int
}

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func list(ctx context.Context, q pgx.Tx, userID string) ([]row, error) {
	rows, err := q.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []row{}
	for rows.Next() {
		var r row
		var p int16
		if err := rows.Scan(&r.ID, &p); err != nil {
			return nil, err
		}
		r.Position = int(p)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *Repository) List(ctx context.Context, userID string) ([]row, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return list(ctx, tx, userID)
}

// Add appends a photo, enforcing the per-user limit under an advisory lock.
func (r *Repository) Add(ctx context.Context, userID string, data []byte) (row, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return row{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return row{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return row{}, err
	}
	if count >= MaxPhotos {
		return row{}, ErrPhotoLimit
	}
	var out row
	var pos int16
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, data) VALUES ($1, $2, $3) RETURNING id, position`,
		userID, count, data).Scan(&out.ID, &pos)
	out.Position = int(pos)
	if err != nil {
		return row{}, err
	}
	return out, tx.Commit(ctx)
}

// Replace swaps the image of a photo for a fresh row (new id => client caches are busted), keeping its position.
func (r *Repository) Replace(ctx context.Context, userID, photoID string, data []byte) (row, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return row{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return row{}, err
	}
	var pos int16
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&pos)
	if errors.Is(err, pgx.ErrNoRows) {
		return row{}, ErrNotFound
	}
	if err != nil {
		return row{}, err
	}
	out := row{Position: int(pos)}
	if err := tx.QueryRow(ctx, `INSERT INTO photos (user_id, position, data) VALUES ($1, $2, $3) RETURNING id`, userID, pos, data).Scan(&out.ID); err != nil {
		return row{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, userID, photoID string) error {
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
	// Close the gap so positions stay 0..n-1 (the unique constraint is deferred).
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.rn
		FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::smallint AS rn FROM photos WHERE user_id = $1) o
		WHERE p.id = o.id AND p.position <> o.rn`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Reorder(ctx context.Context, userID string, ids []string) ([]row, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return nil, err
	}
	current, err := list(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if len(current) != len(ids) {
		return nil, ErrBadOrder
	}
	known := map[string]bool{}
	for _, c := range current {
		known[c.ID] = true
	}
	for _, id := range ids {
		if !known[id] {
			return nil, ErrBadOrder
		}
		delete(known, id) // duplicates fail the next lookup
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids); err != nil {
		return nil, err
	}
	out, err := list(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

func (r *Repository) Content(ctx context.Context, photoID string) ([]byte, string, error) {
	var data []byte
	var ct string
	err := r.pool.QueryRow(ctx, `SELECT data, content_type FROM photos WHERE id = $1`, photoID).Scan(&data, &ct)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return data, ct, err
}
