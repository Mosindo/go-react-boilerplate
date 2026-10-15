package chat

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/users"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = platformerrors.NotFound("conversation_not_found", "conversation not found")
	ErrEmptyMessage         = platformerrors.BadRequest("empty_message", "the message cannot be empty")
	ErrMessageTooLong       = platformerrors.BadRequest("message_too_long", "the message is too long (2000 characters at most)")
	ErrInvalidCursor        = platformerrors.BadRequest("invalid_cursor", "before must be a message id")
)

const notificationPreviewRunes = 80

type Notifier interface {
	Notify(ctx context.Context, userID string, n notifications.NewNotification) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo      Repository
	photos    users.PhotoSource
	notifier  Notifier
	publisher Publisher
}

func NewService(repo Repository, photos users.PhotoSource, notifier Notifier, publisher Publisher) *Service {
	return &Service{repo: repo, photos: photos, notifier: notifier, publisher: publisher}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.PeerID)
	}
	photos, err := s.photos.PhotosForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(rows))
	for _, row := range rows {
		conv := row.Conversation
		if list := photos[row.PeerID]; len(list) > 0 {
			thumb := list[0].ThumbURL
			conv.Peer.ThumbURL = &thumb
		}
		out = append(out, conv)
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, beforeID string, limit int) (MessagesResponse, error) {
	_, peerRead, err := s.repo.PeerOf(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, mapNotFound(err)
	}
	messages, err := s.repo.ListMessages(ctx, conversationID, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return MessagesResponse{Messages: messages, HasMore: hasMore, PeerLastReadAt: peerRead}, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, rawBody string) (Message, error) {
	body, err := normalizeBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	sent, err := s.repo.InsertMessage(ctx, conversationID, userID, body)
	if err != nil {
		return Message{}, mapNotFound(err)
	}

	event := realtime.Event{Type: "message.new", Data: sent.Message}
	s.publisher.Publish(sent.RecipientID, event)
	s.publisher.Publish(userID, event)

	notifyErr := s.notifier.Notify(ctx, sent.RecipientID, notifications.NewNotification{
		Type:      notifications.TypeMessage,
		Title:     sent.SenderName,
		Body:      preview(body),
		Data:      map[string]any{"conversationId": conversationID, "userId": userID, "firstName": sent.SenderName},
		DedupeKey: "message:" + conversationID,
	})
	if notifyErr != nil {
		log.Printf(`{"event":"message_notification_failed","error":%q}`, notifyErr.Error())
	}
	return sent.Message, nil
}

// MarkRead records that the user has seen the conversation and tells the other person.
func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	peerID, readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return time.Time{}, mapNotFound(err)
	}
	if err := s.notifier.MarkConversationRead(ctx, userID, conversationID); err != nil {
		log.Printf(`{"event":"conversation_notification_read_failed","error":%q}`, err.Error())
	}
	s.publisher.Publish(peerID, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": conversationID,
		"readAt":         readAt,
	}})
	return readAt, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return mapNotFound(s.repo.Hide(ctx, conversationID, userID))
}

func mapNotFound(err error) error {
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrConversationNotFound
	}
	return err
}

// normalizeBody trims the message, unifies line breaks and drops control characters.
func normalizeBody(raw string) (string, error) {
	text := strings.ReplaceAll(strings.TrimSpace(raw), "\r\n", "\n")
	var b strings.Builder
	for _, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	body := strings.TrimSpace(b.String())
	if body == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return body, nil
}

func preview(body string) string {
	flat := strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(flat) <= notificationPreviewRunes {
		return flat
	}
	runes := []rune(flat)
	return string(runes[:notificationPreviewRunes-1]) + "…"
}
