package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("photo not found")
	ErrLimit    = errors.New("photo limit reached")
	ErrBadOrder = errors.New("order must list every photo exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]string, error)
	Add(ctx context.Context, userID string, data []byte) (string, error)
	Replace(ctx context.Context, userID, photoID string, data []byte) error
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, ids []string) error
	Get(ctx context.Context, photoID string) (ownerID string, data []byte, err error)
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

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID)
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID string, data []byte) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return "", err
	}
	if count >= MaxPhotos {
		return "", ErrLimit
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, mime_type, size_bytes, data)
		VALUES ($1, $2, 'image/jpeg', $3, $4) RETURNING id`, userID, count, len(data), data).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, data []byte) error {
	tag, err := r.db.Exec(ctx, `UPDATE photos SET id = gen_random_uuid(), data = $3, size_bytes = $4, created_at = NOW() WHERE id = $1 AND user_id = $2`, photoID, userID, data, len(data))
	if isInvalidUUID(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID)
	if isInvalidUUID(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	// Compact positions so the first photo is always position 0 (the main photo).
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = s.rn
		FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::int AS rn FROM photos WHERE user_id = $1) s
		WHERE p.id = s.id`, userID); err != nil {
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
	current, err := listTx(ctx, tx, userID)
	if err != nil {
		return err
	}
	if len(current) != len(ids) {
		return ErrBadOrder
	}
	have := map[string]bool{}
	for _, id := range current {
		have[id] = true
	}
	for _, id := range ids {
		if !have[id] {
			return ErrBadOrder
		}
		delete(have, id)
	}
	for pos, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, pos); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func listTx(ctx context.Context, tx pgx.Tx, userID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
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

func (r *PGRepository) Get(ctx context.Context, photoID string) (string, []byte, error) {
	var owner string
	var data []byte
	err := r.db.QueryRow(ctx, `SELECT user_id, data FROM photos WHERE id = $1`, photoID).Scan(&owner, &data)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return "", nil, ErrNotFound
	}
	return owner, data, err
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
