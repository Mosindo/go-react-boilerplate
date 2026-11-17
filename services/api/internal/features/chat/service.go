package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrMessageEmpty   = errors.New("message content required")
	ErrMessageTooLong = errors.New("message too long")
)

type ProfileReader interface {
	PublicProfiles(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	NotifyCollapsed(ctx context.Context, userID, kind, title, body string, data map[string]string, collapseKey string)
}

type Service struct {
	repo      Repository
	profiles  ProfileReader
	notifier  Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profileReader ProfileReader, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileReader, notifier: notifier, publisher: publisher}
}

// NormalizeMessage trims the message and removes control characters except newlines and tabs.
func NormalizeMessage(raw string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, raw)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", ErrMessageEmpty
	}
	if utf8.RuneCountInString(cleaned) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) ListConversations(ctx context.Context, userID, beforeID string, limit int) (ConversationsResponse, error) {
	if limit <= 0 || limit > MaxConversationsLimit {
		limit = DefaultConversationsLimit
	}
	stored, err := s.repo.ListConversations(ctx, userID, beforeID, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	var next *string
	if len(stored) > limit {
		stored = stored[:limit]
		cursor := stored[len(stored)-1].ID
		next = &cursor
	}
	ids := make([]string, 0, len(stored))
	for _, c := range stored {
		ids = append(ids, c.OtherUserID)
	}
	cards, err := s.profiles.PublicProfiles(ctx, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	byUser := make(map[string]profiles.PublicProfile, len(cards))
	for _, card := range cards {
		byUser[card.UserID] = card
	}
	out := make([]Conversation, 0, len(stored))
	for _, c := range stored {
		card, ok := byUser[c.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, Conversation{
			ID: c.ID, MatchID: c.MatchID, MatchedAt: c.MatchedAt, Profile: card,
			LastMessage: c.LastMessage, UnreadCount: c.UnreadCount, OtherLastReadAt: c.OtherLastReadAt,
		})
	}
	return ConversationsResponse{Conversations: out, NextCursor: next}, nil
}

func (s *Service) UnreadConversations(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadConversations(ctx, userID)
}

func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error) {
	access, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return Conversation{}, err
	}
	cards, err := s.profiles.PublicProfiles(ctx, userID, []string{access.OtherUserID})
	if err != nil {
		return Conversation{}, err
	}
	if len(cards) == 0 {
		return Conversation{}, ErrConversationNotFound
	}
	return Conversation{
		ID: access.ConversationID, MatchID: access.MatchID, MatchedAt: access.MatchedAt,
		Profile: cards[0], OtherLastReadAt: access.OtherLastReadAt,
	}, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) (MessagesResponse, error) {
	if limit <= 0 || limit > MaxMessagesLimit {
		limit = DefaultMessagesLimit
	}
	access, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return MessagesResponse{}, err
	}
	messages, err := s.repo.ListMessages(ctx, conversationID, access.HiddenAt, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	var next *string
	if len(messages) > limit {
		messages = messages[:limit]
		cursor := messages[len(messages)-1].ID
		next = &cursor
	}
	return MessagesResponse{Messages: messages, NextCursor: next, OtherLastReadAt: access.OtherLastReadAt}, nil
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, content, clientMessageID string) (Message, error) {
	normalized, err := NormalizeMessage(content)
	if err != nil {
		return Message{}, err
	}
	access, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, conversationID, userID, normalized)
	if err != nil {
		return Message{}, err
	}
	msg.ClientMessageID = strings.TrimSpace(clientMessageID)

	if s.publisher != nil {
		s.publisher.Publish(access.OtherUserID, "message.created", msg)
		s.publisher.Publish(userID, "message.created", msg) // other devices of the sender
	}
	if s.notifier != nil {
		senderName := "Ton match"
		if cards, err := s.profiles.PublicProfiles(ctx, access.OtherUserID, []string{userID}); err == nil && len(cards) == 1 {
			senderName = cards[0].FirstName
		}
		s.notifier.NotifyCollapsed(ctx, access.OtherUserID, notifications.TypeMessage,
			"Nouveau message", senderName+" t'a écrit.",
			map[string]string{"conversationId": conversationID}, conversationID)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (ReadEvent, error) {
	access, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return ReadEvent{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return ReadEvent{}, err
	}
	event := ReadEvent{ConversationID: conversationID, UserID: userID, ReadAt: readAt}
	if s.publisher != nil {
		s.publisher.Publish(access.OtherUserID, "conversation.read", event)
		s.publisher.Publish(userID, "conversation.read", event)
	}
	return event, nil
}

// Hide deletes the conversation for the caller only; it reappears if a new message arrives.
func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Access(ctx, userID, conversationID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}
