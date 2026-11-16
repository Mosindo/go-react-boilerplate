package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("notification not found")

type Repository interface {
	List(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	FirstName(ctx context.Context, userID string) (string, error)
	CreateMatch(ctx context.Context, userID, actorID, matchID, title, body string) (Notification, error)
	UpsertMessage(ctx context.Context, userID, actorID, matchID, title, body string) (Notification, error)
	MarkRead(ctx context.Context, userID, id string) (Notification, error)
	MarkAllRead(ctx context.Context, userID string) error
	MarkMatchMessagesRead(ctx context.Context, userID, matchID string) error
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

const columns = `id, type, actor_id, match_id, title, body, is_read, created_at, updated_at, read_at`

func scan(row pgx.Row) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Type, &n.ActorID, &n.MatchID, &n.Title, &n.Body, &n.IsRead, &n.CreatedAt, &n.UpdatedAt, &n.ReadAt)
	return n, err
}

func (r *PGRepository) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+columns+` FROM notifications
		WHERE user_id = $1
		ORDER BY updated_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Notification, error) { return scan(row) })
}

func (r *PGRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND NOT is_read`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) FirstName(ctx context.Context, userID string) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT first_name FROM profiles WHERE user_id = $1`, userID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

func (r *PGRepository) CreateMatch(ctx context.Context, userID, actorID, matchID, title, body string) (Notification, error) {
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id, title, body)
		VALUES ($1, 'match', $2, $3, $4, $5)
		RETURNING `+columns, userID, actorID, matchID, title, body))
}

// UpsertMessage keeps a single unread "message" notification per conversation.
func (r *PGRepository) UpsertMessage(ctx context.Context, userID, actorID, matchID, title, body string) (Notification, error) {
	return scan(r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, match_id, title, body)
		VALUES ($1, 'message', $2, $3, $4, $5)
		ON CONFLICT (user_id, match_id) WHERE type = 'message' AND NOT is_read
		DO UPDATE SET title = EXCLUDED.title, body = EXCLUDED.body, updated_at = NOW()
		RETURNING `+columns, userID, actorID, matchID, title, body))
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	n, err := scan(r.db.QueryRow(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
		RETURNING `+columns, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	return n, err
}

func (r *PGRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND NOT is_read`, userID)
	return err
}

func (r *PGRepository) MarkMatchMessagesRead(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE, read_at = NOW()
		WHERE user_id = $1 AND match_id = $2 AND type = 'message' AND NOT is_read
	`, userID, matchID)
	return err
}
