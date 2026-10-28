package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxPhotos = 6

var (
	ErrRepoNoProfile   = errors.New("photos: profile missing")
	ErrRepoLimit       = errors.New("photos: limit reached")
	ErrRepoNotFound    = errors.New("photos: not found")
	ErrRepoBadOrdering = errors.New("photos: order does not match owned photos")
)

type Repository interface {
	Add(ctx context.Context, userID, id, storageKey string, width, height, size int) (Photo, error)
	List(ctx context.Context, userID string) ([]Photo, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Get(ctx context.Context, photoID string) (stored, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// lockProfile serialises photo mutations per user so positions and the cap stay consistent.
func lockProfile(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepoNoProfile
	}
	return err
}

func (r *PGRepository) Add(ctx context.Context, userID, id, storageKey string, width, height, size int) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProfile(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrRepoLimit
	}
	p := Photo{ID: id, Position: int16(count), Width: width, Height: height, URL: "/photos/" + id + "/file"}
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (id, user_id, position, storage_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, id, userID, p.Position, storageKey, width, height, size); err != nil {
		return Photo{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position, width, height FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Photo, error) {
		var p Photo
		err := row.Scan(&p.ID, &p.Position, &p.Width, &p.Height)
		p.URL = "/photos/" + p.ID + "/file"
		return p, err
	})
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProfile(ctx, tx, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.ord - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, ord)
		WHERE p.id = o.id AND p.user_id = $1`, userID, ids)
	if err != nil {
		return err
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return err
	}
	// Every owned photo must appear exactly once and nothing foreign may be named.
	if int(tag.RowsAffected()) != len(ids) || total != len(ids) {
		return ErrRepoBadOrdering
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockProfile(ctx, tx, userID); err != nil {
		if errors.Is(err, ErrRepoNoProfile) {
			return "", ErrRepoNotFound
		}
		return "", err
	}
	var key string
	var position int16
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key, position`, photoID, userID).Scan(&key, &position)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrRepoNotFound
		}
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Get(ctx context.Context, photoID string) (stored, error) {
	var s stored
	err := r.db.QueryRow(ctx, `SELECT id, user_id, storage_key, position, width, height FROM photos WHERE id = $1`, photoID).
		Scan(&s.ID, &s.UserID, &s.StorageKey, &s.Position, &s.Width, &s.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, ErrRepoNotFound
	}
	return s, err
}
