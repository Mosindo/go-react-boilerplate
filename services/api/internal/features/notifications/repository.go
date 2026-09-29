package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectCols = `id, type, title, body, data, is_read, created_at`

func scan(row pgx.Row) (AppNotification, error) {
	var n AppNotification
	var raw []byte
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.IsRead, &n.CreatedAt); err != nil {
		return n, err
	}
	n.CreatedAt = n.CreatedAt.UTC()
	n.Data = map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &n.Data)
	}
	return n, nil
}

// Upsert persists a notification. Unread "message" notifications for the same conversation are
// coalesced into one row; an advisory transaction lock serialises concurrent writers per key.
func (r *Repository) Upsert(ctx context.Context, userID, typ, title, body string, data map[string]any) (AppNotification, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return AppNotification{}, err
	}
	convID, _ := data["conversationId"].(string)
	coalesce := typ == "message" && convID != ""

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AppNotification{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if coalesce {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, "notif:"+userID+":"+convID); err != nil {
			return AppNotification{}, err
		}
		n, err := scan(tx.QueryRow(ctx, `
			UPDATE notifications SET title = $3, body = $4, data = $5::jsonb, created_at = NOW()
			WHERE id = (
				SELECT id FROM notifications
				WHERE user_id = $1 AND type = 'message' AND is_read = FALSE AND data->>'conversationId' = $2
				ORDER BY created_at DESC, id DESC LIMIT 1
			)
			RETURNING `+selectCols, userID, convID, title, body, string(raw)))
		if err == nil {
			return n, tx.Commit(ctx)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return AppNotification{}, err
		}
	}
	n, err := scan(tx.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, is_read)
		VALUES ($1, $2, $3, $4, $5::jsonb, FALSE)
		RETURNING `+selectCols, userID, typ, title, body, string(raw)))
	if err != nil {
		return AppNotification{}, err
	}
	return n, tx.Commit(ctx)
}

func (r *Repository) List(ctx context.Context, userID string, before *time.Time, beforeID string, limit int) ([]AppNotification, int, error) {
	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case before == nil:
		rows, err = r.pool.Query(ctx, `SELECT `+selectCols+` FROM notifications
			WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	case beforeID == "":
		rows, err = r.pool.Query(ctx, `SELECT `+selectCols+` FROM notifications
			WHERE user_id = $1 AND created_at < $2 ORDER BY created_at DESC, id DESC LIMIT $3`, userID, *before, limit)
	default:
		rows, err = r.pool.Query(ctx, `SELECT `+selectCols+` FROM notifications
			WHERE user_id = $1 AND (created_at, id) < ($2, $3::uuid) ORDER BY created_at DESC, id DESC LIMIT $4`,
			userID, *before, beforeID, limit)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []AppNotification{}
	for rows.Next() {
		n, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var unread int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&unread); err != nil {
		return nil, 0, err
	}
	return out, unread, nil
}

// MarkRead marks one notification owned by userID; it reports false when none matches.
func (r *Repository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE user_id = $1 AND is_read = FALSE`, userID)
	return err
}
