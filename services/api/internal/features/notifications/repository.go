package notifications

import (
	"context"
	"errors"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `id, type, actor_id, conversation_id, read_at, created_at`

func scanRow(src pgx.Row) (row, error) {
	var n row
	err := src.Scan(&n.ID, &n.Type, &n.ActorID, &n.ConversationID, &n.ReadAt, &n.CreatedAt)
	return n, err
}

func (r *Repository) list(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]row, int, error) {
	args := []any{userID, limit + 1}
	cursorSQL := ""
	if cur != nil {
		cursorSQL = ` AND (created_at, id) < ($3, $4::uuid)`
		args = append(args, cur.At, cur.ID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1::uuid`+cursorSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		n, err := scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var unread int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&unread); err != nil {
		return nil, 0, err
	}
	return out, unread, nil
}

func (r *Repository) get(ctx context.Context, userID, id string) (row, error) {
	n, err := scanRow(r.pool.QueryRow(ctx, `SELECT `+columns+` FROM notifications WHERE id = $1::uuid AND user_id = $2::uuid`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return row{}, ErrNotFound
	}
	return n, err
}

func (r *Repository) markRead(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1::uuid AND user_id = $2::uuid
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) markAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
