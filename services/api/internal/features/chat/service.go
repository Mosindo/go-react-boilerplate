package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NopPublisher{}
	}
	return &Service{repo: repo, publisher: publisher}
}

// NormalizeContent trims, strips control characters (except new lines) and
// enforces the length limit.
func NormalizeContent(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(raw, "\r\n", "\n") {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return "", httpx.Invalid("message_required", "Le message est vide.")
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return "", httpx.Invalid("message_too_long", "Le message ne peut pas dépasser 2000 caractères.")
	}
	return content, nil
}

func validClientID(id string) bool {
	if len(id) > MaxClientIDLength {
		return false
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return true
}

func toResponse(matchID, viewerID string, m Message, otherLastReadAt *time.Time) MessageResponse {
	isRead := m.SenderUserID == viewerID && otherLastReadAt != nil && !otherLastReadAt.Before(m.CreatedAt)
	return MessageResponse{
		ID:           m.ID,
		MatchID:      matchID,
		SenderUserID: m.SenderUserID,
		Content:      m.Content,
		ClientID:     m.ClientID,
		CreatedAt:    m.CreatedAt,
		IsRead:       isRead,
	}
}

// ListMessages returns a page of messages, newest first.
func (s *Service) ListMessages(ctx context.Context, userID, matchID string, cursor *httpx.Cursor, limit int) (MessagesResponse, error) {
	conv, err := s.repo.GetConversation(ctx, userID, matchID)
	if err != nil {
		return MessagesResponse{}, err
	}
	var beforeAt *time.Time
	beforeID := ""
	if cursor != nil {
		beforeAt = &cursor.At
		beforeID = cursor.ID
	}
	list, err := s.repo.ListMessages(ctx, conv.ID, beforeAt, beforeID, limit)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: make([]MessageResponse, 0, len(list)), OtherLastReadAt: conv.OtherLastReadAt}
	for _, m := range list {
		resp.Messages = append(resp.Messages, toResponse(matchID, userID, m, conv.OtherLastReadAt))
	}
	if len(list) == limit {
		last := list[len(list)-1]
		resp.NextCursor = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	return resp, nil
}

// Send stores a message and pushes it to both participants in real time.
func (s *Service) Send(ctx context.Context, userID, matchID, content, clientID string) (MessageResponse, error) {
	normalized, err := NormalizeContent(content)
	if err != nil {
		return MessageResponse{}, err
	}
	clientID = strings.TrimSpace(clientID)
	if !validClientID(clientID) {
		return MessageResponse{}, httpx.Invalid("invalid_client_id", "Identifiant client invalide.")
	}

	m, conv, err := s.repo.CreateMessage(ctx, userID, matchID, normalized, clientID)
	if err != nil {
		return MessageResponse{}, err
	}
	resp := toResponse(matchID, userID, m, conv.OtherLastReadAt)
	s.publisher.Publish(conv.OtherUserID, "message.new", toResponse(matchID, conv.OtherUserID, m, nil))
	s.publisher.Publish(userID, "message.new", resp)
	return resp, nil
}

// MarkRead moves the caller's read marker and tells the other participant.
func (s *Service) MarkRead(ctx context.Context, userID, matchID string) (ReadReceipt, error) {
	conv, readAt, err := s.repo.MarkRead(ctx, userID, matchID)
	if err != nil {
		return ReadReceipt{}, err
	}
	receipt := ReadReceipt{MatchID: matchID, UserID: userID, ReadAt: readAt}
	s.publisher.Publish(conv.OtherUserID, "message.read", receipt)
	s.publisher.Publish(userID, "message.read", receipt)
	return receipt, nil
}
