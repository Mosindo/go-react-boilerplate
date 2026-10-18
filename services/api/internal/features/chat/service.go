package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrInvalidMessage = errors.New("message must contain 1 to 2000 characters")
	ErrInvalidCursor  = errors.New("invalid cursor")
)

type PhotoLister interface {
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.Photo, error)
}

type Service struct {
	repo      Repository
	photos    PhotoLister
	publisher realtime.Publisher
}

func NewService(repo Repository, photos PhotoLister, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, photos: photos, publisher: publisher}
}

func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(raw string) (*cursor, error) {
	if raw == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || len(parts[1]) != 36 {
		return nil, ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, ErrInvalidCursor
	}
	return &cursor{At: at, ID: parts[1]}, nil
}

func (s *Service) Conversations(ctx context.Context, userID, rawCursor string, limit int) (ConversationsResponse, error) {
	cur, err := decodeCursor(rawCursor)
	if err != nil {
		return ConversationsResponse{}, err
	}
	rows, err := s.repo.ListConversations(ctx, userID, cur, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: []ConversationSummary{}}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		resp.NextCursor = encodeCursor(last.sortAt, last.ID)
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.User.UserID
	}
	pics, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	for _, row := range rows {
		if list := pics[row.User.UserID]; len(list) > 0 {
			url := list[0].ThumbURL
			row.User.ThumbURL = &url
		}
		resp.Conversations = append(resp.Conversations, row.ConversationSummary)
	}
	if resp.UnreadTotal, err = s.repo.UnreadTotal(ctx, userID); err != nil {
		return ConversationsResponse{}, err
	}
	return resp, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, rawCursor string, limit int) (MessagesResponse, error) {
	cur, err := decodeCursor(rawCursor)
	if err != nil {
		return MessagesResponse{}, err
	}
	acc, err := s.repo.Access(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	msgs, err := s.repo.ListMessages(ctx, conversationID, acc.HiddenAt, cur, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{OtherLastReadAt: &acc.OtherLastReadAt}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		last := msgs[len(msgs)-1]
		resp.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	resp.Messages = msgs
	return resp, nil
}

func cleanMessage(raw string) (string, error) {
	content := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, raw)
	content = strings.TrimSpace(content)
	if n := utf8.RuneCountInString(content); n < 1 || n > MaxMessageRunes {
		return "", ErrInvalidMessage
	}
	return content, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, raw string) (Message, error) {
	content, err := cleanMessage(raw)
	if err != nil {
		return Message{}, err
	}
	acc, err := s.repo.Access(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, senderName, notified, err := s.repo.InsertMessage(ctx, conversationID, userID, acc.OtherUserID, content)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: msg}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(acc.OtherUserID, event)
	if notified {
		s.publisher.Publish(acc.OtherUserID, realtime.Event{Type: "notification.new", Data: map[string]string{
			"type": "message", "conversationId": conversationID, "firstName": senderName}})
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	acc, err := s.repo.Access(ctx, conversationID, userID)
	if err != nil {
		return time.Time{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return time.Time{}, err
	}
	s.publisher.Publish(acc.OtherUserID, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": conversationID, "readAt": readAt}})
	return readAt, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Access(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}
