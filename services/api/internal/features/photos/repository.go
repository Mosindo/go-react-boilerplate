package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryLimit    = errors.New("photo limit reached")
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryOrder    = errors.New("order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]string, error)
	// Insert appends a photo (id supplied by caller) unless the user already has max photos.
	Insert(ctx context.Context, id, userID string, width, height, size, max int) error
	// Update changes dimensions/size of an owned photo (replacement).
	Update(ctx context.Context, id, userID string, width, height, size int) error
	// Delete removes an owned photo and compacts positions.
	Delete(ctx context.Context, id, userID string) error
	Reorder(ctx context.Context, userID string, ids []string) error
	Exists(ctx context.Context, id string) (bool, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// lockUser serialises photo mutations of one user so positions stay consistent.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	return tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
}

func (r *PGRepository) Insert(ctx context.Context, id, userID string, width, height, size, max int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&n); err != nil {
		return err
	}
	if n >= max {
		return ErrRepositoryLimit
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (id, user_id, position, width, height, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)
	`, id, userID, n, width, height, size); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Update(ctx context.Context, id, userID string, width, height, size int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE photos SET width = $3, height = $4, size_bytes = $5 WHERE id = $1 AND user_id = $2
	`, id, userID, width, height, size)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) Delete(ctx context.Context, id, userID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	// Positions are constraint-deferred, so renumbering in one statement is safe.
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn
		FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::smallint AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id
	`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	var matched int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM photos WHERE user_id = $1 AND id = ANY($2::uuid[])
	`, userID, ids).Scan(&matched); err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	if matched != len(ids) || total != len(ids) {
		return ErrRepositoryOrder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = (o.ord - 1)::smallint
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Exists(ctx context.Context, id string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE id = $1::uuid)`, id).Scan(&ok)
	return ok, err
}
