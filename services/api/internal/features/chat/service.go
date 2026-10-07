package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
)

const maxMessageRunes = 2000

var (
	errNotFound = platformerrors.NotFound("conversation not found")
	errBlocked  = platformerrors.New(403, platformerrors.CodeBlocked, "you can no longer message this person")
)

type SummaryReader interface {
	Summaries(ctx context.Context, userIDs []string) (map[string]profiles.UserSummary, error)
}

type Notifier interface {
	NotifyMessage(ctx context.Context, userID, conversationID, fromUserID, title, preview string) error
}

type Publisher interface {
	Publish(userID, eventType string, data any)
}

type Service struct {
	repo     *Repository
	profiles SummaryReader
	notifier Notifier
	events   Publisher
}

func NewService(repo *Repository, reader SummaryReader, notifier Notifier, events Publisher) *Service {
	return &Service{repo: repo, profiles: reader, notifier: notifier, events: events}
}

func (s *Service) Conversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.PeerID)
	}
	summaries, err := s.profiles.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(rows))
	for _, r := range rows {
		out = append(out, Conversation{
			ID:             r.ID,
			MatchID:        r.MatchID,
			User:           summaries[r.PeerID],
			LastMessage:    r.Last,
			UnreadCount:    r.Unread,
			PeerLastReadAt: r.PeerLastReadAt,
			UpdatedAt:      r.UpdatedAt,
		})
	}
	return out, nil
}

// peer authorises access: the caller must be a participant, otherwise the conversation "does not exist".
func (s *Service) peer(ctx context.Context, conversationID, userID string) (string, error) {
	peer, err := s.repo.Peer(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, ErrRepoNotMember) {
			return "", errNotFound
		}
		return "", err
	}
	return peer, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, before *string, limit int) (MessagesPage, error) {
	if _, err := s.peer(ctx, conversationID, userID); err != nil {
		return MessagesPage{}, err
	}
	// Fetch one extra row to know whether another page exists.
	list, err := s.repo.Messages(ctx, conversationID, before, limit+1)
	if err != nil {
		return MessagesPage{}, err
	}
	page := MessagesPage{Items: list}
	if len(list) > limit {
		page.Items = list[:limit]
		last := page.Items[limit-1].ID
		page.NextCursor = &last
	}
	if page.Items == nil {
		page.Items = []Message{}
	}
	return page, nil
}

func normalizeContent(raw string) (string, error) {
	content := strings.TrimSpace(raw)
	if content == "" {
		return "", platformerrors.Validation("message cannot be empty")
	}
	if utf8.RuneCountInString(content) > maxMessageRunes {
		return "", platformerrors.Validation("message is too long (2000 characters max)")
	}
	for _, r := range content {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return "", platformerrors.Validation("message contains invalid characters")
		}
	}
	return content, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, raw string) (Message, error) {
	content, err := normalizeContent(raw)
	if err != nil {
		return Message{}, err
	}
	peer, err := s.peer(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	blocked, err := s.repo.IsBlocked(ctx, userID, peer)
	if err != nil {
		return Message{}, err
	}
	if blocked {
		return Message{}, errBlocked
	}
	msg, err := s.repo.Insert(ctx, conversationID, userID, content)
	if err != nil {
		return Message{}, err
	}

	s.events.Publish(peer, "message", msg)
	s.events.Publish(userID, "message", msg) // other devices of the sender
	if summaries, err := s.profiles.Summaries(ctx, []string{userID}); err == nil {
		name := summaries[userID].FirstName
		_ = s.notifier.NotifyMessage(ctx, peer, conversationID, userID, fmt.Sprintf("New message from %s", name), preview(content))
	}
	return msg, nil
}

func preview(content string) string {
	runes := []rune(content)
	if len(runes) > 120 {
		return string(runes[:120]) + "…"
	}
	return content
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	peer, err := s.peer(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	event := ReadEvent{ConversationID: conversationID, ReaderID: userID, ReadAt: at}
	s.events.Publish(peer, "read", event)
	s.events.Publish(userID, "read", event)
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if err := s.repo.Hide(ctx, conversationID, userID); err != nil {
		if errors.Is(err, ErrRepoNotMember) {
			return errNotFound
		}
		return err
	}
	return nil
}
