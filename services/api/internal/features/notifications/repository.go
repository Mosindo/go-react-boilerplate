package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("notifications: not found")

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type Repository interface {
	Insert(ctx context.Context, userID, typ, actorID, refID string) error
	List(ctx context.Context, userID string, limit int, before *Cursor) ([]Notification, *Cursor, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Insert(ctx context.Context, userID, typ, actorID, refID string) error {
	if typ == TypeMessage {
		// One unread message notification per conversation: bump it instead of piling up rows.
		_, err := r.db.Exec(ctx, `
			INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, ref_id) WHERE read_at IS NULL AND type = 'message'
			DO UPDATE SET created_at = NOW(), actor_id = EXCLUDED.actor_id`, userID, typ, actorID, refID)
		return err
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id) VALUES ($1, $2, $3, $4)`, userID, typ, actorID, refID)
	return err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, before *Cursor) ([]Notification, *Cursor, error) {
	var bt *time.Time
	var bid *string
	if before != nil {
		bt, bid = &before.CreatedAt, &before.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT n.id, n.type, n.ref_id, n.read_at, n.created_at,
		       n.actor_id, COALESCE(p.first_name, ''), COALESCE(ph.id::text, '')
		FROM notifications n
		LEFT JOIN profiles p ON p.user_id = n.actor_id
		LEFT JOIN photos ph ON ph.user_id = n.actor_id AND ph.position = 0
		WHERE n.user_id = $1
		  AND ($3::timestamptz IS NULL OR (n.created_at, n.id) < ($3, $4::uuid))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $2`, userID, limit+1, bt, bid)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		var n Notification
		var ref, actorID *string
		var name, photo string
		if err := rows.Scan(&n.ID, &n.Type, &ref, &n.ReadAt, &n.CreatedAt, &actorID, &name, &photo); err != nil {
			return nil, nil, err
		}
		if ref != nil {
			n.RefID = *ref
		}
		if actorID != nil {
			n.Actor = &Actor{UserID: *actorID, FirstName: name, PhotoID: photo}
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *Cursor
	if len(out) > limit {
		out = out[:limit]
		last := out[len(out)-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return out, next, nil
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepoNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
