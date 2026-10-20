package chat

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

// Notifier is implemented by the notifications feature.
type Notifier interface {
	NotifyBestEffort(ctx context.Context, userID, kind, actorID string, data map[string]string)
	MarkConversationRead(ctx context.Context, userID, conversationID string)
}

type Service struct {
	repo     Repository
	notifier Notifier
	events   realtime.Publisher
}

func NewService(repo Repository, notifier Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, notifier: notifier, events: events}
}

func (s *Service) requireParticipant(ctx context.Context, conversationID, userID string) (participant, error) {
	if !profiles.IsUUID(conversationID) {
		return participant{}, errNotFound
	}
	return s.repo.Participant(ctx, conversationID, userID)
}

func (s *Service) ListConversations(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) (ConversationsResult, error) {
	items, err := s.repo.ListConversations(ctx, userID, cursor, limit+1)
	if err != nil {
		return ConversationsResult{}, err
	}
	res := ConversationsResult{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next := httpx.EncodeTimeCursor(last.UpdatedAt, last.ID)
		res.NextCursor = &next
	}
	res.Conversations = items
	return res, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, beforeID int64, limit int) (MessagesResult, error) {
	if _, err := s.requireParticipant(ctx, conversationID, userID); err != nil {
		return MessagesResult{}, err
	}
	msgs, otherRead, err := s.repo.ListMessages(ctx, conversationID, userID, beforeID, limit+1)
	if err != nil {
		return MessagesResult{}, err
	}
	res := MessagesResult{OtherLastRead: otherRead}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		next := httpx.EncodeCursor(msgs[len(msgs)-1].ID)
		res.NextCursor = &next
	}
	res.Messages = msgs
	return res, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	p, err := s.requireParticipant(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > MaxMessageLength {
		return Message{}, platformerrors.BadRequest("message must be 1 to 2000 characters")
	}
	if !utf8.ValidString(body) || strings.ContainsRune(body, 0) {
		return Message{}, platformerrors.BadRequest("message contains invalid characters")
	}

	msg, err := s.repo.InsertMessage(ctx, conversationID, userID, body)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{"type": "message.new", "message": msg}
	s.events.Publish(p.OtherUserID, event)
	s.events.Publish(userID, event) // the sender's other devices
	s.notifier.NotifyBestEffort(ctx, p.OtherUserID, "message", userID, map[string]string{"conversationId": conversationID})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	p, err := s.requireParticipant(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	s.notifier.MarkConversationRead(ctx, userID, conversationID)
	s.events.Publish(p.OtherUserID, realtime.Event{"type": "message.read", "conversationId": conversationID, "userId": userID, "readAt": at})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.requireParticipant(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}

// ParseBefore is exposed for the handler.
func ParseBefore(raw string) (int64, bool) { return parseMessageCursor(raw) }
