package chat

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	defaultMessagesLimit = 50
	maxMessagesLimit     = 100
)

type ProfileProvider interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	NotifyMessage(ctx context.Context, recipientID, conversationID, senderID, senderName, preview string) error
}

type Service struct {
	repo      Repository
	profiles  ProfileProvider
	notifier  Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profiles ProfileProvider, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, publisher: publisher}
}

// NormalizeContent trims and validates message text.
func NormalizeContent(content string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
	if normalized == "" {
		return "", ErrMessageContentNeeded
	}
	if utf8.RuneCountInString(normalized) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return normalized, nil
}

func (s *Service) ListChats(ctx context.Context, userID string, limit, offset int) ([]ChatSummary, error) {
	stored, err := s.repo.ListChats(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, c := range stored {
		ids = append(ids, c.OtherUserID)
	}
	cards, err := s.profiles.PublicProfiles(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]ChatSummary, 0, len(stored))
	for _, c := range stored {
		card, ok := byID[c.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, ChatSummary{
			MatchID: c.MatchID, ConversationID: c.ConversationID, User: card,
			LastMessage: c.LastMessage, UnreadCount: c.UnreadCount, MatchedAt: c.MatchedAt,
		})
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, organizationID, userID, otherUserID, beforeID string, limit int) (MessagesResponse, error) {
	ref, err := s.repo.ConversationForPair(ctx, organizationID, userID, otherUserID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if limit <= 0 {
		limit = defaultMessagesLimit
	}
	limit = min(limit, maxMessagesLimit)
	messages, hasMore, err := s.repo.ListMessages(ctx, ref.ConversationID, beforeID, limit)
	if err != nil {
		return MessagesResponse{}, err
	}
	readAt, err := s.repo.OtherLastReadAt(ctx, ref.ConversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	return MessagesResponse{Messages: messages, OtherLastReadAt: readAt, HasMore: hasMore}, nil
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, otherUserID, content string) (Message, error) {
	normalized, err := NormalizeContent(content)
	if err != nil {
		return Message{}, err
	}
	ref, err := s.repo.ConversationForPair(ctx, organizationID, userID, otherUserID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, ref.ConversationID, userID, otherUserID, normalized)
	if err != nil {
		return Message{}, err
	}

	event := realtime.Event{Type: "message.new", Data: msg}
	s.publisher.Publish(otherUserID, event)
	s.publisher.Publish(userID, event)

	senderName := "Quelqu'un"
	if cards, err := s.profiles.PublicProfiles(ctx, otherUserID, []string{userID}); err == nil && len(cards) == 1 {
		senderName = cards[0].FirstName
	}
	_ = s.notifier.NotifyMessage(ctx, otherUserID, ref.ConversationID, userID, senderName, normalized)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, organizationID, userID, otherUserID string) (ReadResponse, error) {
	ref, err := s.repo.ConversationForPair(ctx, organizationID, userID, otherUserID)
	if err != nil {
		return ReadResponse{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, ref.ConversationID, userID)
	if err != nil {
		return ReadResponse{}, err
	}
	s.publisher.Publish(otherUserID, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": ref.ConversationID, "userId": userID, "readAt": readAt,
	}})
	return ReadResponse{ReadAt: readAt}, nil
}

// Hide removes the conversation from the caller's list only (until a new message arrives).
func (s *Service) Hide(ctx context.Context, organizationID, userID, otherUserID string) error {
	ref, err := s.repo.ConversationForPair(ctx, organizationID, userID, otherUserID)
	if err != nil {
		return err
	}
	return s.repo.Hide(ctx, ref.ConversationID, userID)
}

func (s *Service) UnreadConversations(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadConversations(ctx, userID)
}
