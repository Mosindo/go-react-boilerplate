package chat

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	defaultMessagesLimit = 30
	maxMessagesLimit     = 100
)

type Service struct {
	repo   Repository
	urls   profiles.PhotoURLer
	events realtime.Publisher
	now    func() time.Time
}

func NewService(repo Repository, urls profiles.PhotoURLer, events realtime.Publisher) *Service {
	return &Service{repo: repo, urls: urls, events: events, now: time.Now}
}

func (s *Service) ListChats(ctx context.Context, userID string, limit, offset int) (ChatsResponse, error) {
	stored, err := s.repo.ListChats(ctx, userID, limit, offset)
	if err != nil {
		return ChatsResponse{}, err
	}
	total, err := s.repo.TotalUnread(ctx, userID)
	if err != nil {
		return ChatsResponse{}, err
	}
	now := s.now()
	out := make([]ChatSummary, 0, len(stored))
	for _, row := range stored {
		summary := row.ChatSummary
		if row.PhotoID != nil {
			summary.User.Photo = &profiles.Photo{ID: *row.PhotoID, URL: s.urls.PhotoURL(*row.PhotoID, now), Width: deref(row.PhotoW), Height: deref(row.PhotoH)}
		}
		out = append(out, summary)
	}
	return ChatsResponse{Chats: out, TotalUnread: total}, nil
}

func deref(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (s *Service) Messages(ctx context.Context, userID, otherID, beforeID string, limit int) (MessagesResponse, error) {
	t, err := s.repo.Thread(ctx, userID, otherID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if limit <= 0 || limit > maxMessagesLimit {
		limit = defaultMessagesLimit
	}
	list, hasMore, err := s.repo.ListMessages(ctx, t, beforeID, limit)
	if err != nil {
		return MessagesResponse{}, err
	}
	return MessagesResponse{Messages: list, HasMore: hasMore, OtherLastReadAt: t.OtherLastReadAt}, nil
}

// NormalizeContent trims a message and enforces length rules.
func NormalizeContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", ErrMessageContentNeeded
	}
	if utf8.RuneCountInString(content) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return content, nil
}

func (s *Service) Send(ctx context.Context, userID, otherID, raw string) (Message, error) {
	content, err := NormalizeContent(raw)
	if err != nil {
		return Message{}, err
	}
	t, err := s.repo.Thread(ctx, userID, otherID)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.CreateMessage(ctx, t.ConversationID, userID, otherID, content)
	if err != nil {
		return Message{}, err
	}
	s.events.Publish(otherID, "message.created", m)
	s.events.Publish(userID, "message.created", m)
	return m, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, otherID string) (ReadEvent, error) {
	t, err := s.repo.Thread(ctx, userID, otherID)
	if err != nil {
		return ReadEvent{}, err
	}
	at, err := s.repo.MarkRead(ctx, t.ConversationID, userID)
	if err != nil {
		return ReadEvent{}, err
	}
	evt := ReadEvent{ConversationID: t.ConversationID, UserID: userID, ReadAt: at}
	s.events.Publish(otherID, "conversation.read", evt)
	s.events.Publish(userID, "conversation.read", evt)
	return evt, nil
}

// Clear hides the current history for the caller only.
func (s *Service) Clear(ctx context.Context, userID, otherID string) error {
	t, err := s.repo.Thread(ctx, userID, otherID)
	if err != nil {
		return err
	}
	return s.repo.Clear(ctx, t.ConversationID, userID)
}
