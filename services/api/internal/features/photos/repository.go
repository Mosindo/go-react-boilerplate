package photos

import (
	"context"
	"errors"
	"sort"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxPhotos = 6

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrConflict     = errors.New("photo position conflict")
	ErrBadOrder     = errors.New("order must list every photo exactly once")
	ErrLastPhoto    = errors.New("at least one photo is required")
)

type Photo struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
}

type Repository interface {
	List(ctx context.Context, userID string) ([]profiles.PhotoRef, error)
	Add(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error)
	Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (oldKey string, photo Photo, err error)
	Delete(ctx context.Context, userID, photoID string) (storageKey string, err error)
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	// GetVisible returns the photo only if viewerID may see it.
	GetVisible(ctx context.Context, viewerID, photoID string) (Photo, error)
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]profiles.PhotoRef, error) {
	m, err := profiles.LoadPhotos(ctx, r.pool, []string{userID})
	if err != nil {
		return nil, err
	}
	return m[userID], nil
}

func (r *PGRepository) Add(ctx context.Context, userID, storageKey string, width, height, size int) (Photo, error) {
	var p Photo
	// Lowest free position; the CHECK (position <= 5) and the unique (user_id, position)
	// constraint make the limit hold even under concurrent uploads.
	err := r.pool.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, width, height, size_bytes)
		SELECT $1, s.pos, $2, $3, $4, $5
		FROM generate_series(0, 5) AS s(pos)
		WHERE NOT EXISTS (SELECT 1 FROM photos WHERE user_id = $1 AND position = s.pos)
		ORDER BY s.pos
		LIMIT 1
		RETURNING id, user_id, position, storage_key, width, height, size_bytes
	`, userID, storageKey, width, height, size).Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Photo{}, ErrLimitReached
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Photo{}, ErrConflict
		}
		return Photo{}, err
	}
	return p, nil
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID, storageKey string, width, height, size int) (string, Photo, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldKey string
	if err := tx.QueryRow(ctx, `SELECT storage_key FROM photos WHERE id = $1 AND user_id = $2 FOR UPDATE`, photoID, userID).Scan(&oldKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", Photo{}, ErrNotFound
		}
		return "", Photo{}, err
	}
	var p Photo
	if err := tx.QueryRow(ctx, `
		UPDATE photos SET storage_key = $3, width = $4, height = $5, size_bytes = $6
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, position, storage_key, width, height, size_bytes
	`, photoID, userID, storageKey, width, height, size).Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes); err != nil {
		return "", Photo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", Photo{}, err
	}
	return oldKey, p, nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// A profile always keeps one photo: lock the user's photo rows so two concurrent
	// deletions cannot both pass the check.
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM (SELECT 1 FROM photos WHERE user_id = $1 FOR UPDATE) locked`, userID).Scan(&total); err != nil {
		return "", err
	}
	if total <= 1 {
		var owned bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM photos WHERE id = $1 AND user_id = $2)`, photoID, userID).Scan(&owned); err != nil {
			return "", err
		}
		if owned {
			return "", ErrLastPhoto
		}
		return "", ErrNotFound
	}

	var key string
	if err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	// Close the gap so positions stay 0..n-1 and the first photo remains the main one.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = ranked.new_pos
		FROM (
			SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::smallint AS new_pos
			FROM photos WHERE user_id = $1
		) ranked
		WHERE p.id = ranked.id AND p.position <> ranked.new_pos
	`, userID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return key, nil
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1 FOR UPDATE`, userID)
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
	if !sameSet(existing, orderedIDs) {
		return ErrBadOrder
	}

	if _, err := tx.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
		return err
	}
	for i, id := range orderedIDs {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) GetVisible(ctx context.Context, viewerID, photoID string) (Photo, error) {
	var p Photo
	err := r.pool.QueryRow(ctx, `
		SELECT ph.id, ph.user_id, ph.position, ph.storage_key, ph.width, ph.height, ph.size_bytes
		FROM photos ph
		JOIN users u ON u.id = ph.user_id AND u.status = 'active'
		LEFT JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $2
		  AND (
		        ph.user_id = $1
		        OR (
		             NOT EXISTS (SELECT 1 FROM blocks b
		                         WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id)
		                            OR (b.blocker_id = ph.user_id AND b.blocked_id = $1))
		             AND (
		                  COALESCE(pr.is_discoverable, FALSE)
		                  OR EXISTS (SELECT 1 FROM matches m
		                             WHERE m.user_a_id = LEAST($1::uuid, ph.user_id) AND m.user_b_id = GREATEST($1::uuid, ph.user_id))
		             )
		        )
		  )
	`, viewerID, photoID).Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.Width, &p.Height, &p.SizeBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Photo{}, ErrNotFound
		}
		return Photo{}, err
	}
	return p, nil
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
