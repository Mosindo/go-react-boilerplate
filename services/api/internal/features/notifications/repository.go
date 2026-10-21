package notifications

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type storedNotification struct {
	ID, Type, ActorID, ActorName string
	ActorPhotoID                 *string
	ConversationID               *string
	ReadAt                       *time.Time
	CreatedAt                    time.Time
}

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]storedNotification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID string, ids []string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]storedNotification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id, n.type, n.actor_id, p.first_name,
		       (SELECT id FROM photos WHERE user_id = n.actor_id AND position = 0),
		       CASE WHEN n.type = 'match' THEN (SELECT c.id::text FROM conversations c WHERE c.match_id = n.ref_id) ELSE n.ref_id::text END,
		       n.read_at, n.created_at
		FROM notifications n
		JOIN profiles p ON p.user_id = n.actor_id
		WHERE n.user_id = $1 AND ($2::timestamptz IS NULL OR n.created_at < $2)
		ORDER BY n.created_at DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedNotification
	for rows.Next() {
		var n storedNotification
		if err := rows.Scan(&n.ID, &n.Type, &n.ActorID, &n.ActorName, &n.ActorPhotoID, &n.ConversationID, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead marks the given notifications (or all when ids is empty) as read; only the caller's rows are touched.
func (r *PGRepository) MarkRead(ctx context.Context, userID string, ids []string) error {
	if len(ids) == 0 {
		_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND id = ANY($2::uuid[]) AND read_at IS NULL`, userID, ids)
	return err
}
