package photos

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoProfile = errors.New("profile required")
	ErrLimit     = errors.New("photo limit reached")
	ErrNotFound  = errors.New("photo not found")
	ErrBadOrder  = errors.New("order must list every photo exactly once")
)

type Repository interface {
	Add(ctx context.Context, p StoredPhoto) (StoredPhoto, error)
	Delete(ctx context.Context, userID, photoID string) (StoredPhoto, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	List(ctx context.Context, userID string) ([]profiles.Photo, error)
	FileFor(ctx context.Context, viewerID, photoID string) (storageKey, thumbKey string, createdAt time.Time, err error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Add appends a photo at the next free position. The profile row lock serialises concurrent uploads
// so the six-photo cap and position uniqueness always hold.
func (r *PGRepository) Add(ctx context.Context, p StoredPhoto) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var locked string
	if err := tx.QueryRow(ctx, `SELECT user_id FROM profiles WHERE user_id = $1 FOR UPDATE`, p.UserID).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredPhoto{}, ErrNoProfile
		}
		return StoredPhoto{}, err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, p.UserID).Scan(&count); err != nil {
		return StoredPhoto{}, err
	}
	if count >= MaxPhotos {
		return StoredPhoto{}, ErrLimit
	}
	p.Position = count
	if _, err := tx.Exec(ctx, `
		INSERT INTO photos (id, user_id, position, storage_key, thumb_key, width, height, size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, p.ID, p.UserID, p.Position, p.StorageKey, p.ThumbKey, p.Width, p.Height, p.SizeBytes); err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (StoredPhoto, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return StoredPhoto{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT 1 FROM profiles WHERE user_id = $1 FOR UPDATE`, userID); err != nil {
		return StoredPhoto{}, err
	}
	var p StoredPhoto
	err = tx.QueryRow(ctx, `
		DELETE FROM photos WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, position, storage_key, thumb_key
	`, photoID, userID).Scan(&p.ID, &p.UserID, &p.Position, &p.StorageKey, &p.ThumbKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredPhoto{}, ErrNotFound
	}
	if err != nil {
		return StoredPhoto{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE photos SET position = position - 1 WHERE user_id = $1 AND position > $2`, userID, p.Position); err != nil {
		return StoredPhoto{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT 1 FROM profiles WHERE user_id = $1 FOR UPDATE`, userID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT id FROM photos WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	current := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		current[id] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) != len(current) {
		return ErrBadOrder
	}
	for _, id := range ids {
		if _, ok := current[id]; !ok {
			return ErrBadOrder
		}
		delete(current, id) // duplicates fail on the second sighting
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = o.pos - 1
		FROM unnest($2::uuid[]) WITH ORDINALITY AS o(id, pos)
		WHERE p.id = o.id AND p.user_id = $1
	`, userID, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]profiles.Photo, error) {
	rows, err := r.db.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []profiles.Photo{}
	for rows.Next() {
		var id string
		var pos int
		if err := rows.Scan(&id, &pos); err != nil {
			return nil, err
		}
		out = append(out, profiles.NewPhoto(id, pos))
	}
	return out, rows.Err()
}

// FileFor returns storage keys only if viewer owns the photo or may see its owner's profile.
func (r *PGRepository) FileFor(ctx context.Context, viewerID, photoID string) (string, string, time.Time, error) {
	var sk, tk string
	var created time.Time
	err := r.db.QueryRow(ctx, `
		SELECT ph.storage_key, ph.thumb_key, ph.created_at
		FROM photos ph
		WHERE ph.id = $1 AND (
			ph.user_id = $2
			OR EXISTS (
				SELECT 1 FROM profiles me, profiles t
				WHERE me.user_id = $2 AND t.user_id = ph.user_id AND `+profiles.VisibilitySQL+`
			)
		)
	`, photoID, viewerID).Scan(&sk, &tk, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", time.Time{}, ErrNotFound
	}
	return sk, tk, created, err
}
