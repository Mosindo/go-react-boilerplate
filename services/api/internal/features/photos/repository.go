package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("photo not found")
	ErrLimit      = errors.New("photo limit reached")
	ErrBadReorder = errors.New("order must list each of your photos exactly once")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Add(ctx context.Context, userID string, p Processed) (Photo, error)
	Replace(ctx context.Context, userID, photoID string, p Processed) (Photo, error)
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, ids []string) error
	Image(ctx context.Context, photoID string, thumbnail bool) (StoredImage, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// lockUser serializes photo mutations per user so positions stay contiguous.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position, width, height FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position, &p.Width, &p.Height); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) Add(ctx context.Context, userID string, p Processed) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrLimit
	}
	photo := Photo{Position: count, Width: p.Width, Height: p.Height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, width, height, data, thumbnail)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, count, p.Width, p.Height, p.Data, p.Thumbnail).Scan(&photo.ID); err != nil {
		return Photo{}, err
	}
	return photo, tx.Commit(ctx)
}

// Replace swaps the image at the same position under a new id, so cached
// copies of the old URL can never show stale content.
func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p Processed) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, err
	}
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrNotFound
	}
	if err != nil {
		return Photo{}, err
	}
	photo := Photo{Position: position, Width: p.Width, Height: p.Height}
	if err := tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, width, height, data, thumbnail)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, userID, position, p.Width, p.Height, p.Data, p.Thumbnail).Scan(&photo.ID); err != nil {
		return Photo{}, err
	}
	return photo, tx.Commit(ctx)
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
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = s.rn - 1
		FROM (SELECT id, row_number() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) s
		WHERE p.id = s.id AND p.position <> s.rn - 1
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
	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE id = ANY($2::uuid[])), count(*) FROM photos WHERE user_id = $1
	`, userID, ids).Scan(&matching, &total); err != nil {
		return err
	}
	if matching != len(ids) || total != len(ids) {
		return ErrBadReorder
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = (t.ord - 1)::smallint
		FROM unnest($2::uuid[]) WITH ORDINALITY AS t(id, ord)
		WHERE p.id = t.id AND p.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Image(ctx context.Context, photoID string, thumbnail bool) (StoredImage, error) {
	column := "data"
	if thumbnail {
		column = "thumbnail"
	}
	var img StoredImage
	err := r.db.QueryRow(ctx, `SELECT user_id, content_type, `+column+` FROM photos WHERE id = $1`, photoID).
		Scan(&img.OwnerID, &img.ContentType, &img.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredImage{}, ErrNotFound
	}
	return img, err
}
