package photos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRepositoryNotFound = errors.New("photo not found")
	ErrRepositoryLimit    = errors.New("photo limit reached")
	ErrRepositoryMismatch = errors.New("photo order does not match owned photos")
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	ListByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error)
	Add(ctx context.Context, userID string, p Processed) (Photo, error)
	Replace(ctx context.Context, userID, photoID string, p Processed) (Photo, error)
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	GetBlob(ctx context.Context, viewerID, photoID string, thumb bool) (Blob, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const photoColumns = `p.id, p.position, p.width, p.height, p.updated_at`

func scanPhoto(row pgx.Row) (Photo, error) {
	var (
		id        string
		pos, w, h int
		updatedAt time.Time
	)
	if err := row.Scan(&id, &pos, &w, &h, &updatedAt); err != nil {
		return Photo{}, err
	}
	return NewPhoto(id, pos, w, h, updatedAt), nil
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	m, err := r.ListByUsers(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	return m[userID], nil
}

// ListByUsers fetches photos for many users in one query (no N+1).
func (r *PGRepository) ListByUsers(ctx context.Context, userIDs []string) (map[string][]Photo, error) {
	out := make(map[string][]Photo, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, `+photoColumns+`
		FROM photos p WHERE p.user_id = ANY($1::uuid[])
		ORDER BY p.user_id, p.position
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			uid, id   string
			pos, w, h int
			updatedAt time.Time
		)
		if err := rows.Scan(&uid, &id, &pos, &w, &h, &updatedAt); err != nil {
			return nil, err
		}
		out[uid] = append(out[uid], NewPhoto(id, pos, w, h, updatedAt))
	}
	return out, rows.Err()
}

// lockUser serialises photo mutations of one user.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	var one int
	return tx.QueryRow(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&one)
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
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotosPerUser {
		return Photo{}, ErrRepositoryLimit
	}
	photo, err := scanPhoto(tx.QueryRow(ctx, `
		INSERT INTO photos AS p (user_id, position, width, height) VALUES ($1, $2, $3, $4)
		RETURNING `+photoColumns, userID, count, p.Width, p.Height))
	if err != nil {
		return Photo{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO photo_blobs (photo_id, full_data, thumb_data) VALUES ($1, $2, $3)`,
		photo.ID, p.Full, p.Thumb); err != nil {
		return Photo{}, err
	}
	return photo, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, p Processed) (Photo, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Photo{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	photo, err := scanPhoto(tx.QueryRow(ctx, `
		UPDATE photos AS p SET width = $3, height = $4, updated_at = NOW()
		WHERE p.id = $1 AND p.user_id = $2
		RETURNING `+photoColumns, photoID, userID, p.Width, p.Height))
	if errors.Is(err, pgx.ErrNoRows) {
		return Photo{}, ErrRepositoryNotFound
	}
	if err != nil {
		return Photo{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE photo_blobs SET full_data = $2, thumb_data = $3 WHERE photo_id = $1`,
		photoID, p.Full, p.Thumb); err != nil {
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
	var position int
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position`, photoID, userID).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, position); err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	var matching, total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE id = ANY($2::uuid[])), COUNT(*) FROM photos WHERE user_id = $1
	`, userID, orderedIDs).Scan(&matching, &total); err != nil {
		return err
	}
	if total != len(orderedIDs) || matching != total {
		return ErrRepositoryMismatch
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_key DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = (x.ord - 1)::smallint
		FROM unnest($2::uuid[]) WITH ORDINALITY AS x(id, ord)
		WHERE p.id = x.id AND p.user_id = $1
	`, userID, orderedIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetBlob returns image bytes if the viewer owns the photo, or if the owner is
// active and no block exists in either direction.
func (r *PGRepository) GetBlob(ctx context.Context, viewerID, photoID string, thumb bool) (Blob, error) {
	var b Blob
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN $3 THEN pb.thumb_data ELSE pb.full_data END, p.updated_at
		FROM photos p
		JOIN photo_blobs pb ON pb.photo_id = p.id
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1
		  AND (
		    p.user_id = $2
		    OR (u.status = 'active' AND NOT EXISTS (
		      SELECT 1 FROM blocks bl
		      WHERE (bl.blocker_id = $2 AND bl.blocked_id = p.user_id)
		         OR (bl.blocker_id = p.user_id AND bl.blocked_id = $2)
		    ))
		  )
	`, photoID, viewerID, thumb).Scan(&b.Data, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Blob{}, ErrRepositoryNotFound
	}
	return b, err
}
