package files

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound   = errors.New("files: photo not found")
	ErrRepositoryLimit      = errors.New("files: photo limit reached")
	ErrRepositoryBadReorder = errors.New("files: reorder must list every photo exactly once")
)

type NewPhoto struct {
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
}

type Repository interface {
	ListByUsers(ctx context.Context, userIDs []string) ([]Photo, error)
	Get(ctx context.Context, photoID string) (Photo, error)
	Add(ctx context.Context, userID string, p NewPhoto) (Photo, error)
	Replace(ctx context.Context, userID, photoID string, p NewPhoto) (oldKey string, err error)
	Delete(ctx context.Context, userID, photoID string) (key string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	KeysForUser(ctx context.Context, userID string) ([]string, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

const photoColumns = `id, user_id, storage_key, position, width, height, size_bytes, created_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var p Photo
	err := row.Scan(&p.ID, &p.UserID, &p.StorageKey, &p.Position, &p.Width, &p.Height, &p.SizeBytes, &p.CreatedAt)
	return p, err
}

func (r *PGRepository) ListByUsers(ctx context.Context, userIDs []string) ([]Photo, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT `+photoColumns+`
		FROM photos
		WHERE user_id = ANY($1)
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var photos []Photo
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (Photo, error) {
	p, err := scanPhoto(r.dbPool.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1`, photoID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrRepositoryNotFound
	}
	return p, err
}

// lockUser serializes photo mutations of one user so positions stay dense.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "photos:"+userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID string, np NewPhoto) (Photo, error) {
	tx, err := r.dbPool.Begin(ctx)
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
	if count >= MaxPhotos {
		return Photo{}, ErrRepositoryLimit
	}

	p, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+photoColumns,
		userID, np.StorageKey, count, np.Width, np.Height, np.SizeBytes))
	if err != nil {
		return Photo{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, np NewPhoto) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var oldKey string
	err = tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6
		WHERE id = $1 AND user_id = $2
	`, photoID, userID, np.StorageKey, np.Width, np.Height, np.SizeBytes); err != nil {
		return "", err
	}
	return oldKey, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}

	var key string
	var position int
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position
	`, photoID, userID).Scan(&key, &position)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepositoryNotFound
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2
	`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.dbPool.Begin(ctx)
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
	var current []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		current = append(current, id)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if !sameSet(current, orderedIDs) {
		return ErrRepositoryBadReorder
	}

	positions := make([]int32, len(orderedIDs))
	for i := range orderedIDs {
		positions[i] = int32(i)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = v.pos
		FROM unnest($2::uuid[], $3::int[]) AS v(id, pos)
		WHERE p.id = v.id AND p.user_id = $1
	`, userID, orderedIDs, positions); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) KeysForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.dbPool.Query(ctx, `SELECT storage_key FROM photos WHERE user_id = $1`, userID)
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

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
