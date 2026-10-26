package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryLimit    = errors.New("photo limit reached")
	ErrRepositoryOrder    = errors.New("order must list every photo exactly once")
)

type StoredFile struct {
	ContentType string
	Data        []byte
}

type Repository interface {
	List(ctx context.Context, userID string) ([]string, error)
	Add(ctx context.Context, userID string, img processedImage) (string, error)
	Replace(ctx context.Context, userID, photoID string, img processedImage) (string, error)
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, ids []string) error
	GetFile(ctx context.Context, photoID string) (StoredFile, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) inTx(ctx context.Context, userID string, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialise photo mutations per user so positions never collide.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "photos:"+userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]string, error) {
	return listIDs(ctx, r.pool, userID)
}

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listIDs(ctx context.Context, q queryer, userID string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 ORDER BY position`, userID)
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

func (r *PGRepository) Add(ctx context.Context, userID string, img processedImage) (string, error) {
	var id string
	err := r.inTx(ctx, userID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, position, width, height, data)
			SELECT $1, COALESCE(MAX(position) + 1, 0), $2, $3, $4 FROM photos WHERE user_id = $1
			HAVING COUNT(*) < $5
			RETURNING id
		`, userID, img.Width, img.Height, img.Data, MaxPhotos).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepositoryLimit
		}
		return err
	})
	return id, err
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, img processedImage) (string, error) {
	var id string
	err := r.inTx(ctx, userID, func(tx pgx.Tx) error {
		var position int
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&position)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRepositoryNotFound
		}
		if err != nil {
			return err
		}
		// A fresh id gives the replacement a fresh URL, so caches never serve the old picture.
		return tx.QueryRow(ctx, `
			INSERT INTO photos (user_id, position, width, height, data) VALUES ($1, $2, $3, $4, $5) RETURNING id
		`, userID, position, img.Width, img.Height, img.Data).Scan(&id)
	})
	return id, mapInvalidUUID(err)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) error {
	err := r.inTx(ctx, userID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrRepositoryNotFound
		}
		_, err = tx.Exec(ctx, `
			UPDATE photos p SET position = s.rn - 1
			FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) s
			WHERE p.id = s.id
		`, userID)
		return err
	})
	return mapInvalidUUID(err)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	err := r.inTx(ctx, userID, func(tx pgx.Tx) error {
		current, err := listIDs(ctx, tx, userID)
		if err != nil {
			return err
		}
		if len(current) != len(ids) {
			return ErrRepositoryOrder
		}
		have := make(map[string]struct{}, len(current))
		for _, id := range current {
			have[id] = struct{}{}
		}
		for _, id := range ids {
			if _, ok := have[id]; !ok {
				return ErrRepositoryOrder
			}
			delete(have, id)
		}
		_, err = tx.Exec(ctx, `
			UPDATE photos p SET position = v.pos - 1
			FROM unnest($2::uuid[]) WITH ORDINALITY AS v(id, pos)
			WHERE p.id = v.id AND p.user_id = $1
		`, userID, ids)
		return err
	})
	if err != nil && isInvalidUUID(err) {
		return ErrRepositoryOrder
	}
	return err
}

func (r *PGRepository) GetFile(ctx context.Context, photoID string) (StoredFile, error) {
	var f StoredFile
	err := r.pool.QueryRow(ctx, `SELECT content_type, data FROM photos WHERE id = $1`, photoID).Scan(&f.ContentType, &f.Data)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return StoredFile{}, ErrRepositoryNotFound
	}
	return f, err
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

func mapInvalidUUID(err error) error {
	if err != nil && isInvalidUUID(err) {
		return ErrRepositoryNotFound
	}
	return err
}
