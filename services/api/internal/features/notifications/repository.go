package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, typ, title, body string, data Data) (Notification, error)
	List(ctx context.Context, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const columns = `id, type, title, body, data, read_at, created_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	_ = json.Unmarshal(raw, &n.Data)
	return n, nil
}

// Create inserts a notification. Unread "message" notifications of one
// conversation are coalesced into a single rolling row.
func (r *PGRepository) Create(ctx context.Context, userID, typ, title, body string, data Data) (Notification, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Notification{}, err
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, (data->>'matchId')) WHERE type = 'message' AND read_at IS NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, created_at = NOW()
		RETURNING `+columns, userID, typ, title, body, raw))
}

func (r *PGRepository) List(ctx context.Context, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	if beforeTime != nil {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications
			WHERE user_id = $1 AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4
		`, userID, *beforeTime, beforeID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2
		`, userID, limit)
	}
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

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, id, userID)
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

// MarkConversationRead clears the rolling "new message" notification of a match.
func (r *PGRepository) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE user_id = $1 AND type = 'message' AND data->>'matchId' = $2 AND read_at IS NULL
	`, userID, matchID)
	return err
}
