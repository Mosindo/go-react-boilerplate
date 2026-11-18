package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/people"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageContentNeeded = errors.New("message content required")
	ErrMessageTooLong       = errors.New("message too long (2000 characters max)")
	ErrSelfConversation     = errors.New("you cannot message yourself")
)

const (
	MaxMessageRunes     = 2000
	DefaultMessagesPage = 30
	MaxMessagesPage     = 100
	DefaultChatsPage    = 30
	MaxChatsPage        = 100
)

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, coalesce bool)
}

type URLSigner interface {
	PhotoURL(photoID string) string
}

type Service struct {
	repo      Repository
	signer    URLSigner
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, signer URLSigner, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, notifier: notifier, publisher: publisher, now: time.Now}
}

// NormalizeContent trims a message and rejects empty, oversized or binary input.
func NormalizeContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", ErrMessageContentNeeded
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	var b strings.Builder
	for _, r := range content {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String(), nil
}

func (s *Service) access(ctx context.Context, userID, otherID string) (conversationAccess, error) {
	if userID == otherID {
		return conversationAccess{}, ErrSelfConversation
	}
	access, err := s.repo.Access(ctx, userID, otherID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return conversationAccess{}, ErrConversationNotFound
	}
	return access, err
}

func (s *Service) ListChats(ctx context.Context, userID string, limit, offset int) ([]ChatSummary, error) {
	stored, err := s.repo.ListChats(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ChatSummary, 0, len(stored))
	now := s.now()
	for _, c := range stored {
		summary := ChatSummary{
			User: UserSummary{
				UserID:    c.UserID,
				FirstName: c.FirstName,
				Age:       people.Age(c.Birthdate, now),
			},
			LastMessage: c.LastMessage,
			UnreadCount: c.UnreadCount,
		}
		if c.PhotoID != nil {
			url := s.signer.PhotoURL(*c.PhotoID)
			summary.User.PhotoURL = &url
		}
		out = append(out, summary)
	}
	return out, nil
}

// ListMessages returns a page in chronological order plus the other member's
// read marker so the client can render "seen" states.
func (s *Service) ListMessages(ctx context.Context, userID, otherID string, limit int, beforeID string) (MessagesResponse, error) {
	access, err := s.access(ctx, userID, otherID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if limit <= 0 || limit > MaxMessagesPage {
		limit = DefaultMessagesPage
	}
	messages, err := s.repo.ListMessages(ctx, access.ConversationID, limit, beforeID)
	if err != nil {
		return MessagesResponse{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	otherRead, err := s.repo.LastReadAt(ctx, access.ConversationID, otherID)
	if err != nil {
		return MessagesResponse{}, err
	}
	return MessagesResponse{Messages: messages, HasMore: hasMore, OtherLastReadAt: otherRead}, nil
}

func (s *Service) Send(ctx context.Context, userID, otherID, content string) (Message, error) {
	normalized, err := NormalizeContent(content)
	if err != nil {
		return Message{}, err
	}
	access, err := s.access(ctx, userID, otherID)
	if err != nil {
		return Message{}, err
	}
	message, err := s.repo.CreateMessage(ctx, access.ConversationID, userID, otherID, normalized)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.created", Data: message}
	s.publisher.Publish(otherID, event)
	s.publisher.Publish(userID, event)
	s.notifier.Notify(ctx, otherID, "message.received", "Nouveau message",
		access.SenderFirstName+" vous a écrit.", map[string]string{"userId": userID}, true)
	return message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, otherID string) (time.Time, error) {
	access, err := s.access(ctx, userID, otherID)
	if err != nil {
		return time.Time{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, access.ConversationID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return time.Time{}, ErrConversationNotFound
		}
		return time.Time{}, err
	}
	s.publisher.Publish(otherID, realtime.Event{Type: "conversation.read", Data: map[string]any{"userId": userID, "readAt": readAt}})
	return readAt, nil
}
