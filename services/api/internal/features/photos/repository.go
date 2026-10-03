package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxPhotos = 6

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimit         = errors.New("photo limit reached")
	ErrBadOrder      = errors.New("order must list each of your photos exactly once")
	ErrProfileNeeded = errors.New("create your profile before adding photos")
)

type Photo struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
}

type NewPhoto struct {
	UserID     string
	StorageKey string
	Size       int
	Width      int
	Height     int
	ReplaceID  string // when set, the new photo takes over this photo's position
}

type Repository interface {
	Add(ctx context.Context, in NewPhoto) (Photo, string, error)
	List(ctx context.Context, userID string) ([]Photo, error)
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	StorageKeyForViewer(ctx context.Context, viewerID, photoID string) (string, error)
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

// Add inserts a photo at the end, or in place of ReplaceID. It returns the new photo and,
// for a replacement, the storage key of the file that must now be removed.
func (r *PGRepository) Add(ctx context.Context, in NewPhoto) (Photo, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, in.UserID); err != nil {
		return Photo{}, "", err
	}

	var hasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, in.UserID).Scan(&hasProfile); err != nil {
		return Photo{}, "", err
	}
	if !hasProfile {
		return Photo{}, "", ErrProfileNeeded
	}

	position := 0
	oldKey := ""
	if in.ReplaceID != "" {
		err := tx.QueryRow(ctx, `
			DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key
		`, in.ReplaceID, in.UserID).Scan(&position, &oldKey)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Photo{}, "", ErrNotFound
			}
			return Photo{}, "", err
		}
	} else {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, in.UserID).Scan(&count); err != nil {
			return Photo{}, "", err
		}
		if count >= MaxPhotos {
			return Photo{}, "", ErrLimit
		}
		position = count
	}

	p := Photo{UserID: in.UserID, Position: position, StorageKey: in.StorageKey}
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, content_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING id
	`, in.UserID, position, in.StorageKey, in.Size, in.Width, in.Height).Scan(&p.ID)
	if err != nil {
		return Photo{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, "", err
	}
	return p, oldKey, nil
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, position, storage_key FROM photos WHERE user_id = $1 ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Photo, 0, MaxPhotos)
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Delete removes the photo, closes the position gap and returns the storage key to purge.
func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
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
			return "", ErrNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = sub.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) sub
		WHERE photos.id = sub.id
	`, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
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
		return ErrBadOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE photos.id = o.id AND photos.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// StorageKeyForViewer enforces who may fetch a picture: its owner, anyone who can legitimately
// see the owner's visible profile, or a match — and never across a block.
func (r *PGRepository) StorageKeyForViewer(ctx context.Context, viewerID, photoID string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `
		SELECT ph.storage_key
		FROM photos ph
		JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $1
		  AND (
		    ph.user_id = $2
		    OR (
		      NOT EXISTS (
		        SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $2 AND b.blocked_id = ph.user_id)
		           OR (b.blocker_id = ph.user_id AND b.blocked_id = $2))
		      AND (
		        pr.is_visible
		        OR EXISTS (
		          SELECT 1 FROM matches m
		          WHERE m.user_a = LEAST($2::uuid, ph.user_id) AND m.user_b = GREATEST($2::uuid, ph.user_id))
		      )
		    )
		  )
	`, photoID, viewerID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return key, nil
}

func (r *PGRepository) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
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
