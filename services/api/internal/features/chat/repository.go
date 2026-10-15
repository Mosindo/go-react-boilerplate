package chat

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRepositoryNotFound = errors.New("chat: conversation not found")

type ConversationRow struct {
	Conversation
	PeerID string
}

type SentMessage struct {
	Message
	RecipientID string
	SenderName  string
}

type Repository interface {
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationRow, error)
	PeerOf(ctx context.Context, conversationID, userID string) (peerID string, peerLastReadAt *time.Time, err error)
	ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error)
	InsertMessage(ctx context.Context, conversationID, senderID, body string) (SentMessage, error)
	MarkRead(ctx context.Context, conversationID, userID string) (peerID string, readAt time.Time, err error)
	Hide(ctx context.Context, conversationID, userID string) error
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

// ListConversations returns the user's conversations, most recent activity
// first. A conversation hidden by the user stays hidden until a newer message arrives.
func (r *PGRepository) ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationRow, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT c.id,
		       pr.user_id, pr.first_name,
		       m.created_at,
		       COALESCE(last.created_at, m.created_at) AS activity_at,
		       last.id, last.sender_user_id, last.content, last.created_at,
		       (SELECT COUNT(*) FROM messages mm
		        WHERE mm.conversation_id = c.id
		          AND mm.sender_user_id <> $1
		          AND (me.last_read_at IS NULL OR mm.created_at > me.last_read_at)) AS unread
		FROM conversation_participants me
		JOIN conversations c ON c.id = me.conversation_id
		JOIN matches m ON m.id = c.match_id
		JOIN profiles pr ON pr.user_id = CASE WHEN m.user_a = $1 THEN m.user_b ELSE m.user_a END
		LEFT JOIN LATERAL (
			SELECT id, sender_user_id, content, created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		) last ON TRUE
		WHERE me.user_id = $1
		  AND (me.hidden_at IS NULL OR last.created_at > me.hidden_at)
		ORDER BY activity_at DESC, c.id
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]ConversationRow, 0, limit)
	for rows.Next() {
		var row ConversationRow
		var lastID, lastSender, lastBody *string
		var lastAt *time.Time
		if err := rows.Scan(
			&row.ID, &row.PeerID, &row.Peer.FirstName, &row.MatchedAt, &row.UpdatedAt,
			&lastID, &lastSender, &lastBody, &lastAt, &row.UnreadCount,
		); err != nil {
			return nil, err
		}
		row.Peer.UserID = row.PeerID
		if lastID != nil {
			row.LastMessage = &Message{ID: *lastID, ConversationID: row.ID, SenderID: *lastSender, Body: *lastBody, CreatedAt: *lastAt}
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// PeerOf is the authorization check for every conversation endpoint: it only
// answers for participants, so non-members always get "not found".
func (r *PGRepository) PeerOf(ctx context.Context, conversationID, userID string) (string, *time.Time, error) {
	var peerID string
	var peerRead *time.Time
	err := r.dbPool.QueryRow(ctx, `
		SELECT other.user_id, other.last_read_at
		FROM conversation_participants me
		JOIN conversation_participants other
		  ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
	`, conversationID, userID).Scan(&peerID, &peerRead)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrRepositoryNotFound
	}
	return peerID, peerRead, err
}

func (r *PGRepository) ListMessages(ctx context.Context, conversationID, beforeID string, limit int) ([]Message, error) {
	var before *string
	if beforeID != "" {
		before = &beforeID
	}
	rows, err := r.dbPool.Query(ctx, `
		SELECT id, conversation_id, sender_user_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		  AND ($2::uuid IS NULL OR (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $2::uuid AND conversation_id = $1))
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`, conversationID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Message, 0, limit)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *PGRepository) InsertMessage(ctx context.Context, conversationID, senderID, body string) (SentMessage, error) {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return SentMessage{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sent SentMessage
	err = tx.QueryRow(ctx, `
		INSERT INTO messages (conversation_id, sender_user_id, recipient_user_id, content)
		SELECT me.conversation_id, me.user_id, other.user_id, $3
		FROM conversation_participants me
		JOIN conversation_participants other
		  ON other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		WHERE me.conversation_id = $1 AND me.user_id = $2
		RETURNING id, conversation_id, sender_user_id, content, created_at, recipient_user_id,
		          (SELECT first_name FROM profiles WHERE user_id = $2)
	`, conversationID, senderID, body).Scan(
		&sent.ID, &sent.ConversationID, &sent.SenderID, &sent.Body, &sent.CreatedAt, &sent.RecipientID, &sent.SenderName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SentMessage{}, ErrRepositoryNotFound
	}
	if err != nil {
		return SentMessage{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID); err != nil {
		return SentMessage{}, err
	}
	// Writing again brings back a conversation the sender had hidden.
	if _, err := tx.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NULL, last_read_at = $3
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, senderID, sent.CreatedAt); err != nil {
		return SentMessage{}, err
	}
	return sent, tx.Commit(ctx)
}

func (r *PGRepository) MarkRead(ctx context.Context, conversationID, userID string) (string, time.Time, error) {
	var peerID string
	var readAt time.Time
	err := r.dbPool.QueryRow(ctx, `
		UPDATE conversation_participants me
		SET last_read_at = NOW()
		FROM conversation_participants other
		WHERE me.conversation_id = $1 AND me.user_id = $2
		  AND other.conversation_id = me.conversation_id AND other.user_id <> me.user_id
		RETURNING other.user_id, me.last_read_at
	`, conversationID, userID).Scan(&peerID, &readAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, ErrRepositoryNotFound
	}
	return peerID, readAt, err
}

func (r *PGRepository) Hide(ctx context.Context, conversationID, userID string) error {
	tag, err := r.dbPool.Exec(ctx, `
		UPDATE conversation_participants SET hidden_at = NOW(), last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrRepositoryNotFound
	}
	return nil
}
