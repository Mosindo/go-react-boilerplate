package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

// InsertTx records a notification inside the caller's transaction. Message
// notifications are coalesced: while one is still unread for a conversation no
// further row is added, so a burst of messages produces a single entry.
// It reports whether a row was created.
func InsertTx(ctx context.Context, tx pgx.Tx, userID, kind, actorID, conversationID string) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, conversation_id)
		SELECT $1, $2, $3, $4
		WHERE $2 <> 'new_message' OR NOT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = 'new_message' AND conversation_id = $4 AND read_at IS NULL)
	`, userID, kind, actorID, conversationID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, beforeAt *time.Time, beforeID string) ([]Notification, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.type, n.conversation_id, n.read_at, n.created_at,
		       a.user_id, a.first_name,
		       (SELECT ph.id FROM photos ph WHERE ph.user_id = a.user_id ORDER BY ph.position LIMIT 1)
		FROM notifications n
		LEFT JOIN profiles a ON a.user_id = n.actor_id
		WHERE n.user_id = $1
		  AND ($2::timestamptz IS NULL OR (n.created_at, n.id) < ($2, $3::uuid))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $4
	`, userID, beforeAt, nullUUID(beforeID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var n Notification
		var actorID, actorName *string
		var photoID *string
		if err := rows.Scan(&n.ID, &n.Type, &n.ConversationID, &n.ReadAt, &n.CreatedAt, &actorID, &actorName, &photoID); err != nil {
			return nil, err
		}
		if actorID != nil && actorName != nil {
			n.Actor = &Actor{UserID: *actorID, FirstName: *actorName, PhotoID: photoID}
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func nullUUID(s string) any {
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
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return ErrNotFound
		}
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
