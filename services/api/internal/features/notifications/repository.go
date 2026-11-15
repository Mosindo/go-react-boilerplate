package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Upsert(ctx context.Context, in Input) (Notification, error)
	List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkGroupRead(ctx context.Context, userID, groupKey string) error
	Badges(ctx context.Context, userID string) (Badges, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const notificationColumns = `id, type, title, body, data, read_at, created_at`

func scanNotification(row pgx.Row) (Notification, error) {
	var n Notification
	var data []byte
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &data, &n.ReadAt, &n.CreatedAt)
	n.Data = json.RawMessage(data)
	return n, err
}

func (r *PGRepository) Upsert(ctx context.Context, in Input) (Notification, error) {
	data, err := json.Marshal(in.Data)
	if err != nil {
		return Notification{}, err
	}
	var groupKey any
	if in.GroupKey != "" {
		groupKey = in.GroupKey
	}
	return scanNotification(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, group_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, group_key) WHERE read_at IS NULL AND group_key IS NOT NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, data = EXCLUDED.data, created_at = NOW()
		RETURNING `+notificationColumns, in.UserID, in.Type, in.Title, in.Body, data, groupKey))
}

func (r *PGRepository) List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	if cursor == nil {
		rows, err = r.db.Query(ctx, `
			SELECT `+notificationColumns+` FROM notifications
			WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2
		`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT `+notificationColumns+` FROM notifications
			WHERE user_id = $1 AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4
		`, userID, cursor.At, cursor.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scanNotification(row) })
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) error {
	var id string
	err := r.db.QueryRow(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING id
	`, notificationID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkGroupRead(ctx context.Context, userID, groupKey string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND group_key = $2 AND read_at IS NULL`, userID, groupKey)
	return err
}

// Badges powers the tab-bar counters in one round trip.
func (r *PGRepository) Badges(ctx context.Context, userID string) (Badges, error) {
	var b Badges
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL)::int,
		  (SELECT COUNT(*) FROM conversation_participants me
		     JOIN conversations c ON c.id = me.conversation_id
		    WHERE me.user_id = $1
		      AND c.last_message_at > COALESCE(me.last_read_at, '-infinity')
		      AND EXISTS (SELECT 1 FROM messages m WHERE m.conversation_id = c.id AND m.sender_id <> $1
		                    AND m.created_at > COALESCE(me.last_read_at, '-infinity')))::int,
		  (SELECT COUNT(*) FROM swipes s JOIN profiles p ON p.user_id = s.swiper_id
		    WHERE s.target_id = $1 AND s.action = 'like' AND p.is_paused = FALSE
		      AND `+profiles.CompleteProfileSQL+`
		      AND NOT EXISTS (SELECT 1 FROM swipes mine WHERE mine.swiper_id = $1 AND mine.target_id = s.swiper_id)
		      AND NOT EXISTS (SELECT 1 FROM blocks bl WHERE (bl.blocker_id = $1 AND bl.blocked_id = s.swiper_id)
		                                              OR (bl.blocker_id = s.swiper_id AND bl.blocked_id = $1)))::int
	`, userID).Scan(&b.UnreadNotifications, &b.UnreadConversations, &b.LikesReceived)
	return b, err
}
