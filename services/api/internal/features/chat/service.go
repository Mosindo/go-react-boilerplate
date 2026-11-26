package chat

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = platformerrors.NotFound("conversation_not_found", "Conversation introuvable.")
	ErrEmptyMessage         = platformerrors.Validation("empty_message", "Le message est vide.")
	ErrMessageTooLong       = platformerrors.Validation("message_too_long", "Le message ne peut pas dépasser 2000 caractères.")
)

type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
	MarkGroupRead(ctx context.Context, userID, groupKey string)
}

type Service struct {
	repo      Repository
	profiles  *profiles.Service
	notifier  Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profileService *profiles.Service, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profileService, notifier: notifier, publisher: publisher}
}

func (s *Service) member(ctx context.Context, conversationID, userID string) (membership, error) {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if errors.Is(err, errRepoNotMember) {
		// 404 rather than 403: never confirm that a conversation exists to an outsider.
		return membership{}, ErrConversationNotFound
	}
	return m, err
}

func (s *Service) ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (ConversationsPage, error) {
	rows, err := s.repo.ListConversations(ctx, userID, cursor, limit+1)
	if err != nil {
		return ConversationsPage{}, err
	}
	page := ConversationsPage{Conversations: []Conversation{}}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := httpx.EncodeCursor(last.LastActivityAt, last.ConversationID)
		page.NextCursor = &next
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherUserID
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return ConversationsPage{}, err
	}
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		page.Conversations = append(page.Conversations, Conversation{
			ID: r.ConversationID, MatchID: r.MatchID, Profile: card, LastMessage: r.LastMessage,
			UnreadCount: r.UnreadCount, OtherLastReadAt: r.OtherLastReadAt, CreatedAt: r.CreatedAt,
		})
	}
	return page, nil
}

func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error) {
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return Conversation{}, err
	}
	cards, err := s.profiles.Cards(ctx, userID, []string{m.OtherUserID})
	if err != nil {
		return Conversation{}, err
	}
	card, ok := cards[m.OtherUserID]
	if !ok {
		return Conversation{}, ErrConversationNotFound
	}
	return Conversation{ID: m.ConversationID, MatchID: m.MatchID, Profile: card, OtherLastReadAt: m.OtherLastReadAt, CreatedAt: m.CreatedAt}, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, cursor *httpx.Cursor, limit int) (MessagesPage, error) {
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return MessagesPage{}, err
	}
	messages, err := s.repo.ListMessages(ctx, conversationID, cursor, limit+1)
	if err != nil {
		return MessagesPage{}, err
	}
	page := MessagesPage{Messages: messages, OtherLastReadAt: m.OtherLastReadAt}
	if len(messages) > limit {
		page.Messages = messages[:limit]
		last := page.Messages[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	if page.Messages == nil {
		page.Messages = []Message{}
	}
	return page, nil
}

// NormalizeMessage trims a message body and enforces length limits.
func NormalizeMessage(body string) (string, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(body, "\x00", ""))
	if trimmed == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(trimmed) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return trimmed, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	text, err := NormalizeMessage(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.InsertMessage(ctx, conversationID, userID, text)
	if err != nil {
		return Message{}, err
	}

	event := realtime.Event{Type: "message.created", Data: msg}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(m.OtherUserID, event)

	senderName := "Votre match"
	if cards, err := s.profiles.Cards(ctx, m.OtherUserID, []string{userID}); err == nil {
		if card, ok := cards[userID]; ok {
			senderName = card.FirstName
		}
	}
	preview := text
	if utf8.RuneCountInString(preview) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	s.notifier.Notify(ctx, notifications.Input{
		UserID:   m.OtherUserID,
		Type:     notifications.TypeMessage,
		Title:    "Nouveau message de " + senderName,
		Body:     preview,
		Data:     map[string]any{"conversationId": conversationID, "userId": userID},
		GroupKey: "conversation:" + conversationID,
	})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (ReadReceipt, error) {
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return ReadReceipt{}, err
	}
	at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return ReadReceipt{}, err
	}
	receipt := ReadReceipt{ConversationID: conversationID, UserID: userID, LastReadAt: at}
	s.publisher.Publish(m.OtherUserID, realtime.Event{Type: "conversation.read", Data: receipt})
	s.publisher.Publish(userID, realtime.Event{Type: "conversation.read", Data: receipt})
	s.notifier.MarkGroupRead(ctx, userID, "conversation:"+conversationID)
	return receipt, nil
}

func (s *Service) UnreadConversations(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadConversations(ctx, userID)
}
