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
	ErrRepositoryOrder    = errors.New("order must list each photo exactly once")
)

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}

type NewPhoto struct {
	UserID     string
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
}

type Repository interface {
	Insert(ctx context.Context, p NewPhoto, limit int) (StoredPhoto, error)
	Get(ctx context.Context, photoID string) (StoredPhoto, error)
	Replace(ctx context.Context, photoID, userID string, p NewPhoto) (oldKey string, err error)
	Delete(ctx context.Context, photoID, userID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	StorageKeys(ctx context.Context, userID string) ([]string, error)
	ListIDs(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

// lockUser serialises photo mutations per user so positions and the cap stay consistent.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, p NewPhoto, limit int) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, p.UserID); err != nil {
		return StoredPhoto{}, err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= limit {
		return StoredPhoto{}, ErrRepositoryFull
	}

	out := StoredPhoto{UserID: p.UserID, StorageKey: p.StorageKey, Position: count}
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, content_type, size_bytes, width, height, position)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5, $6)
		RETURNING id
	`, p.UserID, p.StorageKey, p.SizeBytes, p.Width, p.Height, count).Scan(&out.ID)
	if err != nil {
		return StoredPhoto{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (StoredPhoto, error) {
	var p StoredPhoto
	err := r.db.QueryRow(ctx, `SELECT id, user_id, storage_key, position FROM photos WHERE id = $1`, photoID).
		Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPhoto{}, ErrRepositoryNotFound
		}
		return StoredPhoto{}, err
	}
	return p, nil
}

func (r *PGRepository) Replace(ctx context.Context, photoID, userID string, p NewPhoto) (string, error) {
	var oldKey string
	err := r.db.QueryRow(ctx, `
		UPDATE photos AS ph
		SET storage_key = $3, size_bytes = $4, width = $5, height = $6
		FROM (SELECT id, storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE) AS old
		WHERE ph.id = old.id
		RETURNING old.storage_key
	`, photoID, userID, p.StorageKey, p.SizeBytes, p.Width, p.Height).Scan(&oldKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepositoryNotFound
		}
		return "", err
	}
	return oldKey, nil
}

func (r *PGRepository) Delete(ctx context.Context, photoID, userID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepositoryNotFound
		}
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

// Reorder applies a full permutation. The position constraint is deferred, so the
// intermediate duplicate positions inside the transaction are fine.
func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}

	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*) FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != len(orderedIDs) || total != len(orderedIDs) {
		return ErrRepositoryOrder
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

func (r *PGRepository) ListIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
