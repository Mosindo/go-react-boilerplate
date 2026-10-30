package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type PhotoURLs interface {
	PhotoPath(photoID string) string
}

type Events interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo   Repository
	urls   PhotoURLs
	events Events
}

func NewService(repo Repository, urls PhotoURLs, events Events) *Service {
	return &Service{repo: repo, urls: urls, events: events}
}

func (s *Service) ListConversations(ctx context.Context, userID string, before time.Time, limit int) (ConversationsResponse, error) {
	stored, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: []Conversation{}}
	if len(stored) > limit {
		stored = stored[:limit]
		next := stored[len(stored)-1].LastActivityAt
		resp.NextBefore = &next
	}
	for _, c := range stored {
		p := Participant{UserID: c.OtherUserID, DisplayName: c.OtherName}
		if c.OtherPhotoID != nil {
			p.PhotoURL = s.urls.PhotoPath(*c.OtherPhotoID)
		}
		resp.Conversations = append(resp.Conversations, Conversation{
			ID: c.ID, MatchID: c.MatchID, User: p, LastMessage: c.LastMessage,
			UnreadCount: c.UnreadCount, LastActivityAt: c.LastActivityAt,
		})
	}
	return resp, nil
}

// ListMessages returns one page in chronological order; NextBefore points at
// the cursor for the next (older) page when there is one.
func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, before time.Time, limit int) (MessagesResponse, error) {
	list, err := s.repo.ListMessages(ctx, userID, conversationID, before, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{}
	if len(list) > limit {
		list = list[:limit]
		next := list[len(list)-1].CreatedAt
		resp.NextBefore = &next
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	if list == nil {
		list = []Message{}
	}
	resp.Messages = list
	return resp, nil
}

func normalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", platformerrors.Invalid("body", "message cannot be empty")
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", platformerrors.Invalid("body", "message is too long (max 2000 characters)")
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", platformerrors.Invalid("body", "message contains invalid characters")
		}
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	clean, err := normalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, recipient, err := s.repo.SendMessage(ctx, userID, conversationID, clean)
	if err != nil {
		return Message{}, err
	}
	s.events.Publish(recipient, realtime.Event{Type: realtime.TypeMessage, ConversationID: conversationID})
	s.events.Publish(recipient, realtime.Event{Type: realtime.TypeNotification})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	other, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	s.events.Publish(other, realtime.Event{Type: realtime.TypeRead, ConversationID: conversationID})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return s.repo.Hide(ctx, userID, conversationID)
}
