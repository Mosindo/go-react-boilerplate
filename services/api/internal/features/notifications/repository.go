package notifications

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errRepoNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, userID, typ, title, body string, data json.RawMessage) (Notification, error)
	List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, notificationID string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

const columns = `id, type, title, body, data, read_at, created_at`

func scan(row pgx.CollectableRow) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Data, &n.ReadAt, &n.CreatedAt)
	return n, err
}

func (r *PGRepository) Create(ctx context.Context, userID, typ, title, body string, data json.RawMessage) (Notification, error) {
	rows, err := r.db.Query(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data) VALUES ($1, $2, $3, $4, $5)
		RETURNING `+columns, userID, typ, title, body, data)
	if err != nil {
		return Notification{}, err
	}
	return pgx.CollectExactlyOneRow(rows, scan)
}

// List returns notifications newest first using keyset pagination.
func (r *PGRepository) List(ctx context.Context, userID, beforeID string, limit int) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	if beforeID == "" {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications
			WHERE user_id = $1
			  AND (created_at, id) < (SELECT created_at, id FROM notifications WHERE id = $2 AND user_id = $1)
			ORDER BY created_at DESC, id DESC LIMIT $3`, userID, beforeID, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, scan)
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, notificationID, userID)
	if err != nil {
		return Notification{}, err
	}
	n, err := pgx.CollectExactlyOneRow(rows, scan)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, errRepoNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}
