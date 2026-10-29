package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

var errNotFound = platformerrors.NotFound("conversation not found")

func checkConversationID(id string) error {
	if !profiles.IsUUID(id) {
		return errNotFound
	}
	return nil
}

func mapErr(err error) error {
	if errors.Is(err, ErrNotParticipant) {
		return errNotFound
	}
	return err
}

func (s *Service) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) (ConversationsResponse, error) {
	items, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].SortAt
		resp.NextBefore = &next
	}
	if items == nil {
		items = []Conversation{}
	}
	resp.Conversations = items
	return resp, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, before *int64, limit int) (MessagesResponse, error) {
	if err := checkConversationID(conversationID); err != nil {
		return MessagesResponse{}, err
	}
	items, err := s.repo.ListMessages(ctx, userID, conversationID, before, limit+1)
	if err != nil {
		return MessagesResponse{}, mapErr(err)
	}
	resp := MessagesResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].ID
		resp.NextBefore = &next
	}
	// Newest first from the database; clients render oldest first.
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	if items == nil {
		items = []Message{}
	}
	resp.Messages = items
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	if err := checkConversationID(conversationID); err != nil {
		return Message{}, err
	}
	body, err := cleanBody(body)
	if err != nil {
		return Message{}, err
	}
	res, err := s.repo.Send(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, mapErr(err)
	}

	event := realtime.Event{Type: "message", Data: res.Message}
	s.publisher.Publish(res.RecipientID, event)
	s.publisher.Publish(userID, event) // keeps the sender's other devices in sync
	if res.Notification != nil {
		s.publisher.Publish(res.RecipientID, realtime.Event{Type: "notification", Data: res.Notification})
	}
	return res.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (ReadResponse, error) {
	if err := checkConversationID(conversationID); err != nil {
		return ReadResponse{}, err
	}
	res, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return ReadResponse{}, mapErr(err)
	}
	if res.ReadCount > 0 {
		s.publisher.Publish(res.OtherID, realtime.Event{Type: "read", Data: map[string]any{
			"conversationId": conversationID, "readerId": userID, "readAt": res.ReadAt,
		}})
	}
	return ReadResponse{ReadCount: res.ReadCount}, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if err := checkConversationID(conversationID); err != nil {
		return err
	}
	return mapErr(s.repo.Hide(ctx, userID, conversationID))
}

// cleanBody trims the message and rejects empty text or control characters
// (newlines and tabs are allowed).
func cleanBody(body string) (string, error) {
	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	if body == "" {
		return "", platformerrors.Invalid("message cannot be empty")
	}
	if utf8.RuneCountInString(body) > MaxMessageLength {
		return "", platformerrors.Invalid("message must be at most %d characters", MaxMessageLength)
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", platformerrors.Invalid("message contains invalid characters")
		}
	}
	return body, nil
}
