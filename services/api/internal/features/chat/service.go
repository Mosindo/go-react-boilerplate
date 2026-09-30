package chat

import (
	"context"
	"errors"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = apperr.NotFound("conversation not found")
	ErrEmptyMessage         = apperr.Validation("message cannot be empty")
	ErrMessageTooLong       = apperr.Validation("message is too long (2000 characters max)")
	ErrInvalidCursor        = apperr.Validation("invalid cursor")
)

type SummaryProvider interface {
	Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Summary, error)
}

type Service struct {
	repo      Repository
	profiles  SummaryProvider
	notifier  notifications.Notifier
	publisher realtime.Publisher
}

func NewService(repo Repository, profiles SummaryProvider, notifier notifications.Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, publisher: publisher}
}

func (s *Service) ListConversations(ctx context.Context, userID, rawCursor string, limit int) (ConversationsResponse, error) {
	before, ok := decodeCursor(rawCursor)
	if !ok {
		return ConversationsResponse{}, ErrInvalidCursor
	}
	rows, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next := encodeCursor(last.ActivityAt, last.ID)
		resp.NextCursor = &next
	}
	resp.Conversations, err = s.decorate(ctx, userID, rows)
	return resp, err
}

func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (Conversation, error) {
	row, err := s.repo.GetConversation(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotParticipant) {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, err
	}
	items, err := s.decorate(ctx, userID, []conversationRow{row})
	if err != nil {
		return Conversation{}, err
	}
	return items[0], nil
}

func (s *Service) decorate(ctx context.Context, userID string, rows []conversationRow) ([]Conversation, error) {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.OtherUserID
	}
	summaries, err := s.profiles.Summaries(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	items := make([]Conversation, 0, len(rows))
	for _, row := range rows {
		summary, ok := summaries[row.OtherUserID]
		if !ok {
			summary = profiles.Summary{UserID: row.OtherUserID}
		}
		items = append(items, Conversation{
			ID:          row.ID,
			MatchID:     row.MatchID,
			MatchedAt:   row.MatchedAt,
			User:        summary,
			LastMessage: row.LastMessage,
			UnreadCount: row.UnreadCount,
		})
	}
	return items, nil
}

// ListMessages returns a page of messages, newest first. Only participants
// can read a conversation; anyone else gets a 404.
func (s *Service) ListMessages(ctx context.Context, userID, conversationID, rawCursor string, limit int) (MessagesResponse, error) {
	before, ok := decodeCursor(rawCursor)
	if !ok {
		return MessagesResponse{}, ErrInvalidCursor
	}
	_, hiddenAt, err := s.repo.Participants(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotParticipant) {
			return MessagesResponse{}, ErrConversationNotFound
		}
		return MessagesResponse{}, err
	}
	messages, err := s.repo.ListMessages(ctx, conversationID, hiddenAt, before, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: messages}
	if len(messages) > limit {
		resp.Messages = messages[:limit]
		last := resp.Messages[limit-1]
		next := encodeCursor(last.CreatedAt, last.ID)
		resp.NextCursor = &next
	}
	resp.OtherLastReadAt, err = s.repo.OtherLastReadAt(ctx, userID, conversationID)
	return resp, err
}

func (s *Service) SendMessage(ctx context.Context, userID, conversationID, rawBody string) (Message, error) {
	body, err := profiles.CleanText(rawBody, MaxMessageRunes, "body", true)
	if err != nil {
		return Message{}, ErrMessageTooLong
	}
	if body == "" {
		return Message{}, ErrEmptyMessage
	}
	otherID, _, err := s.repo.Participants(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotParticipant) {
			return Message{}, ErrConversationNotFound
		}
		return Message{}, err
	}

	message, err := s.repo.CreateMessage(ctx, conversationID, userID, body)
	if err != nil {
		return Message{}, err
	}

	s.publisher.Publish(ctx, []string{userID, otherID}, realtime.Event{Type: "message.created", ConversationID: conversationID, Data: message})

	sender, _ := s.profiles.Summaries(ctx, otherID, []string{userID})
	name := sender[userID].FirstName
	if name == "" {
		name = "Votre match"
	}
	s.notifier.Notify(ctx, otherID, notifications.TypeMessage, "Nouveau message", name+" vous a écrit", map[string]string{
		"conversationId": conversationID,
		"userId":         userID,
		"dedupeKey":      "conversation:" + conversationID,
	})
	return message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	otherID, _, err := s.repo.Participants(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotParticipant) {
			return ErrConversationNotFound
		}
		return err
	}
	at, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	s.notifier.MarkConversationRead(ctx, userID, conversationID)
	if at != nil {
		event := ReadEvent{ConversationID: conversationID, ReaderID: userID, LastReadAt: at.UTC()}
		s.publisher.Publish(ctx, []string{otherID, userID}, realtime.Event{Type: "conversation.read", ConversationID: conversationID, Data: event})
	}
	return nil
}

// Hide deletes the conversation locally: it disappears from the caller's list
// until a new message arrives, and older messages stay hidden for them only.
func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, _, err := s.repo.Participants(ctx, userID, conversationID); err != nil {
		if errors.Is(err, ErrRepositoryNotParticipant) {
			return ErrConversationNotFound
		}
		return err
	}
	return s.repo.Hide(ctx, userID, conversationID)
}
