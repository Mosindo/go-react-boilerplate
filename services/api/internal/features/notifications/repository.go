package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func isUUIDError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// Upsert creates an unread notification, or refreshes the existing unread one for the same
// (user, type, reference) so repeated messages in one conversation coalesce.
func (r *Repository) Upsert(ctx context.Context, userID, kind, actorID, refID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, type, ref_id) WHERE read_at IS NULL
		DO UPDATE SET created_at = NOW(), actor_id = EXCLUDED.actor_id
		RETURNING id
	`, userID, kind, actorID, refID).Scan(&id)
	return id, err
}

func (r *Repository) MarkRefRead(ctx context.Context, userID, kind, refID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND type = $2 AND ref_id = $3 AND read_at IS NULL`, userID, kind, refID)
	return err
}

// List returns notifications newest first, hiding anything involving a blocked user.
func (r *Repository) List(ctx context.Context, userID string, limit int, before *time.Time) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.ref_id, n.actor_id, COALESCE(p.first_name, ''),
		       CASE WHEN ph.id IS NOT NULL THEN '/photos/' || ph.id || '/thumb' END,
		       n.read_at IS NOT NULL, n.created_at, n.read_at
		FROM notifications n
		LEFT JOIN profiles p ON p.user_id = n.actor_id
		LEFT JOIN photos ph ON ph.user_id = n.actor_id AND ph.position = 0
		WHERE n.user_id = $1 AND ($3::timestamptz IS NULL OR n.created_at < $3)
		  AND (n.actor_id IS NULL OR NOT EXISTS (
		        SELECT 1 FROM blocks b WHERE (b.blocker_id = $1 AND b.blocked_id = n.actor_id) OR (b.blocker_id = n.actor_id AND b.blocked_id = $1)))
		ORDER BY n.created_at DESC, n.id DESC LIMIT $2
	`, userID, limit, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.RefID, &n.ActorID, &n.ActorName, &n.ActorThumbURL, &n.IsRead, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead reports false when the notification does not belong to userID.
func (r *Repository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $2 AND user_id = $1`, userID, id)
	if err != nil {
		if isUUIDError(err) {
			return false, nil
		}
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
