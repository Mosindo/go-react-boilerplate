package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"example.com/api/internal/platform/realtime"
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.PublicProfile, error)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, publisher: publisher}
}

// notFound deliberately hides whether a conversation exists for non-participants.
var notFound = platformerrors.NotFound("not_found", "conversation not found")

func mapErr(err error) error {
	if errors.Is(err, ErrNotParticipant) {
		return notFound
	}
	return err
}

func cleanBody(body string) (string, error) {
	body = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, body))
	if body == "" || len([]rune(body)) > MaxMessageLength {
		return "", platformerrors.BadRequest("invalid_message", "message must be 1 to 2000 characters")
	}
	return body, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cursor string) (ConversationsResponse, error) {
	cur, err := pagination.Decode(cursor)
	if err != nil {
		return ConversationsResponse{}, err
	}
	var afterAt *time.Time
	afterID := ""
	if cur != nil {
		afterAt, afterID = &cur.At, cur.ID
	}
	rows, err := s.repo.ListConversations(ctx, userID, limit+1, afterAt, afterID)
	if err != nil {
		return ConversationsResponse{}, err
	}
	res := ConversationsResponse{Conversations: []ConversationItem{}}
	if len(rows) > limit {
		res.NextCursor = pagination.Cursor{At: rows[limit-1].LastMessageAt, ID: rows[limit-1].ID}.Encode()
		rows = rows[:limit]
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	for _, r := range rows {
		card, ok := cards[r.OtherID]
		if !ok {
			continue
		}
		res.Conversations = append(res.Conversations, ConversationItem{
			ID: r.ID, MatchID: r.MatchID, User: card, LastMessage: r.Last, LastMessageAt: r.LastMessageAt, UnreadCount: r.Unread,
		})
	}
	return res, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, convID string, limit int, cursor string) (MessagesResponse, error) {
	cur, err := pagination.Decode(cursor)
	if err != nil {
		return MessagesResponse{}, err
	}
	var beforeAt *time.Time
	beforeID := ""
	if cur != nil {
		beforeAt, beforeID = &cur.At, cur.ID
	}
	msgs, readAt, err := s.repo.ListMessages(ctx, userID, convID, limit+1, beforeAt, beforeID)
	if err != nil {
		return MessagesResponse{}, mapErr(err)
	}
	res := MessagesResponse{OtherLastReadAt: readAt}
	if len(msgs) > limit {
		last := msgs[limit-1]
		res.NextCursor = pagination.Cursor{At: last.CreatedAt, ID: last.ID}.Encode()
		msgs = msgs[:limit]
	}
	res.Messages = msgs
	return res, nil
}

func (s *Service) Send(ctx context.Context, userID, convID, body string) (Message, error) {
	body, err := cleanBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, other, err := s.repo.Send(ctx, userID, convID, body)
	if err != nil {
		return Message{}, mapErr(err)
	}
	ev := realtime.Event{Type: "message.new", Data: msg}
	s.publisher.Publish(other, ev)
	s.publisher.Publish(userID, ev) // keep the sender's other devices in sync
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, convID string) error {
	other, readAt, err := s.repo.MarkRead(ctx, userID, convID)
	if err != nil {
		return mapErr(err)
	}
	s.publisher.Publish(other, realtime.Event{Type: "conversation.read", Data: map[string]any{"conversationId": convID, "readAt": readAt}})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, convID string) error {
	return mapErr(s.repo.Hide(ctx, userID, convID))
}

func (s *Service) Unread(ctx context.Context, userID string) (UnreadResponse, error) {
	return s.repo.Unread(ctx, userID)
}
