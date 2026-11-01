package notifications

import (
	"context"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID string, ids []string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Notification, error) {
	var curAt *time.Time
	var curID *string
	if cur != nil {
		curAt, curID = &cur.At, &cur.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.actor_id, COALESCE(p.first_name, ''), n.ref_id, n.read_at, n.created_at
		FROM notifications n LEFT JOIN profiles p ON p.user_id = n.actor_id
		WHERE n.user_id = $1
		  AND ($3::timestamptz IS NULL OR (n.created_at, n.id) < ($3::timestamptz, $4::uuid))
		ORDER BY n.created_at DESC, n.id DESC LIMIT $2`, userID, limit, curAt, curID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.ActorName, &n.RefID, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
		return err
	}
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND id = ANY($2::uuid[]) AND read_at IS NULL`, userID, ids)
	return err
}
