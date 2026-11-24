package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryFull     = errors.New("photo limit reached")
	ErrRepositoryMismatch = errors.New("photo set mismatch")
)

type Repository interface {
	ListForUser(ctx context.Context, userID string) ([]Photo, error)
	ListForUsers(ctx context.Context, userIDs []string) ([]Photo, error)
	Get(ctx context.Context, photoID string) (Photo, error)
	Insert(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error)
	ReplaceFile(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, photo Photo, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	StorageKeysForUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const photoColumns = `id, user_id, position, storage_key, width, height, size_bytes, created_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes, &p.CreatedAt)
	return p, err
}

func collect(rows pgx.Rows) ([]Photo, error) {
	defer rows.Close()
	out := make([]Photo, 0)
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListForUser(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *PGRepository) ListForUsers(ctx context.Context, userIDs []string) ([]Photo, error) {
	if len(userIDs) == 0 {
		return []Photo{}, nil
	}
	rows, err := r.db.Query(ctx, `SELECT `+photoColumns+` FROM photos WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	return collect(rows)
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (Photo, error) {
	p, err := scanPhoto(r.db.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, photoID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrRepositoryNotFound
	}
	return p, err
}

// lockUser serialises photo mutations of one user so positions stay dense.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('photos:' || $1::text, 0))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error) {
	tx, err := r.db.Begin(ctx)
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
	if count >= MaxPhotosPerUser {
		return Photo{}, ErrRepositoryFull
	}
	p, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, mime_type, width, height, size_bytes)
		VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6)
		RETURNING `+photoColumns, userID, count, storageKey, width, height, size))
	if err != nil {
		return Photo{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) ReplaceFile(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", Photo{}, ErrRepositoryNotFound
	}
	if err != nil {
		return "", Photo{}, err
	}
	p, err := scanPhoto(tx.QueryRow(ctx, `
		UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6, created_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING `+photoColumns, photoID, userID, storageKey, width, height, size))
	if err != nil {
		return "", Photo{}, err
	}
	return oldKey, p, tx.Commit(ctx)
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
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*)
		FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != len(orderedIDs) || total != len(orderedIDs) {
		return ErrRepositoryMismatch
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
