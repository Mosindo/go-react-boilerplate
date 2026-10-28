package notifications

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, actorID, matchID, kind string) (*Notification, error)
	List(ctx context.Context, userID string, limit int, cursor *httpx.Cursor) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const selectColumns = `
	n.id, n.type, n.match_id, n.actor_id, COALESCE(p.first_name, ''), n.read_at, n.created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.ConversationID, &n.Actor.UserID, &n.Actor.FirstName, &n.ReadAt, &n.CreatedAt)
	return n, err
}

// Create inserts a notification. A "message" notification is skipped while an unread one already
// exists for the same conversation (no spam on message bursts); nil is returned in that case.
func (r *PGRepository) Create(ctx context.Context, userID, actorID, matchID, kind string) (*Notification, error) {
	row := r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO notifications (user_id, type, actor_id, match_id)
			SELECT $1, $4, $2, $3
			WHERE $4 <> 'message' OR NOT EXISTS (
				SELECT 1 FROM notifications
				WHERE user_id = $1 AND match_id = $3 AND type = 'message' AND read_at IS NULL)
			RETURNING *
		)
		SELECT `+selectColumns+` FROM ins n LEFT JOIN profiles p ON p.user_id = n.actor_id`, userID, actorID, matchID, kind)
	n, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, cursor *httpx.Cursor) ([]Notification, error) {
	var at *time.Time
	var id *string
	if cursor != nil {
		at, id = &cursor.At, &cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+selectColumns+`
		FROM notifications n LEFT JOIN profiles p ON p.user_id = n.actor_id
		WHERE n.user_id = $1
		  AND ($2::timestamptz IS NULL OR (n.created_at, n.id) < ($2, $3::uuid))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $4`, userID, at, id, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
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

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND match_id = $2 AND type = 'message' AND read_at IS NULL`, userID, matchID)
	return err
}
