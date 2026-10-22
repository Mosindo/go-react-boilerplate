package notifications

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.conversation_id, n.created_at, n.read_at,
		       a.user_id, a.first_name, ph.id, ph.position, ph.width, ph.height, left(md5(ph.storage_key), 8)
		FROM notifications n
		LEFT JOIN profiles a ON a.user_id = n.actor_user_id
		LEFT JOIN photos ph ON ph.user_id = a.user_id AND ph.position = 0
		WHERE n.user_id = $1
		  AND ($2::timestamptz IS NULL OR n.created_at < $2)
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $3
	`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Notification, 0, limit)
	for rows.Next() {
		var n Notification
		var conversationID, actorID, actorName, photoID, version *string
		var pos, w, h *int
		if err := rows.Scan(&n.ID, &n.Type, &conversationID, &n.CreatedAt, &n.ReadAt,
			&actorID, &actorName, &photoID, &pos, &w, &h, &version); err != nil {
			return nil, err
		}
		if conversationID != nil {
			n.ConversationID = *conversationID
		}
		if actorID != nil {
			n.Actor = &Actor{UserID: *actorID, FirstName: *actorName}
			if photoID != nil {
				n.Actor.Photo = &profiles.PhotoRef{ID: *photoID, Position: *pos, Width: *w, Height: *h, Version: *version}
			}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, notificationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
