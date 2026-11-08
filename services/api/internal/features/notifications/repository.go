package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("repository: not found")

type Repository interface {
	Create(ctx context.Context, userID, kind string, actorID, refID *string) (stored, bool, error)
	List(ctx context.Context, userID string, before *time.Time, limit int) ([]stored, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Create inserts a notification. Message notifications are coalesced: while one is still unread
// for the same conversation no new row is created (created=false), so a burst of messages does not
// flood the list.
func (r *PGRepository) Create(ctx context.Context, userID, kind string, actorID, refID *string) (stored, bool, error) {
	var s stored
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, ref_id)
		SELECT $1, $2, $3, $4
		WHERE $2 <> 'message' OR NOT EXISTS (
		  SELECT 1 FROM notifications n WHERE n.user_id = $1 AND n.type = 'message' AND n.ref_id IS NOT DISTINCT FROM $4::uuid AND n.read_at IS NULL)
		RETURNING id, type, actor_id, ref_id, read_at, created_at`, userID, kind, actorID, refID).
		Scan(&s.ID, &s.Type, &s.ActorID, &s.RefID, &s.ReadAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return stored{}, false, nil
	}
	return s, err == nil, err
}

func (r *PGRepository) List(ctx context.Context, userID string, before *time.Time, limit int) ([]stored, error) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id, n.type, n.actor_id, n.ref_id, n.read_at, n.created_at
		FROM notifications n
		WHERE n.user_id = $1
		  AND ($2::timestamptz IS NULL OR n.created_at < $2)
		  AND (n.actor_id IS NULL OR NOT EXISTS (
		        SELECT 1 FROM blocks b WHERE (b.blocker_id = n.user_id AND b.blocked_id = n.actor_id) OR (b.blocker_id = n.actor_id AND b.blocked_id = n.user_id)))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []stored
	for rows.Next() {
		var s stored
		if err := rows.Scan(&s.ID, &s.Type, &s.ActorID, &s.RefID, &s.ReadAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications n
		WHERE n.user_id = $1 AND n.read_at IS NULL
		  AND (n.actor_id IS NULL OR NOT EXISTS (
		        SELECT 1 FROM blocks b WHERE (b.blocker_id = n.user_id AND b.blocked_id = n.actor_id) OR (b.blocker_id = n.actor_id AND b.blocked_id = n.user_id)))`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
