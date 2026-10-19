package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func decodeCursor(cursor string) (*time.Time, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	at, id, err := pagination.Decode(cursor)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return &at, id, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cursor string) (ConversationsResponse, error) {
	limit = pagination.Limit(limit, ConversationsDefaultLimit, ConversationsMaxLimit)
	at, id, err := decodeCursor(cursor)
	if err != nil {
		return ConversationsResponse{}, err
	}
	items, err := s.repo.ListConversations(ctx, userID, limit+1, at, id)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		resp.NextCursor = pagination.Encode(last.UpdatedAt, last.ID)
	}
	resp.Conversations = items
	return resp, nil
}

func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error) {
	return s.repo.GetConversation(ctx, userID, conversationID)
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, limit int, cursor string) (MessagesResponse, error) {
	limit = pagination.Limit(limit, MessagesDefaultLimit, MessagesMaxLimit)
	at, id, err := decodeCursor(cursor)
	if err != nil {
		return MessagesResponse{}, err
	}
	items, peerRead, err := s.repo.ListMessages(ctx, userID, conversationID, limit+1, at, id)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{PeerLastReadAt: peerRead}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		resp.NextCursor = pagination.Encode(last.CreatedAt, last.ID)
	}
	resp.Messages = items
	return resp, nil
}

// NormalizeBody trims a message and rejects empty, oversized or control-char
// content (newlines and tabs are allowed).
func NormalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" || !utf8.ValidString(body) || utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", ErrInvalidMessage
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrInvalidMessage
		}
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	body, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	sent, err := s.repo.SendMessage(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, err
	}
	s.pub.Publish(sent.PeerID, realtime.Event{Type: "message.new", Data: sent.Message})
	s.pub.Publish(userID, realtime.Event{Type: "message.new", Data: sent.Message})
	if sent.NotificationAdded {
		s.pub.Publish(sent.PeerID, realtime.Event{Type: "notification.new"})
	}
	return sent.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	peerID, readAt, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	s.pub.Publish(peerID, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": conversationID,
		"readAt":         readAt,
	}})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return s.repo.Hide(ctx, userID, conversationID)
}
