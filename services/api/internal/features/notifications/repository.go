package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, n NewNotification) (Notification, error)
	// UpsertMessage keeps at most one unread "message" notification per conversation.
	UpsertMessage(ctx context.Context, n NewNotification, matchID string) (Notification, error)
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, matchID string) error
	UnreadCount(ctx context.Context, userID string) (int, error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id::text, type, title, body, data, read_at IS NOT NULL, created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	if err := json.Unmarshal(raw, &n.Data); err != nil || n.Data == nil {
		n.Data = map[string]any{}
	}
	return n, nil
}

func (r *PGRepository) Create(ctx context.Context, n NewNotification) (Notification, error) {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return Notification{}, err
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, n.UserID, n.Type, n.Title, n.Body, data))
}

func (r *PGRepository) UpsertMessage(ctx context.Context, n NewNotification, matchID string) (Notification, error) {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return Notification{}, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Notification{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Serialise per recipient+conversation so two quick messages cannot both insert.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 13))`, n.UserID+matchID); err != nil {
		return Notification{}, err
	}
	out, err := scan(tx.QueryRow(ctx, `
		UPDATE notifications SET title = $3, body = $4, data = $5, created_at = NOW()
		WHERE id = (
		  SELECT id FROM notifications
		  WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'matchId' = $2
		  ORDER BY created_at DESC LIMIT 1)
		RETURNING `+columns, n.UserID, matchID, n.Title, n.Body, data))
	if errors.Is(err, pgx.ErrNoRows) {
		out, err = scan(tx.QueryRow(ctx, `
			INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, 'message', $2, $3, $4)
			RETURNING `+columns, n.UserID, n.Title, n.Body, data))
	}
	if err != nil {
		return Notification{}, err
	}
	return out, tx.Commit(ctx)
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+` FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Notification{}
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW())
		WHERE id = $2 AND user_id = $1
		RETURNING `+columns, userID, notificationID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Notification{}, ErrNotFound
		}
		return Notification{}, err
	}
	return n, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND read_at IS NULL AND data->>'matchId' = $2
	`, userID, matchID)
	return err
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}
