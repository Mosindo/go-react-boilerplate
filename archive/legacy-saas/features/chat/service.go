package chat

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrValidateChatTarget   = errors.New("could not validate chat target")
	ErrMessageContentNeeded = errors.New("message content required")
)

type Service struct {
	repo Repository
}

const (
	defaultChatsLimit        = 20
	maxChatsLimit            = 100
	defaultChatMessagesLimit = 200
	maxChatMessagesLimit     = 500
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListChats(ctx context.Context, organizationID, userID string) ([]ChatSummary, error) {
	chats, err := s.repo.ListChats(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}

	payload := make([]ChatSummary, 0, len(chats))
	for _, chat := range chats {
		item := ChatSummary{
			UserID:        chat.UserID,
			UserEmail:     chat.UserEmail,
			UserCreatedAt: chat.UserCreatedAt,
		}
		if chat.LastMessage != nil {
			item.LastMessage = &ChatMessagePreview{
				Content:   chat.LastMessage.Content,
				CreatedAt: chat.LastMessage.CreatedAt,
			}
		}
		payload = append(payload, item)
	}

	return payload, nil
}

func (s *Service) ListChatsWithPagination(ctx context.Context, organizationID, userID string, limit, offset int) ([]ChatSummary, error) {
	normalizedLimit := normalizeChatsLimit(limit)
	normalizedOffset := normalizeChatsOffset(offset)
	chats, err := s.repo.ListChatsWithPagination(ctx, organizationID, userID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]ChatSummary, 0, len(chats))
	for _, chat := range chats {
		item := ChatSummary{
			UserID:        chat.UserID,
			UserEmail:     chat.UserEmail,
			UserCreatedAt: chat.UserCreatedAt,
		}
		if chat.LastMessage != nil {
			item.LastMessage = &ChatMessagePreview{
				Content:   chat.LastMessage.Content,
				CreatedAt: chat.LastMessage.CreatedAt,
			}
		}
		payload = append(payload, item)
	}

	return payload, nil
}

func (s *Service) ListMessages(ctx context.Context, organizationID, userID, otherUserID string) ([]ChatMessage, error) {
	return s.ListMessagesWithLimit(ctx, organizationID, userID, otherUserID, defaultChatMessagesLimit)
}

func (s *Service) ListMessagesWithLimit(ctx context.Context, organizationID, userID, otherUserID string, limit int) ([]ChatMessage, error) {
	if err := s.ensureCanChat(ctx, organizationID, userID, otherUserID); err != nil {
		return nil, err
	}

	normalizedLimit := normalizeChatMessagesLimit(limit)
	messages, err := s.repo.ListMessages(ctx, userID, otherUserID, normalizedLimit)
	if err != nil {
		return nil, err
	}
	payload := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		payload = append(payload, ChatMessage{
			ID:              message.ID,
			SenderUserID:    message.SenderUserID,
			RecipientUserID: message.RecipientUserID,
			Content:         message.Content,
			CreatedAt:       message.CreatedAt,
		})
	}
	return payload, nil
}

func (s *Service) SendMessage(ctx context.Context, organizationID, userID, otherUserID, content string) (ChatMessage, error) {
	if err := s.ensureCanChat(ctx, organizationID, userID, otherUserID); err != nil {
		return ChatMessage{}, err
	}

	normalizedContent := strings.TrimSpace(content)
	if normalizedContent == "" {
		return ChatMessage{}, ErrMessageContentNeeded
	}

	message, err := s.repo.CreateMessage(ctx, userID, otherUserID, normalizedContent)
	if err != nil {
		return ChatMessage{}, err
	}

	return ChatMessage{
		ID:              message.ID,
		SenderUserID:    message.SenderUserID,
		RecipientUserID: message.RecipientUserID,
		Content:         message.Content,
		CreatedAt:       message.CreatedAt,
	}, nil
}

func (s *Service) ensureCanChat(ctx context.Context, organizationID, userID, otherUserID string) error {
	exists, err := s.repo.UserExists(ctx, organizationID, otherUserID)
	if err != nil {
		return ErrValidateChatTarget
	}
	if !exists {
		return ErrUserNotFound
	}

	return nil
}

func normalizeChatMessagesLimit(limit int) int {
	if limit <= 0 {
		return defaultChatMessagesLimit
	}
	if limit > maxChatMessagesLimit {
		return maxChatMessagesLimit
	}
	return limit
}

func normalizeChatsLimit(limit int) int {
	if limit <= 0 {
		return defaultChatsLimit
	}
	if limit > maxChatsLimit {
		return maxChatsLimit
	}
	return limit
}

func normalizeChatsOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
