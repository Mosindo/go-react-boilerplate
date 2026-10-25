package safety

import (
	"context"
	"errors"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTargetNotFound = errors.New("user not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) userExists(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id string) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1::uuid)`, id).Scan(&ok)
	return ok, err
}

// block records the block and removes any match with the target. It returns the id of the deleted
// conversation (empty when they were not matched) so clients can be told.
func (r *Repository) block(ctx context.Context, blockerID, blockedID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if ok, err := r.userExists(ctx, tx, blockedID); err != nil {
		return "", err
	} else if !ok {
		return "", ErrTargetNotFound
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, blockerID, blockedID); err != nil {
		return "", err
	}
	var conversationID string
	err = tx.QueryRow(ctx, `
		WITH target AS (
			SELECT m.id AS match_id, c.id AS conversation_id FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.user_a = LEAST($1::uuid, $2::uuid) AND m.user_b = GREATEST($1::uuid, $2::uuid)
		), gone AS (DELETE FROM matches WHERE id IN (SELECT match_id FROM target))
		SELECT conversation_id FROM target
	`, blockerID, blockedID).Scan(&conversationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return conversationID, tx.Commit(ctx)
}

func (r *Repository) unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blocks WHERE blocker_id = $1 AND blocked_id = $2::uuid`, blockerID, blockedID)
	return err
}

func (r *Repository) listBlocked(ctx context.Context, blockerID string, limit int, cur *httpx.Cursor) ([]BlockedUser, error) {
	args := []any{blockerID, limit + 1}
	cursorSQL := ""
	if cur != nil {
		cursorSQL = ` AND (b.created_at, b.blocked_id) < ($3, $4::uuid)`
		args = append(args, cur.At, cur.ID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT b.blocked_id, COALESCE(p.first_name, ''), ph.id, b.created_at
		FROM blocks b
		LEFT JOIN profiles p ON p.user_id = b.blocked_id
		LEFT JOIN photos ph ON ph.user_id = b.blocked_id AND ph.position = 0
		WHERE b.blocker_id = $1::uuid`+cursorSQL+`
		ORDER BY b.created_at DESC, b.blocked_id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BlockedUser
	for rows.Next() {
		var b BlockedUser
		var photoID *string
		if err := rows.Scan(&b.UserID, &b.FirstName, &photoID, &b.CreatedAt); err != nil {
			return nil, err
		}
		if photoID != nil {
			thumb := "/photos/" + *photoID + "/image?size=thumb"
			b.ThumbURL = &thumb
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Repository) report(ctx context.Context, reporterID, reportedID, reason, details string) (string, error) {
	if ok, err := r.userExists(ctx, r.pool, reportedID); err != nil {
		return "", err
	} else if !ok {
		return "", ErrTargetNotFound
	}
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details) VALUES ($1, $2, $3, $4) RETURNING id
	`, reporterID, reportedID, reason, details).Scan(&id)
	return id, err
}
