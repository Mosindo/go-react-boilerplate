package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepoNotFound = errors.New("conversation not found")

type Repository interface {
	Access(ctx context.Context, userID, matchID string) (access, error)
	Conversations(ctx context.Context, userID string, matchID *string, limit int, cursor *httpx.Cursor) ([]Conversation, error)
	Messages(ctx context.Context, userID, matchID string, limit int, cursor *httpx.Cursor) ([]Message, error)
	InsertMessage(ctx context.Context, matchID, senderID, body string) (Message, error)
	MarkRead(ctx context.Context, userID, matchID string) (*time.Time, error)
	Clear(ctx context.Context, userID, matchID string) error
	Unmatch(ctx context.Context, userID, matchID string) (string, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// Access is the single authorisation gate: only participants get a row (non-members see "not found").
func (r *PGRepository) Access(ctx context.Context, userID, matchID string) (access, error) {
	var a access
	err := r.db.QueryRow(ctx, `
		SELECT peer.id,
		       EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = $1 AND b.blocked_id = peer.id),
		       EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = peer.id AND b.blocked_id = $1)
		FROM matches m
		CROSS JOIN LATERAL (SELECT CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END AS id) peer
		WHERE m.id = $2 AND (m.user_a = $1 OR m.user_b = $1)`, userID, matchID).
		Scan(&a.PeerID, &a.BlockedByMe, &a.BlockedByPeer)
	if errors.Is(err, pgx.ErrNoRows) {
		return access{}, ErrRepoNotFound
	}
	return a, err
}

func (r *PGRepository) Conversations(ctx context.Context, userID string, matchID *string, limit int, cursor *httpx.Cursor) ([]Conversation, error) {
	var curAt *time.Time
	var curID *string
	if cursor != nil {
		curAt, curID = &cursor.At, &cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.created_at, m.activity_at,
		       peer.user_id, peer.first_name, date_part('year', age(CURRENT_DATE, peer.birth_date))::int, ph.id,
		       lm.id, lm.sender_id, lm.body, lm.created_at, lm.read_at,
		       COALESCE(unread.n, 0)
		FROM match_participants mp
		JOIN matches m ON m.id = mp.match_id
		JOIN profiles peer ON peer.user_id = CASE WHEN m.user_a = mp.user_id THEN m.user_b ELSE m.user_a END
		LEFT JOIN photos ph ON ph.user_id = peer.user_id AND ph.position = 0
		LEFT JOIN LATERAL (
			SELECT id, sender_id, body, created_at, read_at FROM messages
			WHERE match_id = m.id AND (mp.cleared_at IS NULL OR created_at > mp.cleared_at)
			ORDER BY created_at DESC, id DESC LIMIT 1) lm ON TRUE
		LEFT JOIN LATERAL (
			SELECT count(*) AS n FROM messages
			WHERE match_id = m.id AND sender_id <> mp.user_id AND read_at IS NULL
			  AND (mp.cleared_at IS NULL OR created_at > mp.cleared_at)) unread ON TRUE
		WHERE mp.user_id = $1
		  AND ($2::uuid IS NULL OR m.id = $2)
		  AND (mp.cleared_at IS NULL OR lm.id IS NOT NULL)
		  AND NOT EXISTS (SELECT 1 FROM blocks b
		                  WHERE (b.blocker_id = $1 AND b.blocked_id = peer.user_id)
		                     OR (b.blocker_id = peer.user_id AND b.blocked_id = $1))
		  AND ($3::timestamptz IS NULL OR (m.activity_at, m.id) < ($3, $4::uuid))
		ORDER BY m.activity_at DESC, m.id DESC
		LIMIT $5`, userID, matchID, curAt, curID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Conversation, error) {
		var c Conversation
		var photoID *string
		var mid, msender, mbody *string
		var mat *time.Time
		var mread *time.Time
		err := row.Scan(&c.ID, &c.MatchedAt, &c.ActivityAt,
			&c.User.UserID, &c.User.FirstName, &c.User.Age, &photoID,
			&mid, &msender, &mbody, &mat, &mread, &c.UnreadCount)
		if err != nil {
			return c, err
		}
		if photoID != nil {
			url := "/photos/" + *photoID + "/file"
			c.User.PhotoURL = &url
		}
		if mid != nil {
			c.LastMessage = &Message{ID: *mid, ConversationID: c.ID, SenderID: *msender, Body: *mbody, CreatedAt: *mat, ReadAt: mread}
		}
		return c, nil
	})
}

func (r *PGRepository) Messages(ctx context.Context, userID, matchID string, limit int, cursor *httpx.Cursor) ([]Message, error) {
	var curAt *time.Time
	var curID *string
	if cursor != nil {
		curAt, curID = &cursor.At, &cursor.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT msg.id, msg.match_id, msg.sender_id, msg.body, msg.created_at, msg.read_at
		FROM messages msg
		JOIN match_participants mp ON mp.match_id = msg.match_id AND mp.user_id = $1
		WHERE msg.match_id = $2
		  AND (mp.cleared_at IS NULL OR msg.created_at > mp.cleared_at)
		  AND ($3::timestamptz IS NULL OR (msg.created_at, msg.id) < ($3, $4::uuid))
		ORDER BY msg.created_at DESC, msg.id DESC
		LIMIT $5`, userID, matchID, curAt, curID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Message, error) {
		var m Message
		err := row.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt)
		return m, err
	})
}

func (r *PGRepository) InsertMessage(ctx context.Context, matchID, senderID, body string) (Message, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (match_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, match_id, sender_id, body, created_at, read_at`, matchID, senderID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
		return Message{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET activity_at = $2 WHERE id = $1`, matchID, m.CreatedAt); err != nil {
		return Message{}, err
	}
	return m, tx.Commit(ctx)
}

// MarkRead flags the peer's unread messages; it returns the read time, or nil when nothing changed.
func (r *PGRepository) MarkRead(ctx context.Context, userID, matchID string) (*time.Time, error) {
	var readAt *time.Time
	err := r.db.QueryRow(ctx, `
		WITH upd AS (
			UPDATE messages SET read_at = clock_timestamp()
			WHERE match_id = $2 AND sender_id <> $1 AND read_at IS NULL
			RETURNING read_at)
		SELECT max(read_at) FROM upd`, userID, matchID).Scan(&readAt)
	return readAt, err
}

func (r *PGRepository) Clear(ctx context.Context, userID, matchID string) error {
	_, err := r.db.Exec(ctx, `UPDATE match_participants SET cleared_at = clock_timestamp() WHERE match_id = $2 AND user_id = $1`, userID, matchID)
	return err
}

// Unmatch deletes the match (messages, notifications and participants cascade) and returns the peer id.
// The swipes remain, so neither user is shown to the other again.
func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (string, error) {
	var peer string
	err := r.db.QueryRow(ctx, `
		DELETE FROM matches WHERE id = $2 AND (user_a = $1 OR user_b = $1)
		RETURNING CASE WHEN user_a = $1 THEN user_b ELSE user_a END`, userID, matchID).Scan(&peer)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrRepoNotFound
	}
	return peer, err
}
