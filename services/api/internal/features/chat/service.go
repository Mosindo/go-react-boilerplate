package chat

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = apperr.New(http.StatusNotFound, "conversation_not_found", "conversation not found")
	ErrEmptyMessage         = apperr.Validation("empty_message", "message cannot be empty")
	ErrMessageTooLong       = apperr.Validation("message_too_long", "message must be at most 2000 characters")
)

type PhotoLister interface {
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.Photo, error)
}

type Service struct {
	repo   Repository
	photos PhotoLister
	pub    realtime.Publisher
}

func NewService(repo Repository, p PhotoLister, pub realtime.Publisher) *Service {
	return &Service{repo: repo, photos: p, pub: pub}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ConversationsResponse, error) {
	list, err := s.repo.ListConversations(ctx, userID, "", limit+1, offset)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{}
	if len(list) > limit {
		list = list[:limit]
		resp.HasMore = true
	}
	if err := s.attachPhotos(ctx, list); err != nil {
		return ConversationsResponse{}, err
	}
	resp.Conversations = list
	return resp, nil
}

func (s *Service) Get(ctx context.Context, userID, conversationID string) (Conversation, error) {
	list, err := s.repo.ListConversations(ctx, userID, conversationID, 1, 0)
	if err != nil {
		return Conversation{}, err
	}
	if len(list) == 0 {
		return Conversation{}, ErrConversationNotFound
	}
	if err := s.attachPhotos(ctx, list); err != nil {
		return Conversation{}, err
	}
	return list[0], nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, beforeID string, limit int) (MessagesResponse, error) {
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	list, err := s.repo.ListMessages(ctx, conversationID, m.ClearedAt, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: list, PeerLastReadAt: m.PeerLastReadAt}
	if len(list) > limit {
		resp.Messages = list[:limit]
		cursor := list[limit-1].ID
		resp.NextCursor = &cursor
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	clean, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.InsertMessage(ctx, conversationID, userID, clean)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.created", Data: msg}
	s.pub.Publish(m.PeerID, event)
	s.pub.Publish(userID, event)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	m, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return time.Time{}, err
	}
	at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return time.Time{}, err
	}
	s.pub.Publish(m.PeerID, realtime.Event{Type: "conversation.read", Data: map[string]any{
		"conversationId": conversationID, "userId": userID, "readAt": at,
	}})
	return at, nil
}

// Clear hides the conversation history for userID only (local deletion).
func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.member(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, conversationID, userID)
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}

func (s *Service) member(ctx context.Context, conversationID, userID string) (membership, error) {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if errors.Is(err, errRepoNotFound) {
		// Same answer whether the conversation does not exist or belongs to
		// someone else: existence is never leaked.
		return membership{}, ErrConversationNotFound
	}
	return m, err
}

func (s *Service) attachPhotos(ctx context.Context, list []Conversation) error {
	ids := make([]string, 0, len(list))
	for _, c := range list {
		ids = append(ids, c.Peer.UserID)
	}
	byUser, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return err
	}
	for i := range list {
		if p := byUser[list[i].Peer.UserID]; len(p) > 0 {
			primary := p[0]
			list[i].Peer.Photo = &primary
		}
	}
	return nil
}

// NormalizeBody trims a message, strips control characters (except new
// lines) and enforces the length limit.
func NormalizeBody(body string) (string, error) {
	clean := strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, body)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(clean) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return clean, nil
}
