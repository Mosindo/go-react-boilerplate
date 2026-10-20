package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Upsert(ctx context.Context, userID, kind, actorID string, data map[string]string) (Notification, error)
	List(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const selectColumns = `
	n.id::text, n.type, n.actor_id::text, pf.first_name,
	CASE WHEN ph.id IS NULL THEN NULL ELSE '/photos/' || ph.id::text || '/file' END,
	n.data, n.read_at, n.created_at`

const selectJoins = `
	FROM notifications n
	LEFT JOIN profiles pf ON pf.user_id = n.actor_id
	LEFT JOIN photos ph ON ph.user_id = n.actor_id AND ph.position = 0`

type scanner interface{ Scan(dest ...any) error }

func scanNotification(row scanner) (Notification, error) {
	var (
		n         Notification
		actorID   *string
		firstName *string
		photoURL  *string
		raw       []byte
	)
	if err := row.Scan(&n.ID, &n.Type, &actorID, &firstName, &photoURL, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	if actorID != nil {
		n.Actor = &profiles.UserSummary{UserID: *actorID, PhotoURL: photoURL}
		if firstName != nil {
			n.Actor.FirstName = *firstName
		}
	}
	n.Data = map[string]string{}
	if err := json.Unmarshal(raw, &n.Data); err != nil {
		return Notification{}, err
	}
	return n, nil
}

// Upsert inserts a notification. Message notifications are coalesced: while
// one is unread for a conversation, a new message only refreshes it.
func (r *PGRepository) Upsert(ctx context.Context, userID, kind, actorID string, data map[string]string) (Notification, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	var id string
	err = r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, data) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, (data->>'conversationId')) WHERE type = 'message' AND read_at IS NULL
		DO UPDATE SET created_at = NOW(), actor_id = EXCLUDED.actor_id
		RETURNING id::text
	`, userID, kind, actorID, payload).Scan(&id)
	if err != nil {
		return Notification{}, err
	}
	return scanNotification(r.db.QueryRow(ctx, `SELECT `+selectColumns+selectJoins+` WHERE n.id = $1`, id))
}

func (r *PGRepository) List(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) ([]Notification, error) {
	var at *time.Time
	cursorID := "00000000-0000-0000-0000-000000000000"
	if cursor != nil {
		at = &cursor.At
		cursorID = cursor.ID
	}
	rows, err := r.db.Query(ctx, `SELECT `+selectColumns+selectJoins+`
		WHERE n.user_id = $1 AND ($2::timestamptz IS NULL OR (n.created_at, n.id) < ($2::timestamptz, $3::uuid))
		ORDER BY n.created_at DESC, n.id DESC LIMIT $4`, userID, at, cursorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&count)
	return count, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'conversationId' = $2`, userID, conversationID)
	return err
}
