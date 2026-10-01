package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPhotoNotFound = errors.New("photo not found")
	ErrPhotoLimit    = errors.New("photo limit reached")
	ErrBadOrder      = errors.New("order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]stored, error)
	Add(ctx context.Context, userID, key string, size, w, h int) (stored, error)
	Replace(ctx context.Context, userID, photoID, key string, size, w, h int) (stored, string, error)
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	Get(ctx context.Context, photoID string) (stored, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const photoColumns = `id, user_id, position, storage_key, size_bytes, width, height, created_at`

func scanPhoto(row pgx.Row) (stored, error) {
	var s stored
	err := row.Scan(&s.ID, &s.UserID, &s.Position, &s.StorageKey, &s.SizeBytes, &s.Width, &s.Height, &s.CreatedAt)
	return s, err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]stored, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]stored, 0, MaxPhotos)
	for rows.Next() {
		s, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (stored, error) {
	s, err := scanPhoto(r.pool.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, photoID))
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, ErrPhotoNotFound
	}
	return s, err
}

// lockUser serializes photo mutations of one user inside tx.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Add(ctx context.Context, userID, key string, size, w, h int) (stored, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return stored{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return stored{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return stored{}, err
	}
	if count >= MaxPhotos {
		return stored{}, ErrPhotoLimit
	}
	s, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, count, key, size, w, h))
	if err != nil {
		return stored{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, key string, size, w, h int) (stored, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return stored{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return stored{}, "", err
	}
	var oldKey string
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&oldKey, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, "", ErrPhotoNotFound
	}
	if err != nil {
		return stored{}, "", err
	}
	s, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, size_bytes, width, height)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, position, key, size, w, h))
	if err != nil {
		return stored{}, "", err
	}
	return s, oldKey, tx.Commit(ctx)
}

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
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPhotoNotFound
	}
	if err != nil {
		return "", err
	}
	// Close the gap so positions stay 0..n-1.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.pos
		FROM (SELECT id, (row_number() OVER (ORDER BY position) - 1)::int AS pos FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.pos`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
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
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrBadOrder
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS uq_photos_user_position DEFERRED`); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.pos::int - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, pos)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != count {
		return ErrBadOrder
	}
	return tx.Commit(ctx)
}
