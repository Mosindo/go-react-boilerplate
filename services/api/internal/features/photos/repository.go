package photos

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string) ([]Photo, error)
	Add(ctx context.Context, userID string, data []byte, w, h int) (Photo, error)
	Replace(ctx context.Context, userID, photoID string, data []byte, w, h int) (Photo, error)
	Delete(ctx context.Context, userID, photoID string) error
	Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error)
	Content(ctx context.Context, viewerID, photoID string) (Content, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func list(ctx context.Context, q querier, userID string) ([]Photo, error) {
	rows, err := q.Query(ctx, `SELECT id, position FROM photos WHERE user_id = $1 ORDER BY position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Photo{}
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PGRepository) List(ctx context.Context, userID string) ([]Photo, error) {
	return list(ctx, r.db, userID)
}

// begin opens a transaction serialized per user (advisory lock) so concurrent
// uploads cannot race on positions.
func (r *PGRepository) begin(ctx context.Context, userID string) (pgx.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "photos:"+userID); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func (r *PGRepository) Add(ctx context.Context, userID string, data []byte, w, h int) (Photo, error) {
	tx, err := r.begin(ctx, userID)
	if err != nil {
		return Photo{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var hasProfile bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM profiles WHERE user_id = $1)`, userID).Scan(&hasProfile); err != nil {
		return Photo{}, err
	}
	if !hasProfile {
		return Photo{}, ErrNeedProfile
	}
	existing, err := list(ctx, tx, userID)
	if err != nil {
		return Photo{}, err
	}
	if len(existing) >= MaxPhotos {
		return Photo{}, ErrTooMany
	}
	used := map[int]bool{}
	for _, p := range existing {
		used[p.Position] = true
	}
	pos := 0
	for used[pos] {
		pos++
	}
	var p Photo
	err = tx.QueryRow(ctx, `
		INSERT INTO photos (user_id, position, content_type, width, height, data)
		VALUES ($1, $2, 'image/jpeg', $3, $4, $5) RETURNING id, position`,
		userID, pos, w, h, data).Scan(&p.ID, &p.Position)
	if err != nil {
		return Photo{}, err
	}
	return p, tx.Commit(ctx)
}

func (r *PGRepository) Replace(ctx context.Context, userID, photoID string, data []byte, w, h int) (Photo, error) {
	var p Photo
	err := r.db.QueryRow(ctx, `
		UPDATE photos SET data = $3, width = $4, height = $5, created_at = NOW()
		WHERE id = $1 AND user_id = $2 RETURNING id, position`,
		photoID, userID, data, w, h).Scan(&p.ID, &p.Position)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return Photo{}, ErrNotFound
	}
	return p, err
}

func (r *PGRepository) Delete(ctx context.Context, userID, photoID string) error {
	tx, err := r.begin(ctx, userID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	tag, err := tx.Exec(ctx, `DELETE FROM photos WHERE id = $1 AND user_id = $2`, photoID, userID)
	if isInvalidUUID(err) || (err == nil && tag.RowsAffected() == 0) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	// Close the gap so the main photo is always position 0.
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE photos p SET position = n.rn - 1
		FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY position) AS rn FROM photos WHERE user_id = $1) n
		WHERE p.id = n.id`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PGRepository) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	tx, err := r.begin(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	existing, err := list(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) != len(existing) {
		return nil, ErrInvalidOrder
	}
	have := make([]string, len(existing))
	for i, p := range existing {
		have[i] = p.ID
	}
	want := append([]string(nil), ids...)
	sort.Strings(have)
	sort.Strings(want)
	for i := range have {
		if have[i] != want[i] {
			return nil, ErrInvalidOrder
		}
	}
	if _, err := tx.Exec(ctx, `SET CONSTRAINTS photos_user_position DEFERRED`); err != nil {
		return nil, err
	}
	for pos, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE photos SET position = $3 WHERE id = $1 AND user_id = $2`, id, userID, pos); err != nil {
			return nil, err
		}
	}
	out, err := list(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// Content enforces visibility: the owner always; anyone else only when no
// block exists in either direction and the owner has a visible profile
// (discoverable, or matched with the viewer).
func (r *PGRepository) Content(ctx context.Context, viewerID, photoID string) (Content, error) {
	var c Content
	err := r.db.QueryRow(ctx, `
		SELECT ph.content_type, ph.data
		FROM photos ph
		JOIN profiles pr ON pr.user_id = ph.user_id
		WHERE ph.id = $1
		  AND (
		    ph.user_id = $2
		    OR (
		      NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $2 AND b.blocked_id = ph.user_id)
		                     OR (b.blocker_id = ph.user_id AND b.blocked_id = $2))
		      AND (pr.is_discoverable OR EXISTS (
		            SELECT 1 FROM matches m
		            WHERE m.ended_at IS NULL
		              AND m.user_a = LEAST($2::uuid, ph.user_id) AND m.user_b = GREATEST($2::uuid, ph.user_id)))
		    )
		  )`, photoID, viewerID).Scan(&c.ContentType, &c.Data)
	if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
		return Content{}, ErrNotFound
	}
	return c, err
}
