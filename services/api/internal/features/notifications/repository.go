package notifications

import (
	"context"
	"encoding/json"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Upsert(ctx context.Context, in Input) (Notification, error)
	List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) (bool, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkGroupRead(ctx context.Context, userID, groupKey string) error
	DeleteGroups(ctx context.Context, groupKeys []string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository {
	return &PGRepository{db: db}
}

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	var raw []byte
	var readAt *time.Time
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &readAt, &n.CreatedAt); err != nil {
		return Notification{}, err
	}
	n.IsRead = readAt != nil
	n.Data = map[string]any{}
	_ = json.Unmarshal(raw, &n.Data)
	return n, nil
}

const columns = `id, type, title, body, data, read_at, created_at`

func (r *PGRepository) Upsert(ctx context.Context, in Input) (Notification, error) {
	data, err := json.Marshal(in.Data)
	if err != nil {
		return Notification{}, err
	}
	var groupKey any
	if in.GroupKey != "" {
		groupKey = in.GroupKey
	}
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data, group_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, group_key) WHERE read_at IS NULL AND group_key IS NOT NULL
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, data = EXCLUDED.data, created_at = NOW()
		RETURNING `+columns, in.UserID, in.Type, in.Title, in.Body, data, groupKey))
}

func (r *PGRepository) List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]Notification, error) {
	var rows pgx.Rows
	var err error
	if cursor == nil {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC, id DESC LIMIT $2
		`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT `+columns+` FROM notifications WHERE user_id = $1 AND (created_at, id) < ($2, $3::uuid)
			ORDER BY created_at DESC, id DESC LIMIT $4
		`, userID, cursor.Time, cursor.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, NOW()) WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

func (r *PGRepository) MarkGroupRead(ctx context.Context, userID, groupKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND group_key = $2 AND read_at IS NULL
	`, userID, groupKey)
	return err
}

func (r *PGRepository) DeleteGroups(ctx context.Context, groupKeys []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notifications WHERE group_key = ANY($1)`, groupKeys)
	return err
}
