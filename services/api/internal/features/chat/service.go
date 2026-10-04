package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service {
	return &Service{repo: repo, rt: rt}
}

func (s *Service) MatchSummary(ctx context.Context, userID, matchID string) (*MatchSummary, error) {
	sum, err := s.repo.GetSummary(ctx, userID, matchID)
	if errors.Is(err, ErrNotFound) {
		return nil, platformerrors.NotFound("Match not found")
	}
	return sum, err
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit int, cursorValue string) ([]MatchSummary, string, error) {
	cursor, err := pagination.Decode(cursorValue)
	if err != nil {
		return nil, "", err
	}
	items, err := s.repo.ListMatches(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, "", err
	}
	items, next := page(items, limit, func(m MatchSummary) string { return pagination.Encode(m.MatchedAt, m.ID) })
	return items, next, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cursorValue string) ([]MatchSummary, string, error) {
	cursor, err := pagination.Decode(cursorValue)
	if err != nil {
		return nil, "", err
	}
	items, err := s.repo.ListConversations(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, "", err
	}
	items, next := page(items, limit, func(m MatchSummary) string { return pagination.Encode(m.UpdatedAt, m.ConversationID) })
	return items, next, nil
}

// page trims the extra look-ahead row and derives the next cursor from the last kept item.
func page(items []MatchSummary, limit int, cursor func(MatchSummary) string) ([]MatchSummary, string) {
	if len(items) <= limit {
		return items, ""
	}
	items = items[:limit]
	return items, cursor(items[len(items)-1])
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, limit int, cursorValue string) ([]Message, string, error) {
	if _, err := uuid.Parse(conversationID); err != nil {
		return nil, "", platformerrors.NotFound("Conversation not found")
	}
	cursor, err := pagination.Decode(cursorValue)
	if err != nil {
		return nil, "", err
	}
	items, err := s.repo.ListMessages(ctx, userID, conversationID, limit+1, cursor)
	if err != nil {
		return nil, "", mapErr(err)
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = pagination.Encode(last.CreatedAt, last.ID)
	}
	return items, next, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	if _, err := uuid.Parse(conversationID); err != nil {
		return Message{}, platformerrors.NotFound("Conversation not found")
	}
	clean := cleanBody(body)
	if clean == "" {
		return Message{}, platformerrors.FieldInvalid("body", "Write a message first")
	}
	if utf8.RuneCountInString(clean) > MaxMessageRunes {
		return Message{}, platformerrors.FieldInvalid("body", "Messages are limited to 2000 characters")
	}
	msg, p, err := s.repo.SendMessage(ctx, userID, conversationID, clean)
	if err != nil {
		return Message{}, mapErr(err)
	}
	for _, uid := range p.UserIDs {
		s.rt.Publish(uid, "message.new", msg)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	if _, err := uuid.Parse(conversationID); err != nil {
		return platformerrors.NotFound("Conversation not found")
	}
	readAt, changed, p, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return mapErr(err)
	}
	if changed {
		for _, uid := range p.UserIDs {
			s.rt.Publish(uid, "message.read", map[string]any{"conversationId": conversationID, "readAt": readAt, "readerId": userID})
		}
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := uuid.Parse(conversationID); err != nil {
		return platformerrors.NotFound("Conversation not found")
	}
	return mapErr(s.repo.Hide(ctx, userID, conversationID))
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if _, err := uuid.Parse(matchID); err != nil {
		return platformerrors.NotFound("Match not found")
	}
	p, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return mapErr(err)
	}
	for _, uid := range p.UserIDs {
		s.rt.Publish(uid, "match.removed", map[string]string{"matchId": p.MatchID, "conversationId": p.ConversationID})
	}
	return nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return platformerrors.NotFound("Conversation not found")
	case errors.Is(err, ErrClosed):
		return platformerrors.Forbidden("This conversation is closed")
	}
	return err
}

// cleanBody trims, normalises newlines and removes control characters (keeping line breaks).
func cleanBody(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
