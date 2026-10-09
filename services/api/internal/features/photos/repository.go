package photos

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("photos: not found")
	ErrLimit    = errors.New("photos: limit reached")
	ErrBadOrder = errors.New("photos: order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]StoredPhoto, error)
	Insert(ctx context.Context, userID, storageKey, mime string, size, width, height int) (StoredPhoto, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (oldKey string, p StoredPhoto, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, ids []string) error
	GetVisible(ctx context.Context, viewerID, photoID string) (StoredPhoto, error)
	StorageKeys(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]StoredPhoto, error) {
	return listTx(ctx, r.db, userID)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listTx(ctx context.Context, q querier, userID string) ([]StoredPhoto, error) {
	rows, err := q.Query(ctx, `SELECT id::text, user_id::text, storage_key, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredPhoto
	for rows.Next() {
		var p StoredPhoto
		if err := rows.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey, mime string, size, width, height int) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return StoredPhoto{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotosPerUser {
		return StoredPhoto{}, ErrLimit
	}

	p := StoredPhoto{UserID: userID, StorageKey: storageKey, Position: count}
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text
	`, userID, storageKey, mime, size, width, height, count).Scan(&p.ID)
	if err != nil {
		if isFK(err) {
			return StoredPhoto{}, ErrNotFound
		}
		return StoredPhoto{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return StoredPhoto{}, err
	}
	return p, nil
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, size, width, height int) (string, StoredPhoto, error) {
	var oldKey string
	p := StoredPhoto{UserID: userID, StorageKey: storageKey}
	err := r.db.QueryRow(ctx, `
		UPDATE photos AS ph
		SET storage_key = $3, size_bytes = $4, width = $5, height = $6, updated_at = NOW()
		FROM (SELECT id, storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE) old
		WHERE ph.id = old.id
		RETURNING old.storage_key, ph.id::text, ph.position
	`, photoID, userID, storageKey, size, width, height).Scan(&oldKey, &p.ID, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", StoredPhoto{}, ErrNotFound
		}
		return "", StoredPhoto{}, err
	}
	return oldKey, p, nil
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
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return "", ErrNotFound
		}
		return "", err
	}
	// Close the gap so positions stay 0..n-1 and the next photo becomes main if needed.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn - 1
	`, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	current, err := listTx(ctx, tx, userID)
	if err != nil {
		return err
	}
	if len(current) != len(ids) {
		return ErrBadOrder
	}
	have := make(map[string]struct{}, len(current))
	for _, p := range current {
		have[p.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := have[id]; !ok {
			return ErrBadOrder
		}
		if _, dup := seen[id]; dup {
			return ErrBadOrder
		}
		seen[id] = struct{}{}
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3, updated_at = NOW() WHERE id = $1 AND user_id = $2`, id, userID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// GetVisible returns the photo only if viewerID may see it: the owner, or any
// user not in a block relation with the owner while the owner's profile is
// discoverable, matched with the viewer, or has liked the viewer.
func (r *PGRepository) GetVisible(ctx context.Context, viewerID, photoID string) (StoredPhoto, error) {
	var p StoredPhoto
	err := r.db.QueryRow(ctx, `
		SELECT ph.id::text, ph.user_id::text, ph.storage_key, ph.position
		FROM photos ph
		LEFT JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $2
		  AND (
		    ph.user_id = $1
		    OR (
		      NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id) OR (b.blocker_id = ph.user_id AND b.blocked_id = $1))
		      AND (
		        COALESCE(pr.discoverable, FALSE)
		        OR EXISTS (SELECT 1 FROM matches m
		                   WHERE m.unmatched_at IS NULL AND m.user_a = LEAST($1::uuid, ph.user_id) AND m.user_b = GREATEST($1::uuid, ph.user_id))
		        OR EXISTS (SELECT 1 FROM swipes s WHERE s.user_id = ph.user_id AND s.target_id = $1 AND s.action = 'like')
		      )
		    )
		  )
	`, viewerID, photoID).Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return StoredPhoto{}, ErrNotFound
		}
		return StoredPhoto{}, err
	}
	return p, nil
}

func (r *PGRepository) StorageKeys(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
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

func toPhoto(p StoredPhoto) Photo {
	return Photo{ID: p.ID, Position: p.Position, URL: profiles.PhotoURL(p.ID)}
}

func isFK(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
