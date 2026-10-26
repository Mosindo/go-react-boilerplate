package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound      = errors.New("conversation not found")
	ErrInvalidBody   = errors.New("message must contain 1 to 2000 characters")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Service struct {
	repo   Repository
	loader *profiles.Loader
	db     profiles.Querier
	pub    realtime.Publisher
}

func NewService(repo Repository, loader *profiles.Loader, db profiles.Querier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, loader: loader, db: db, pub: pub}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cursor string) (ConversationsResponse, error) {
	if limit <= 0 {
		limit = DefaultConversation
	}
	if limit > MaxConversation {
		limit = MaxConversation
	}
	var beforeAt *time.Time
	var beforeID string
	if cursor != "" {
		at, id, err := decodeCursor(cursor)
		if err != nil {
			return ConversationsResponse{}, ErrInvalidCursor
		}
		beforeAt, beforeID = &at, id
	}
	rows, err := s.repo.ListConversations(ctx, userID, limit+1, beforeAt, beforeID)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: make([]Conversation, 0, len(rows))}
	if len(rows) > limit {
		last := rows[limit-1]
		resp.NextCursor = encodeCursor(last.ActivityAt, last.MatchID)
		rows = rows[:limit]
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherID)
	}
	views, err := s.loader.Load(ctx, s.db, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	for _, r := range rows {
		c := r.Conversation
		c.Profile = views[r.OtherID]
		resp.Conversations = append(resp.Conversations, c)
	}
	return resp, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, matchID string, limit int, beforeID int64) (MessagesResponse, error) {
	if limit <= 0 {
		limit = DefaultMessagePage
	}
	if limit > MaxMessagePage {
		limit = MaxMessagePage
	}
	msgs, err := s.repo.ListMessages(ctx, userID, matchID, limit+1, beforeID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return MessagesResponse{}, ErrNotFound
		}
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{}
	if len(msgs) > limit {
		resp.HasMore = true
		msgs = msgs[:limit]
	}
	resp.Messages = msgs
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	clean, err := cleanBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, recipient, err := s.repo.AddMessage(ctx, userID, matchID, clean)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	s.pub.Publish(recipient, realtime.Event{Type: "message", Data: msg})
	s.pub.Publish(userID, realtime.Event{Type: "message", Data: msg}) // sender's other devices
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) (ReadReceipt, error) {
	receipt, other, err := s.repo.MarkRead(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ReadReceipt{}, ErrNotFound
		}
		return ReadReceipt{}, err
	}
	if receipt.Count > 0 {
		s.pub.Publish(other, realtime.Event{Type: "read", Data: receipt})
	}
	return receipt, nil
}

func (s *Service) Clear(ctx context.Context, userID, matchID string) error {
	if err := s.repo.Clear(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func cleanBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if !utf8.ValidString(body) {
		return "", ErrInvalidBody
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' {
			return "", ErrInvalidBody
		}
	}
	n := utf8.RuneCountInString(body)
	if n < 1 || n > MaxBodyLength {
		return "", ErrInvalidBody
	}
	return body, nil
}

func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	return at, parts[1], err
}
