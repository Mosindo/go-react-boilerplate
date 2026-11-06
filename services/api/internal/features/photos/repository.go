package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrBadOrder     = errors.New("order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Stored, error)
	Add(ctx context.Context, userID string, key string, size, w, h int) (Stored, error)
	Replace(ctx context.Context, userID, photoID, key string, size, w, h int) (Stored, string, error)
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	// Viewable returns the storage key if viewerID may see the photo.
	Viewable(ctx context.Context, viewerID, photoID string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const cols = `id, position, width, height, created_at, storage_key`

func scanStored(row pgx.Row) (Stored, error) {
	var s Stored
	err := row.Scan(&s.ID, &s.Position, &s.Width, &s.Height, &s.CreatedAt, &s.StorageKey)
	s.URL = "/photos/" + s.ID
	return s, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Stored, error) {
	rows, err := r.db.Query(ctx, `SELECT `+cols+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Stored{}
	for rows.Next() {
		s, err := scanStored(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// lockUser serialises photo mutations per user so position/limit checks are race free.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, size, w, h int) (Stored, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Stored{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Stored{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Stored{}, err
	}
	if count >= MaxPhotos {
		return Stored{}, ErrLimitReached
	}
	s, err := scanStored(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, mime_type, size_bytes, width, height, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+cols, userID, key, storedMime, size, w, h, count))
	if err != nil {
		return Stored{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, key string, size, w, h int) (Stored, string, error) {
	var old string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID).Scan(&old)
	if err != nil {
		return Stored{}, "", mapNotFound(err)
	}
	s, err := scanStored(r.db.QueryRow(ctx, `
		UPDATE photos SET storage_key = $3, size_bytes = $4, width = $5, height = $6
		WHERE id = $1 AND user_id = $2 RETURNING `+cols, photoID, userID, key, size, w, h))
	if err != nil {
		return Stored{}, "", mapNotFound(err)
	}
	return s, old, nil
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
	if err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key); err != nil {
		return "", mapNotFound(err)
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_position DEFERRED`); err != nil {
		return "", err
	}
	// Close the gap so the main photo is always position 0.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = n.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) n
		WHERE p.id = n.id
	`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
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
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_position DEFERRED`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = (o.ord - 1)::smallint
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return ErrBadOrder
		}
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(ids) || total != len(ids) {
		return ErrBadOrder
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Viewable(ctx context.Context, viewerID, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `
		SELECT ph.storage_key FROM photos ph
		LEFT JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $2 AND (
		  ph.user_id = $1 OR (pr.user_id IS NOT NULL AND
		    NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id) OR (b.blocker_id = ph.user_id AND b.blocked_id = $1))
		    AND (pr.is_discoverable OR EXISTS (
		      SELECT 1 FROM matches m WHERE m.user_a = least($1::uuid, ph.user_id) AND m.user_b = greatest($1::uuid, ph.user_id)))
		  ))
	`, viewerID, photoID).Scan(&key)
	if err != nil {
		return "", mapNotFound(err)
	}
	return key, nil
}

func mapNotFound(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
		return ErrNotFound
	}
	return err
}
