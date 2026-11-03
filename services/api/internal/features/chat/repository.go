package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotParticipant is returned for both unknown conversations and
// conversations the user does not belong to (no existence leak).
var ErrNotParticipant = errors.New("not a participant")

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

func (r *Repository) ListConversations(ctx context.Context, userID string, before time.Time, limit int) ([]Conversation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profiles.CardColumns+`,
			c.id, c.match_id,
			lm.id, lm.sender_id, lm.body, lm.created_at,
			(SELECT COUNT(*) FROM messages x
			   WHERE x.conversation_id = c.id AND x.sender_id <> $1
			     AND x.id > GREATEST(cp.last_read_message_id, cp.cleared_up_to_id)),
			COALESCE(lm.created_at, c.created_at)
		FROM conversation_participants cp
		JOIN conversations c ON c.id = cp.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN profiles p ON p.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		LEFT JOIN profiles v ON v.user_id = $1
		LEFT JOIN LATERAL (
			SELECT x.id, x.sender_id, x.body, x.created_at FROM messages x
			WHERE x.conversation_id = c.id AND x.id > cp.cleared_up_to_id
			ORDER BY x.id DESC LIMIT 1) lm ON TRUE
		WHERE cp.user_id = $1 AND NOT cp.hidden
		  AND COALESCE(lm.created_at, c.created_at) < $2
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id
		LIMIT $3`, userID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conversation{}
	for rows.Next() {
		var (
			conv      Conversation
			lmID      *int64
			lmSender  *string
			lmBody    *string
			lmCreated *time.Time
		)
		card, err := profiles.ScanCard(rows, &conv.ID, &conv.MatchID, &lmID, &lmSender, &lmBody, &lmCreated, &conv.UnreadCount, &conv.ActivityAt)
		if err != nil {
			return nil, err
		}
		conv.User = card
		if lmID != nil {
			conv.LastMessage = &Message{ID: *lmID, ConversationID: conv.ID, SenderID: *lmSender, Body: *lmBody, CreatedAt: *lmCreated}
		}
		out = append(out, conv)
	}
	return out, rows.Err()
}

// participants returns the other user's id if userID belongs to the conversation.
func participantOther(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, convID, userID string) (string, error) {
	var other string
	err := q.QueryRow(ctx, `
		SELECT o.user_id
		FROM conversation_participants me
		JOIN conversation_participants o ON o.conversation_id = me.conversation_id AND o.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2`, convID, userID).Scan(&other)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotParticipant
	}
	return other, err
}

func (r *Repository) Messages(ctx context.Context, convID, userID string, beforeID int64, limit int) (MessagesPage, error) {
	other, err := participantOther(ctx, r.db, convID, userID)
	if err != nil {
		return MessagesPage{}, err
	}
	page := MessagesPage{Messages: []Message{}}
	if err := r.db.QueryRow(ctx, `SELECT last_read_message_id FROM conversation_participants WHERE conversation_id = $1 AND user_id = $2`,
		convID, other).Scan(&page.OtherLastReadMessageID); err != nil {
		return MessagesPage{}, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT x.id, x.conversation_id, x.sender_id, x.body, x.created_at
		FROM messages x
		JOIN conversation_participants cp ON cp.conversation_id = x.conversation_id AND cp.user_id = $2
		WHERE x.conversation_id = $1 AND x.id > cp.cleared_up_to_id AND x.id < $3
		ORDER BY x.id DESC LIMIT $4`, convID, userID, beforeID, limit)
	if err != nil {
		return MessagesPage{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return MessagesPage{}, err
		}
		page.Messages = append(page.Messages, m)
	}
	// Return the page oldest-first so clients can render it directly.
	for i, j := 0, len(page.Messages)-1; i < j; i, j = i+1, j-1 {
		page.Messages[i], page.Messages[j] = page.Messages[j], page.Messages[i]
	}
	return page, rows.Err()
}

// Send stores a message; it re-shows the conversation for both sides.
func (r *Repository) Send(ctx context.Context, convID, senderID, body string) (Message, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Message{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	other, err := participantOther(ctx, tx, convID, senderID)
	if err != nil {
		return Message{}, "", err
	}
	var blocked bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))`,
		senderID, other).Scan(&blocked); err != nil {
		return Message{}, "", err
	}
	if blocked {
		return Message{}, "", ErrNotParticipant
	}
	var m Message
	if err := tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at`, convID, senderID, body).
		Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
		return Message{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversation_participants SET hidden = FALSE WHERE conversation_id = $1 AND hidden`, convID); err != nil {
		return Message{}, "", err
	}
	// The sender has implicitly read everything up to their own message.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET last_read_message_id = GREATEST(last_read_message_id, $3)
		WHERE conversation_id = $1 AND user_id = $2`, convID, senderID, m.ID); err != nil {
		return Message{}, "", err
	}
	return m, other, tx.Commit(ctx)
}

// MarkRead advances the read marker to the latest message and returns it.
func (r *Repository) MarkRead(ctx context.Context, convID, userID string) (int64, string, error) {
	other, err := participantOther(ctx, r.db, convID, userID)
	if err != nil {
		return 0, "", err
	}
	var upTo int64
	err = r.db.QueryRow(ctx, `
		UPDATE conversation_participants cp
		SET last_read_message_id = GREATEST(cp.last_read_message_id,
			COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = $1), 0))
		WHERE cp.conversation_id = $1 AND cp.user_id = $2
		RETURNING cp.last_read_message_id`, convID, userID).Scan(&upTo)
	return upTo, other, err
}

// Hide removes the conversation from the user's list and clears their local
// history. The other participant is unaffected; a new message re-opens it.
func (r *Repository) Hide(ctx context.Context, convID, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE conversation_participants
		SET hidden = TRUE,
		    cleared_up_to_id = COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = $1), 0)
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotParticipant
	}
	return nil
}
