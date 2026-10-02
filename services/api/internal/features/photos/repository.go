package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepoNotFound = errors.New("photo not found")
	ErrRepoFull     = errors.New("photo limit reached")
)

type NewPhoto struct {
	StorageKey  string
	ContentType string
	SizeBytes   int
	Width       int
	Height      int
}

type Repository interface {
	Add(ctx context.Context, userID string, p NewPhoto) (Stored, error)
	Replace(ctx context.Context, userID, photoID string, p NewPhoto) (oldKey string, s Stored, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	ListByUsers(ctx context.Context, userIDs []string) ([]Stored, error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	StorageKey(ctx context.Context, photoID string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Add(ctx context.Context, userID string, p NewPhoto) (Stored, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Stored{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialise uploads of one user so the count/position check is race free.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		return Stored{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Stored{}, err
	}
	if count >= MaxPhotosPerUser {
		return Stored{}, ErrRepoFull
	}
	var s Stored
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, content_type, size_bytes, width, height)
		VALUES ($1, (SELECT COALESCE(MAX(position) + 1, 0) FROM photos WHERE user_id = $1), $2, $3, $4, $5, $6)
		RETURNING id, user_id, position, storage_key, created_at`,
		userID, p.StorageKey, p.ContentType, p.SizeBytes, p.Width, p.Height).
		Scan(&s.ID, &s.UserID, &s.Position, &s.StorageKey, &s.CreatedAt)
	if err != nil {
		return Stored{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p NewPhoto) (string, Stored, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", Stored{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var oldKey string
	if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", Stored{}, ErrRepoNotFound
		}
		return "", Stored{}, err
	}
	var s Stored
	err = tx.QueryRow(ctx, `
		UPDATE photos SET storage_key = $3, content_type = $4, size_bytes = $5, width = $6, height = $7, created_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, position, storage_key, created_at`,
		photoID, userID, p.StorageKey, p.ContentType, p.SizeBytes, p.Width, p.Height).
		Scan(&s.ID, &s.UserID, &s.Position, &s.StorageKey, &s.CreatedAt)
	if err != nil {
		return "", Stored{}, err
	}
	return oldKey, s, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	if err != nil {
		return "", err
	}
	// Close the gap so positions stay 0..n-1.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn
		FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::int AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) ListByUsers(ctx context.Context, userIDs []string) ([]Stored, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, position, storage_key, created_at FROM photos
		WHERE user_id = ANY($1::uuid[]) ORDER BY user_id, position`, userIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Stored, error) {
		var s Stored
		err := row.Scan(&s.ID, &s.UserID, &s.Position, &s.StorageKey, &s.CreatedAt)
		return s, err
	})
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	existing, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if !samePhotoSet(existing, orderedIDs) {
		return ErrRepoNotFound
	}
	for i, id := range orderedIDs {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) StorageKey(ctx context.Context, photoID string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1`, photoID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return key, err
}

func samePhotoSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			return false
		}
		delete(seen, id)
	}
	return len(seen) == 0
}
