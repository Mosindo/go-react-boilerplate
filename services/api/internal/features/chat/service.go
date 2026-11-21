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
	ErrMessageContentNeeded = errors.New("message content required")
	ErrMessageTooLong       = errors.New("message must be at most 2000 characters")
)

type Service struct {
	repo          Repository
	profiles      *profiles.Service
	notifications *notifications.Service
	publisher     realtime.Publisher
}

func NewService(repo Repository, profilesService *profiles.Service, notificationsService *notifications.Service, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, profiles: profilesService, notifications: notificationsService, publisher: publisher}
}

// NormalizeContent trims a message and removes control characters except newlines.
func NormalizeContent(content string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ReplaceAll(content, "\r\n", "\n") {
		if r != '\n' && unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	normalized := strings.TrimSpace(b.String())
	if normalized == "" {
		return "", ErrMessageContentNeeded
	}
	if utf8.RuneCountInString(normalized) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return normalized, nil
}

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func (s *Service) ListChats(ctx context.Context, userID string, limit, offset int) ([]ChatSummaryResponse, error) {
	if offset < 0 {
		offset = 0
	}
	rows, err := s.repo.ListChats(ctx, userID, clampLimit(limit, DefaultChatsLimit, MaxChatsLimit), offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.OtherUserID)
	}
	summaries, err := s.profiles.Summaries(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ChatSummaryResponse, 0, len(rows))
	for _, row := range rows {
		summary, ok := summaries[row.OtherUserID]
		if !ok {
			summary = profiles.DeletedSummary(row.OtherUserID)
		}
		item := ChatSummaryResponse{MatchID: row.MatchID, User: summary, MatchedAt: row.MatchedAt, UnreadCount: row.UnreadCount}
		if row.LastMessage != nil {
			item.LastMessage = &ChatMessagePreviewResponse{
				ID:           row.LastMessage.ID,
				SenderUserID: row.LastMessage.SenderUserID,
				Content:      row.LastMessage.Content,
				CreatedAt:    row.LastMessage.CreatedAt,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, otherUserID string, beforeID *string, limit int) (ChatMessagesResponse, error) {
	conv, err := s.repo.GetConversation(ctx, userID, otherUserID)
	if err != nil {
		return ChatMessagesResponse{}, err
	}
	limit = clampLimit(limit, DefaultChatMessagesLimit, MaxChatMessagesLimit)
	messages, err := s.repo.ListMessages(ctx, conv, beforeID, limit+1)
	if err != nil {
		return ChatMessagesResponse{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[1:]
	}
	payload := make([]ChatMessageResponse, 0, len(messages))
	for _, m := range messages {
		payload = append(payload, toResponse(m, conv))
	}
	return ChatMessagesResponse{Messages: payload, HasMore: hasMore, OtherLastReadAt: conv.OtherLastReadAt}, nil
}

func toResponse(m ChatMessage, conv Conversation) ChatMessageResponse {
	isRead := false
	if conv.OtherLastReadAt != nil && !conv.OtherLastReadAt.Before(m.CreatedAt) {
		isRead = true
	}
	return ChatMessageResponse{
		ID:              m.ID,
		SenderUserID:    m.SenderUserID,
		RecipientUserID: m.RecipientUserID,
		Content:         m.Content,
		CreatedAt:       m.CreatedAt,
		IsRead:          isRead,
	}
}

func (s *Service) SendMessage(ctx context.Context, userID, otherUserID, content string) (ChatMessageResponse, error) {
	normalized, err := NormalizeContent(content)
	if err != nil {
		return ChatMessageResponse{}, err
	}
	conv, err := s.repo.GetConversation(ctx, userID, otherUserID)
	if err != nil {
		return ChatMessageResponse{}, err
	}
	message, err := s.repo.CreateMessage(ctx, conv.ID, userID, otherUserID, normalized)
	if err != nil {
		return ChatMessageResponse{}, err
	}
	resp := toResponse(message, conv)
	resp.IsRead = false

	s.profiles.TouchActive(ctx, userID)
	event := realtime.Event{Type: "message.new", Data: map[string]any{"message": resp, "fromUserId": userID, "toUserId": otherUserID}}
	s.publisher.Publish(otherUserID, event)
	s.publisher.Publish(userID, event)

	senderName := "Quelqu'un"
	if summaries, err := s.profiles.Summaries(ctx, otherUserID, []string{userID}); err == nil {
		if summary, ok := summaries[userID]; ok {
			senderName = summary.FirstName
		}
	}
	s.notifications.NotifyCoalesced(ctx, otherUserID, notifications.TypeMessage, "userId",
		"Nouveau message", senderName+" vous a écrit.", map[string]string{"userId": userID})
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, otherUserID string) (ReadReceiptResponse, error) {
	conv, err := s.repo.GetConversation(ctx, userID, otherUserID)
	if err != nil {
		return ReadReceiptResponse{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, conv.ID, userID)
	if err != nil {
		return ReadReceiptResponse{}, err
	}
	s.notifications.MarkReadByRef(ctx, userID, notifications.TypeMessage, "userId", otherUserID)
	s.publisher.Publish(otherUserID, realtime.Event{Type: "message.read", Data: map[string]any{"userId": userID, "readAt": readAt}})
	return ReadReceiptResponse{ReadAt: readAt}, nil
}

// Clear hides the conversation history for the caller only.
func (s *Service) Clear(ctx context.Context, userID, otherUserID string) error {
	conv, err := s.repo.GetConversation(ctx, userID, otherUserID)
	if err != nil {
		return err
	}
	return s.repo.Clear(ctx, conv.ID, userID)
}
