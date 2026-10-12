package alerts

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID, beforeID string, limit int) ([]Stored, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID, beforeID string, limit int) ([]Stored, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, kind, actor_id, match_id, read_at, created_at
		FROM user_notifications
		WHERE user_id = $1::uuid
		  AND ($2 = '' OR (created_at, id) < (SELECT created_at, id FROM user_notifications WHERE id = NULLIF($2, '')::uuid AND user_id = $1::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, userID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Stored{}
	for rows.Next() {
		var s Stored
		if err := rows.Scan(&s.ID, &s.Kind, &s.ActorID, &s.MatchID, &s.ReadAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE user_notifications SET read_at = COALESCE(read_at, NOW())
		WHERE id = $2::uuid AND user_id = $1::uuid
	`, userID, id)
	return tag.RowsAffected() > 0, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE user_notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
