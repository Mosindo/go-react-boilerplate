package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("photo not found")
	ErrLimit     = errors.New("photo limit reached")
	ErrLastPhoto = errors.New("cannot delete the last photo")
	ErrBadOrder  = errors.New("order must list every photo exactly once")
)

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type Blob struct {
	Data        []byte
	ContentType string
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// lockUser serialises photo mutations of one user inside a transaction.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
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
		p.URL = "/photos/" + p.ID
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Add(ctx context.Context, userID string, p Processed, max int) (Photo, error) {
	tx, err := r.pool.Begin(ctx)
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
	if count >= max {
		return Photo{}, ErrLimit
	}
	var ph Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, width, height, data)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, position`, userID, count, p.Width, p.Height, p.Data).Scan(&ph.ID, &ph.Position)
	if err != nil {
		return Photo{}, err
	}
	ph.URL = "/photos/" + ph.ID
	return ph, tx.Commit(ctx)
}

// Replace swaps the image of an existing photo, keeping its position. The new
// photo gets a new id so that cached copies of the old one are never reused.
func (r *Repository) Replace(ctx context.Context, userID, photoID string, p Processed) (Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var pos int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&pos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Photo{}, ErrNotFound
		}
		return Photo{}, err
	}
	var ph Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, width, height, data)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, position`, userID, pos, p.Width, p.Height, p.Data).Scan(&ph.ID, &ph.Position)
	if err != nil {
		return Photo{}, err
	}
	ph.URL = "/photos/" + ph.ID
	return ph, tx.Commit(ctx)
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
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	var pos int
	err = tx.QueryRow(ctx, `SELECT position FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&pos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if count <= 1 {
		return ErrLastPhoto
	}
	if _, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1`, photoID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, pos); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Reorder applies the full ordered list of photo ids; index 0 is the main photo.
func (r *Repository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing[id] = true
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if len(ids) != len(existing) {
		return ErrBadOrder
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !existing[id] || seen[id] {
			return ErrBadOrder
		}
		seen[id] = true
	}
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $2 WHERE id = $1 AND user_id = $3`, id, i, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Fetch returns the image only if viewerID may see it: the owner, a match
// partner, or any non-blocked viewer while the owner is discoverable.
func (r *Repository) Fetch(ctx context.Context, viewerID, photoID string) (Blob, error) {
	var b Blob
	err := r.pool.QueryRow(ctx, `
		SELECT p.data, p.content_type FROM photos p
		WHERE p.id = $1 AND (
			p.user_id = $2
			OR (
				NOT EXISTS (SELECT 1 FROM blocks bl
					WHERE (bl.blocker_id = $2 AND bl.blocked_id = p.user_id) OR (bl.blocker_id = p.user_id AND bl.blocked_id = $2))
				AND (
					EXISTS (SELECT 1 FROM matches m
						WHERE m.user_a_id = LEAST($2::uuid, p.user_id) AND m.user_b_id = GREATEST($2::uuid, p.user_id))
					OR EXISTS (SELECT 1 FROM profiles pr WHERE pr.user_id = p.user_id AND pr.discoverable)
				)
			)
		)`, photoID, viewerID).Scan(&b.Data, &b.ContentType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Blob{}, ErrNotFound
		}
		return Blob{}, err
	}
	return b, nil
}
