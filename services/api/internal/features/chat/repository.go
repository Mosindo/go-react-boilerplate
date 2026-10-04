package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
	ErrClosed   = errors.New("conversation closed")
)

// Participant is who is in a conversation (used for realtime fan-out).
type Participant struct {
	ConversationID string
	MatchID        string
	UserIDs        [2]string
}

type Repository interface {
	GetSummary(ctx context.Context, userID, matchID string) (*MatchSummary, error)
	ListMatches(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]MatchSummary, error)
	ListConversations(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]MatchSummary, error)
	ListMessages(ctx context.Context, userID, conversationID string, limit int, cursor *pagination.Cursor) ([]Message, error)
	SendMessage(ctx context.Context, userID, conversationID, body string) (Message, Participant, error)
	MarkRead(ctx context.Context, userID, conversationID string) (readAt time.Time, changed bool, p Participant, err error)
	Hide(ctx context.Context, userID, conversationID string) error
	UnreadTotal(ctx context.Context, userID string) (int, error)
	Unmatch(ctx context.Context, userID, matchID string) (Participant, error)
}

type PGRepository struct{ db *pgxpool.Pool }

func NewPGRepository(db *pgxpool.Pool) *PGRepository { return &PGRepository{db: db} }

// summarySelect projects a conversation from the point of view of $1 (the viewer).
// cp = viewer's participant row, op = the other participant's row (read receipts).
const summarySelect = `
	SELECT m.id, c.id, m.created_at,
	       o.user_id, o.display_name, o.birth_date, ph.id, ph.position,
	       lm.id, lm.sender_id, lm.body, lm.created_at,
	       CASE WHEN lm.sender_id = $1 THEN
	              (CASE WHEN op.last_read_at >= lm.created_at THEN op.last_read_at END)
	            ELSE (CASE WHEN cp.last_read_at >= lm.created_at THEN cp.last_read_at END) END,
	       (SELECT COUNT(*) FROM messages x
	         WHERE x.conversation_id = c.id AND x.sender_id <> $1
	           AND x.created_at > cp.last_read_at
	           AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at)),
	       COALESCE(lm.created_at, m.created_at)
	FROM matches m
	JOIN conversations c ON c.match_id = m.id
	JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = $1
	JOIN conversation_participants op ON op.conversation_id = c.id AND op.user_id <> $1
	JOIN profiles o ON o.user_id = op.user_id
	JOIN users ou ON ou.id = o.user_id AND ou.status = 'active'
	LEFT JOIN photos ph ON ph.user_id = o.user_id AND ph.position = 0
	LEFT JOIN LATERAL (
	    SELECT id, sender_id, body, created_at FROM messages
	    WHERE conversation_id = c.id AND (cp.cleared_at IS NULL OR created_at > cp.cleared_at)
	    ORDER BY created_at DESC, id DESC LIMIT 1
	) lm ON TRUE
	WHERE m.unmatched_at IS NULL`

func scanSummary(row pgx.Row) (MatchSummary, error) {
	var s MatchSummary
	var birth time.Time
	var photoID *string
	var photoPos *int
	var lmID, lmSender, lmBody *string
	var lmAt, lmRead *time.Time
	err := row.Scan(&s.ID, &s.ConversationID, &s.MatchedAt,
		&s.User.UserID, &s.User.DisplayName, &birth, &photoID, &photoPos,
		&lmID, &lmSender, &lmBody, &lmAt, &lmRead, &s.UnreadCount, &s.UpdatedAt)
	if err != nil {
		return MatchSummary{}, err
	}
	s.User.Age = profiles.AgeAt(birth, time.Now().UTC())
	if photoID != nil && photoPos != nil {
		p := profiles.NewPhoto(*photoID, *photoPos)
		s.User.Photo = &p
	}
	if lmID != nil {
		s.LastMessage = &Message{ID: *lmID, ConversationID: s.ConversationID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmAt, ReadAt: lmRead}
	}
	return s, nil
}

func (r *PGRepository) GetSummary(ctx context.Context, userID, matchID string) (*MatchSummary, error) {
	s, err := scanSummary(r.db.QueryRow(ctx, summarySelect+` AND m.id = $2`, userID, matchID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PGRepository) collect(ctx context.Context, query string, args ...any) ([]MatchSummary, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MatchSummary{}
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PGRepository) ListMatches(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]MatchSummary, error) {
	return r.collect(ctx, summarySelect+`
		AND ($2::timestamptz IS NULL OR (m.created_at, m.id) < ($2, $3::uuid))
		ORDER BY m.created_at DESC, m.id DESC LIMIT $4`,
		userID, cursorTime(cursor), cursorID(cursor), limit)
}

// ListConversations returns conversations that have visible messages and are not hidden,
// ordered by latest activity. The cursor is (last_message_at, conversation id).
func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit int, cursor *pagination.Cursor) ([]MatchSummary, error) {
	return r.collect(ctx, summarySelect+`
		AND NOT cp.hidden AND lm.id IS NOT NULL
		AND ($2::timestamptz IS NULL OR (lm.created_at, c.id) < ($2, $3::uuid))
		ORDER BY lm.created_at DESC, c.id DESC LIMIT $4`,
		userID, cursorTime(cursor), cursorID(cursor), limit)
}

func (r *PGRepository) ListMessages(ctx context.Context, userID, conversationID string, limit int, cursor *pagination.Cursor) ([]Message, error) {
	var member bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2)
	`, conversationID, userID).Scan(&member); err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `
		SELECT x.id, x.conversation_id, x.sender_id, x.body, x.created_at,
		       CASE WHEN x.sender_id = $1 THEN (CASE WHEN op.last_read_at >= x.created_at THEN op.last_read_at END)
		            ELSE (CASE WHEN cp.last_read_at >= x.created_at THEN cp.last_read_at END) END
		FROM conversation_participants cp
		JOIN conversation_participants op ON op.conversation_id = cp.conversation_id AND op.user_id <> cp.user_id
		JOIN messages x ON x.conversation_id = cp.conversation_id
		WHERE cp.user_id = $1 AND cp.conversation_id = $2
		  AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at)
		  AND ($3::timestamptz IS NULL OR (x.created_at, x.id) < ($3, $4::uuid))
		ORDER BY x.created_at DESC, x.id DESC
		LIMIT $5
	`, userID, conversationID, cursorTime(cursor), cursorID(cursor), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// participant loads the conversation if userID takes part in it, whether or not it is still open.
func participant(ctx context.Context, q pgx.Tx, userID, conversationID string) (Participant, bool, error) {
	var p Participant
	var closed bool
	var a, b string
	err := q.QueryRow(ctx, `
		SELECT c.id, m.id, m.user_a, m.user_b, m.unmatched_at IS NOT NULL
		FROM conversations c JOIN matches m ON m.id = c.match_id
		WHERE c.id = $1 AND (m.user_a = $2 OR m.user_b = $2)
		FOR SHARE OF m
	`, conversationID, userID).Scan(&p.ConversationID, &p.MatchID, &a, &b, &closed)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participant{}, false, ErrNotFound
	}
	p.UserIDs = [2]string{a, b}
	return p, closed, err
}

func (r *PGRepository) SendMessage(ctx context.Context, userID, conversationID, body string) (Message, Participant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, Participant{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	p, closed, err := participant(ctx, tx, userID, conversationID)
	if err != nil {
		return Message{}, Participant{}, err
	}
	if closed {
		return Message{}, Participant{}, ErrClosed
	}
	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))
	`, p.UserIDs[0], p.UserIDs[1]).Scan(&blocked); err != nil {
		return Message{}, Participant{}, err
	}
	if blocked {
		return Message{}, Participant{}, ErrClosed
	}

	m := Message{ConversationID: conversationID, SenderID: userID, Body: body}
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, conversationID, userID, body).Scan(&m.ID, &m.CreatedAt); err != nil {
		return Message{}, Participant{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET last_message_at = $2 WHERE id = $1`, conversationID, m.CreatedAt); err != nil {
		return Message{}, Participant{}, err
	}
	// New activity resurfaces a conversation hidden by either side; the sender has by definition read up to now.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants
		SET hidden = FALSE, last_read_at = CASE WHEN user_id = $2 THEN $3 ELSE last_read_at END
		WHERE conversation_id = $1
	`, conversationID, userID, m.CreatedAt); err != nil {
		return Message{}, Participant{}, err
	}
	return m, p, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, bool, Participant, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return time.Time{}, false, Participant{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	p, _, err := participant(ctx, tx, userID, conversationID)
	if err != nil {
		return time.Time{}, false, Participant{}, err
	}
	var unread int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages x JOIN conversation_participants cp ON cp.conversation_id = x.conversation_id AND cp.user_id = $1
		WHERE x.conversation_id = $2 AND x.sender_id <> $1 AND x.created_at > cp.last_read_at
	`, userID, conversationID).Scan(&unread); err != nil {
		return time.Time{}, false, Participant{}, err
	}
	var readAt time.Time
	if err := tx.QueryRow(ctx, `
		UPDATE conversation_participants SET last_read_at = GREATEST(last_read_at, NOW())
		WHERE conversation_id = $1 AND user_id = $2 RETURNING last_read_at
	`, conversationID, userID).Scan(&readAt); err != nil {
		return time.Time{}, false, Participant{}, err
	}
	return readAt, unread > 0, p, tx.Commit(ctx)
}

func (r *PGRepository) Hide(ctx context.Context, userID, conversationID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants
		SET hidden = TRUE, cleared_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PGRepository) UnreadTotal(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id AND m.unmatched_at IS NULL
		JOIN messages x ON x.conversation_id = c.id AND x.sender_id <> cp.user_id AND x.created_at > cp.last_read_at
		WHERE cp.user_id = $1 AND (cp.cleared_at IS NULL OR x.created_at > cp.cleared_at)
	`, userID).Scan(&n)
	return n, err
}

func (r *PGRepository) Unmatch(ctx context.Context, userID, matchID string) (Participant, error) {
	var p Participant
	var a, b string
	err := r.db.QueryRow(ctx, `
		UPDATE matches SET unmatched_at = NOW(), unmatched_by = $1
		WHERE id = $2 AND unmatched_at IS NULL AND (user_a = $1 OR user_b = $1)
		RETURNING id, user_a, user_b, (SELECT id FROM conversations WHERE match_id = matches.id)
	`, userID, matchID).Scan(&p.MatchID, &a, &b, &p.ConversationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participant{}, ErrNotFound
	}
	p.UserIDs = [2]string{a, b}
	return p, err
}

func cursorTime(c *pagination.Cursor) any {
	if c == nil {
		return nil
	}
	return c.Time
}

func cursorID(c *pagination.Cursor) any {
	if c == nil {
		return nil
	}
	return c.ID
}
