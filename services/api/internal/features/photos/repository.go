package photos

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("photo not found")
	ErrLimit      = errors.New("photo limit reached")
	ErrBadReorder = errors.New("photo order must list every photo exactly once")
)

type Repository interface {
	ListByUsers(ctx context.Context, userIDs []string) ([]storedPhoto, error)
	Insert(ctx context.Context, p storedPhoto) (storedPhoto, error)
	Replace(ctx context.Context, userID, photoID string, p storedPhoto) (old storedPhoto, updated storedPhoto, err error)
	Delete(ctx context.Context, userID, photoID string) (storedPhoto, error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	GetKeys(ctx context.Context, photoID string) (storageKey, thumbKey string, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const photoColumns = `id, user_id, position, storage_key, thumb_key, width, height, size_bytes, created_at`

func scanPhoto(row pgx.Row) (storedPhoto, error) {
	var p storedPhoto
	err := row.Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.ThumbKey, &p.Width, &p.Height, &p.SizeBytes, &p.CreatedAt)
	return p, err
}

// ListByUsers loads the photos of many users in one query (no N+1).
func (r *PGRepository) ListByUsers(ctx context.Context, userIDs []string) ([]storedPhoto, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+photoColumns+` FROM photos
		WHERE user_id = ANY($1::uuid[])
		ORDER BY user_id, position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedPhoto
	for rows.Next() {
		p, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// lockUser serialises concurrent photo mutations of one user.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Insert(ctx context.Context, p storedPhoto) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, p.UserID); err != nil {
		return storedPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return storedPhoto{}, err
	}
	if count >= MaxPhotosPerUser {
		return storedPhoto{}, ErrLimit
	}
	out, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, thumb_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+photoColumns,
		p.UserID, count, p.StorageKey, p.ThumbKey, p.Width, p.Height, p.SizeBytes))
	if err != nil {
		return storedPhoto{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p storedPhoto) (storedPhoto, storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	old, err := scanPhoto(tx.QueryRow(ctx, `SELECT `+photoColumns+` FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedPhoto{}, storedPhoto{}, ErrNotFound
		}
		return storedPhoto{}, storedPhoto{}, err
	}
	updated, err := scanPhoto(tx.QueryRow(ctx, `
		UPDATE photos SET storage_key = $3, thumb_key = $4, width = $5, height = $6, size_bytes = $7
		WHERE id = $1 AND user_id = $2
		RETURNING `+photoColumns,
		photoID, userID, p.StorageKey, p.ThumbKey, p.Width, p.Height, p.SizeBytes))
	if err != nil {
		return storedPhoto{}, storedPhoto{}, err
	}
	return old, updated, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (storedPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return storedPhoto{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return storedPhoto{}, err
	}
	old, err := scanPhoto(tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING `+photoColumns, photoID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedPhoto{}, ErrNotFound
		}
		return storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return storedPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn
		FROM (SELECT id, (row_number() OVER (ORDER BY position) - 1) AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn
	`, userID); err != nil {
		return storedPhoto{}, err
	}
	return old, tx.Commit(ctx)
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
	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	var existing []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, id)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	want := append([]string(nil), orderedIDs...)
	sort.Strings(existing)
	sort.Strings(want)
	if len(existing) != len(want) {
		return ErrBadReorder
	}
	for i := range existing {
		if existing[i] != want[i] || (i > 0 && want[i] == want[i-1]) {
			return ErrBadReorder
		}
	}

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.pos - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, pos)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetKeys(ctx context.Context, photoID string) (string, string, error) {
	var a, b string
	err := r.db.QueryRow(ctx, `SELECT storage_key, thumb_key FROM photos WHERE id = $1`, photoID).Scan(&a, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return a, b, err
}
