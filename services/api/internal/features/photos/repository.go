package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/imaging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLimit    = errors.New("photos: limit reached")
	ErrNotFound = errors.New("photos: not found")
	ErrNotPerm  = errors.New("photos: ids are not a permutation of own photos")
)

type Repository interface {
	Add(ctx context.Context, userID string, img imaging.Result) (profiles.Photo, error)
	Replace(ctx context.Context, userID, photoID string, img imaging.Result) (profiles.Photo, error)
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error)
	Content(ctx context.Context, viewerID, photoID string) ([]byte, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

// lockUser serialises photo mutations of one user (count checks, reorders).
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID string, img imaging.Result) (profiles.Photo, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return profiles.Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return profiles.Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return profiles.Photo{}, err
	}
	if count >= MaxPhotos {
		return profiles.Photo{}, ErrLimit
	}
	var p profiles.Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, content, byte_size, width, height)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, position`, userID, count, img.JPEG, len(img.JPEG), img.Width, img.Height).Scan(&p.ID, &p.Position)
	if err != nil {
		return profiles.Photo{}, err
	}
	p.URL = profiles.PhotoURL(p.ID)
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, img imaging.Result) (profiles.Photo, error) {
	var p profiles.Photo
	err := r.pool.QueryRow(ctx, `
		UPDATE photos
		SET content = $3, byte_size = $4, width = $5, height = $6, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, position`, photoID, userID, img.JPEG, len(img.JPEG), img.Width, img.Height).Scan(&p.ID, &p.Position)
	if errors.Is(err, pgx.ErrNoRows) {
		return profiles.Photo{}, ErrNotFound
	}
	p.URL = profiles.PhotoURL(p.ID)
	return p, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) error {
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
	// The unique (user_id, position) constraint is deferred, so compaction is safe in one statement.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn - 1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) ([]profiles.Photo, error) {
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
	own := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		own[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) != len(own) {
		return nil, ErrNotPerm
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, ok := own[id]; !ok {
			return nil, ErrNotPerm
		}
		if _, dup := seen[id]; dup {
			return nil, ErrNotPerm
		}
		seen[id] = struct{}{}
	}
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE photos p SET position = t.ord - 1
			FROM unnest($2::uuid[]) WITH ORDINALITY AS t(id, ord)
			WHERE p.id = t.id AND p.user_id = $1`, userID, ids); err != nil {
			return nil, err
		}
	}
	list, err := profiles.PhotosFor(ctx, tx, []string{userID})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return list[userID], nil
}

// Content returns the JPEG bytes if viewerID may see the photo: the owner;
// or anyone not blocked in either direction when the owner is discoverable or
// matched with the viewer. Otherwise ErrNotFound (indistinguishable from missing).
func (r *PGRepository) Content(ctx context.Context, viewerID, photoID string) ([]byte, error) {
	var content []byte
	err := r.pool.QueryRow(ctx, `
		SELECT p.content
		FROM photos p
		WHERE p.id = $1
		  AND (
		    p.user_id = $2
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $2 AND b.blocked_id = p.user_id)
		           OR (b.blocker_id = p.user_id AND b.blocked_id = $2)
		      )
		      AND (
		        EXISTS (SELECT 1 FROM profiles pr WHERE pr.user_id = p.user_id AND pr.is_discoverable)
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.user_a = LEAST(p.user_id, $2::uuid) AND m.user_b = GREATEST(p.user_id, $2::uuid)
		        )
		      )
		    )
		  )`, photoID, viewerID).Scan(&content)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return content, err
}
