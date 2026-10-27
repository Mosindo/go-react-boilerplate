package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound       = errors.New("conversation not found")
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrMessageTooLong = errors.New("message is too long")
	ErrInvalidCursor  = errors.New("invalid cursor")
)

// Notifier persists a notification for the recipient of a new message.
type Notifier interface {
	NewMessage(ctx context.Context, recipientID, senderID, conversationID string)
}

type Service struct {
	repo     Repository
	pub      realtime.Publisher
	notifier Notifier
}

func NewService(repo Repository, pub realtime.Publisher, notifier Notifier) *Service {
	return &Service{repo: repo, pub: pub, notifier: notifier}
}

func (s *Service) Conversations(ctx context.Context, userID string, limit int, cursor string) (ConversationPage, error) {
	if limit < 1 || limit > httpx.MaxLimit {
		limit = httpx.DefaultLimit
	}
	var before *ConvCursor
	if cursor != "" {
		c, err := decodeConvCursor(cursor)
		if err != nil {
			return ConversationPage{}, ErrInvalidCursor
		}
		before = &c
	}
	items, next, err := s.repo.ListConversations(ctx, userID, limit, before)
	if err != nil {
		return ConversationPage{}, err
	}
	page := ConversationPage{Conversations: items}
	if next != nil {
		page.NextCursor = encodeConvCursor(*next)
	}
	return page, nil
}

func (s *Service) Messages(ctx context.Context, convID, userID string, limit int, cursor string) (MessagePage, error) {
	if limit < 1 || limit > httpx.MaxLimit {
		limit = httpx.MaxLimit
	}
	var beforeID int64
	if cursor != "" {
		id, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil || id < 1 {
			return MessagePage{}, ErrInvalidCursor
		}
		beforeID = id
	}
	page, err := s.repo.ListMessages(ctx, convID, userID, limit, beforeID)
	if errors.Is(err, ErrRepoNotFound) {
		return MessagePage{}, ErrNotFound
	}
	return page, err
}

func (s *Service) Send(ctx context.Context, convID, userID, body string) (Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return Message{}, ErrMessageTooLong
	}
	for _, r := range body {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return Message{}, ErrEmptyMessage
		}
	}
	msg, recipient, err := s.repo.InsertMessage(ctx, convID, userID, body)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	// Delivered to the recipient and to the sender's other devices.
	s.pub.Publish(recipient, "message", msg)
	s.pub.Publish(userID, "message", msg)
	if s.notifier != nil {
		s.notifier.NewMessage(ctx, recipient, userID, convID)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, convID, userID string) (ReadResult, error) {
	upTo, other, err := s.repo.MarkRead(ctx, convID, userID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ReadResult{}, ErrNotFound
		}
		return ReadResult{}, err
	}
	s.pub.Publish(other, "read", map[string]any{"conversationId": convID, "userId": userID, "upToMessageId": upTo})
	return ReadResult{UpToMessageID: upTo}, nil
}

// Clear hides the conversation history for the caller only.
func (s *Service) Clear(ctx context.Context, convID, userID string) error {
	if err := s.repo.Clear(ctx, convID, userID); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func formatID(id int64) string { return strconv.FormatInt(id, 10) }

func encodeConvCursor(c ConvCursor) string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.At.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

func decodeConvCursor(raw string) (ConvCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ConvCursor{}, err
	}
	ts, id, ok := strings.Cut(string(b), "|")
	if !ok || !httpx.IsUUID(id) {
		return ConvCursor{}, ErrInvalidCursor
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ConvCursor{}, err
	}
	return ConvCursor{At: t, ID: id}, nil
}
