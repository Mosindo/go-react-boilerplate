package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("conversation not found")

type Repository interface {
	Member(ctx context.Context, matchID, userID string) (Member, error)
	Insert(ctx context.Context, matchID, senderID, content string) (Message, error)
	List(ctx context.Context, matchID, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Message, error)
	MarkRead(ctx context.Context, matchID, readerID string) (count int, readAt time.Time, otherID string, err error)
}

type PGRepository struct {
	db *pgxpool.Pool
}

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

func (r *PGRepository) Member(ctx context.Context, matchID, userID string) (Member, error) {
	var m Member
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN mt.user_a = $2 THEN mt.user_b ELSE mt.user_a END,
		       COALESCE((SELECT first_name FROM profiles WHERE user_id = $2), '')
		FROM matches mt
		WHERE mt.id = $1 AND (mt.user_a = $2 OR mt.user_b = $2) AND mt.unmatched_at IS NULL
	`, matchID, userID).Scan(&m.OtherUserID, &m.SelfName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrRepositoryNotFound
	}
	return m, err
}

// Insert stores a message only if the sender belongs to an active match; the
// membership test and the insert are one statement, so an unmatch cannot race it.
func (r *PGRepository) Insert(ctx context.Context, matchID, senderID, content string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var m Message
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, content)
		SELECT mt.id, $2, $3 FROM matches mt
		WHERE mt.id = $1 AND (mt.user_a = $2 OR mt.user_b = $2) AND mt.unmatched_at IS NULL
		RETURNING id, match_id, sender_id, content, created_at, read_at
	`, matchID, senderID, content).Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Message{}, ErrRepositoryNotFound
	}
	if err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET last_activity_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

// List returns messages newest first. Membership and the user's "hide" cutoff are enforced in SQL.
func (r *PGRepository) List(ctx context.Context, matchID, userID string, limit int, beforeTime *time.Time, beforeID string) ([]Message, error) {
	const base = `
		SELECT x.id, x.match_id, x.sender_id, x.content, x.created_at, x.read_at
		FROM messages x JOIN matches mt ON mt.id = x.match_id
		WHERE mt.id = $1 AND (mt.user_a = $2 OR mt.user_b = $2) AND mt.unmatched_at IS NULL
		  AND x.created_at > COALESCE(CASE WHEN mt.user_a = $2 THEN mt.a_hidden_at ELSE mt.b_hidden_at END, '-infinity')`
	var rows pgx.Rows
	var err error
	if beforeTime != nil {
		rows, err = r.db.Query(ctx, base+` AND (x.created_at, x.id) < ($3, $4::uuid)
			ORDER BY x.created_at DESC, x.id DESC LIMIT $5`, matchID, userID, *beforeTime, beforeID, limit)
	} else {
		rows, err = r.db.Query(ctx, base+` ORDER BY x.created_at DESC, x.id DESC LIMIT $3`, matchID, userID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.MatchID, &m.SenderID, &m.Content, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PGRepository) MarkRead(ctx context.Context, matchID, readerID string) (int, time.Time, string, error) {
	var other string
	err := r.db.QueryRow(ctx, `
		SELECT CASE WHEN user_a = $2 THEN user_b ELSE user_a END FROM matches
		WHERE id = $1 AND (user_a = $2 OR user_b = $2) AND unmatched_at IS NULL
	`, matchID, readerID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, "", ErrRepositoryNotFound
	}
	if err != nil {
		return 0, time.Time{}, "", err
	}
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx, `
		UPDATE messages SET read_at = $3 WHERE match_id = $1 AND sender_id <> $2 AND read_at IS NULL
	`, matchID, readerID, now)
	if err != nil {
		return 0, time.Time{}, "", err
	}
	return int(tag.RowsAffected()), now, other, nil
}
