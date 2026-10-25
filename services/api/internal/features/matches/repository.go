package matches

import (
	"context"
	"errors"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("match not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) list(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]row, error) {
	args := []any{userID, limit + 1}
	cursorSQL := ""
	if cur != nil {
		cursorSQL = ` AND (m.created_at, m.id) < ($3, $4::uuid)`
		args = append(args, cur.At, cur.ID)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, c.id, m.created_at, CASE WHEN m.user_a = $1::uuid THEN m.user_b ELSE m.user_a END
		FROM matches m
		JOIN conversations c ON c.match_id = m.id
		WHERE (m.user_a = $1::uuid OR m.user_b = $1::uuid)`+cursorSQL+`
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT $2
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		var rw row
		if err := rows.Scan(&rw.MatchID, &rw.ConversationID, &rw.CreatedAt, &rw.OtherUserID); err != nil {
			return nil, err
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

// remove deletes the match (and, by cascade, its conversation and messages) if the user is part of it.
func (r *Repository) remove(ctx context.Context, userID, matchID string) (row, error) {
	var rw row
	err := r.pool.QueryRow(ctx, `
		WITH target AS (
			SELECT m.id AS match_id, c.id AS conversation_id, m.created_at,
			       CASE WHEN m.user_a = $1::uuid THEN m.user_b ELSE m.user_a END AS other
			FROM matches m JOIN conversations c ON c.match_id = m.id
			WHERE m.id = $2::uuid AND (m.user_a = $1::uuid OR m.user_b = $1::uuid)
		), gone AS (
			DELETE FROM matches WHERE id IN (SELECT match_id FROM target)
		)
		SELECT match_id, conversation_id, created_at, other FROM target
	`, userID, matchID).Scan(&rw.MatchID, &rw.ConversationID, &rw.CreatedAt, &rw.OtherUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return row{}, ErrNotFound
	}
	return rw, err
}
