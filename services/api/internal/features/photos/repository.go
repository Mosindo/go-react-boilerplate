package photos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	// Insert ajoute une photo a la premiere position libre. Si replaceID est non vide,
	// la photo remplacee est supprimee et sa position reutilisee; sa cle est renvoyee.
	Insert(ctx context.Context, userID string, f File, width, height, size int, replaceID string) (Photo, string, error)
	Delete(ctx context.Context, userID, photoID string) (string, error)
	Reorder(ctx context.Context, userID string, ids []string) error
	// FileForViewer renvoie le fichier si le viewer a le droit de le voir.
	FileForViewer(ctx context.Context, viewerID, photoID string) (File, error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, position, width, height, created_at FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position, &p.Width, &p.Height, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('photos:' || $1))`, userID)
	return err
}

func (r *PGRepository) Insert(ctx context.Context, userID string, f File, width, height, size int, replaceID string) (Photo, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Photo{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return Photo{}, "", err
	}

	oldKey := ""
	position := -1
	if replaceID != "" {
		err := tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING position, storage_key`,
			replaceID, userID).Scan(&position, &oldKey)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Photo{}, "", ErrNotFound
			}
			return Photo{}, "", err
		}
	} else {
		err := tx.QueryRow(ctx, `
			SELECT s FROM generate_series(0, $2 - 1) s
			WHERE NOT EXISTS (SELECT 1 FROM photos WHERE user_id = $1 AND position = s)
			ORDER BY s LIMIT 1`, userID, MaxPhotos).Scan(&position)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Photo{}, "", ErrLimitReached
			}
			return Photo{}, "", err
		}
	}

	var p Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, storage_key, content_type, width, height, size_bytes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, position, width, height, created_at`,
		userID, position, f.StorageKey, f.ContentType, width, height, size).
		Scan(&p.ID, &p.Position, &p.Width, &p.Height, &p.CreatedAt)
	if err != nil {
		return Photo{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Photo{}, "", err
	}
	return p, oldKey, nil
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return "", err
	}
	var key string
	err = tx.QueryRow(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2 RETURNING storage_key`, photoID, userID).Scan(&key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	// Compacte les positions pour garder la photo principale en position 0.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_unique DEFERRED`); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = r.rn
		FROM (SELECT id, (ROW_NUMBER() OVER (ORDER BY position) - 1)::smallint AS rn FROM photos WHERE user_id = $1) r
		WHERE p.id = r.id AND p.position <> r.rn`, userID); err != nil {
		return "", err
	}
	return key, tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockUser(ctx, tx, userID); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM photos WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrInvalidOrder
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position_unique DEFERRED`); err != nil {
		return err
	}
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, i)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrInvalidOrder // id inconnu, appartenant a un autre utilisateur ou duplique
		}
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) FileForViewer(ctx context.Context, viewerID, photoID string) (File, error) {
	var f File
	err := r.pool.QueryRow(ctx, `
		SELECT ph.storage_key, ph.content_type
		FROM photos ph
		JOIN preferences pr ON pr.user_id = ph.user_id
		WHERE ph.id = $2
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		        WHERE (b.blocker_id = $1 AND b.blocked_id = ph.user_id) OR (b.blocker_id = ph.user_id AND b.blocked_id = $1))
		  AND (ph.user_id = $1 OR pr.discoverable OR EXISTS (
		        SELECT 1 FROM matches m
		        WHERE m.user_a = LEAST($1::uuid, ph.user_id) AND m.user_b = GREATEST($1::uuid, ph.user_id)))
	`, viewerID, photoID).Scan(&f.StorageKey, &f.ContentType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return File{}, ErrNotFound
		}
		return File{}, err
	}
	return f, nil
}
