package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("notification not found")

// Execer is satisfied by *pgxpool.Pool and pgx.Tx.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Create inserts a notification using the caller's executor (usually its
// transaction). Unread "message" notifications are coalesced per conversation.
func Create(ctx context.Context, db Execer, n New) error {
	_, err := db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_user_id, match_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, match_id) WHERE type = 'message' AND read_at IS NULL DO NOTHING
	`, n.UserID, n.Type, n.ActorID, n.MatchID)
	return err
}

type row struct {
	Notification
	actorPhotoID *string
}

type Repository interface {
	List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]row, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]row, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.match_id, n.read_at, n.created_at,
		       p.user_id, p.first_name, ph.id
		FROM notifications n
		LEFT JOIN profiles p ON p.user_id = n.actor_user_id
		LEFT JOIN LATERAL (
		  SELECT id FROM photos WHERE user_id = n.actor_user_id ORDER BY position LIMIT 1
		) ph ON TRUE
		WHERE n.user_id = $1
		  AND ($2::timestamptz IS NULL OR (n.created_at, n.id) < ($2, $3::uuid))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $4
	`, userID, beforeAt, nullIfEmpty(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []row{}
	for rows.Next() {
		var rw row
		var actorID, actorName *string
		if err := rows.Scan(&rw.ID, &rw.Type, &rw.MatchID, &rw.ReadAt, &rw.CreatedAt, &actorID, &actorName, &rw.actorPhotoID); err != nil {
			return nil, err
		}
		if actorID != nil && actorName != nil {
			rw.Actor = &Actor{UserID: *actorID, FirstName: *actorName}
		}
		out = append(out, rw)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return ErrRepositoryNotFound
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
