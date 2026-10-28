package chat

import (
	"context"
	"errors"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

const maxMessageRunes = 2000

var (
	errNotFound    = platformerrors.NotFound("not_found", "conversation not found")
	errBlockedByMe = platformerrors.Forbidden("blocked", "you blocked this person; unblock them to write")
)

// Notifier creates and pushes notifications (notifications.Service).
type Notifier interface {
	Notify(ctx context.Context, userID, actorID, matchID, kind string) (*notifications.Notification, error)
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type Service struct {
	repo     Repository
	pub      realtime.Publisher
	notifier Notifier
}

func NewService(repo Repository, pub realtime.Publisher, notifier Notifier) *Service {
	return &Service{repo: repo, pub: pub, notifier: notifier}
}

// authorize returns the peer id, hiding conversations the caller does not belong to or that the peer's block hides.
func (s *Service) authorize(ctx context.Context, userID, matchID string) (access, error) {
	a, err := s.repo.Access(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return access{}, errNotFound
		}
		return access{}, err
	}
	if a.BlockedByPeer {
		return access{}, errNotFound
	}
	return a, nil
}

func (s *Service) List(ctx context.Context, userID string, limit int, rawCursor string) (ConversationPage, error) {
	cursor, err := httpx.DecodeCursor(rawCursor)
	if err != nil {
		return ConversationPage{}, err
	}
	items, err := s.repo.Conversations(ctx, userID, nil, limit+1, cursor)
	if err != nil {
		return ConversationPage{}, err
	}
	page := ConversationPage{Conversations: items}
	if len(items) > limit {
		page.Conversations = items[:limit]
		last := page.Conversations[limit-1]
		next := httpx.EncodeCursor(last.ActivityAt, last.ID)
		page.NextCursor = &next
	}
	if page.Conversations == nil {
		page.Conversations = []Conversation{}
	}
	return page, nil
}

// Conversation loads a single conversation as seen by userID.
func (s *Service) Conversation(ctx context.Context, userID, matchID string) (Conversation, error) {
	items, err := s.repo.Conversations(ctx, userID, &matchID, 1, nil)
	if err != nil {
		return Conversation{}, err
	}
	if len(items) == 0 {
		return Conversation{}, errNotFound
	}
	return items[0], nil
}

func (s *Service) Messages(ctx context.Context, userID, matchID string, limit int, rawCursor string) (MessagePage, error) {
	if _, err := s.authorize(ctx, userID, matchID); err != nil {
		return MessagePage{}, err
	}
	cursor, err := httpx.DecodeCursor(rawCursor)
	if err != nil {
		return MessagePage{}, err
	}
	items, err := s.repo.Messages(ctx, userID, matchID, limit+1, cursor)
	if err != nil {
		return MessagePage{}, err
	}
	page := MessagePage{Messages: items}
	if len(items) > limit {
		page.Messages = items[:limit]
		last := page.Messages[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	if page.Messages == nil {
		page.Messages = []Message{}
	}
	return page, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, rawBody string) (Message, error) {
	body, err := cleanBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	a, err := s.authorize(ctx, userID, matchID)
	if err != nil {
		return Message{}, err
	}
	if a.BlockedByMe {
		return Message{}, errBlockedByMe
	}
	msg, err := s.repo.InsertMessage(ctx, matchID, userID, body)
	if err != nil {
		return Message{}, err
	}
	s.pub.Publish(a.PeerID, realtime.Event{Type: "message", Data: msg})
	s.pub.Publish(userID, realtime.Event{Type: "message", Data: msg}) // sender's other devices
	if _, err := s.notifier.Notify(ctx, a.PeerID, userID, matchID, notifications.TypeMessage); err != nil {
		log.Printf(`{"event":"notify_error","operation":"chat.send","error":%q}`, err.Error())
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	a, err := s.authorize(ctx, userID, matchID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if err := s.notifier.MarkConversationRead(ctx, userID, matchID); err != nil {
		log.Printf(`{"event":"notify_error","operation":"chat.read","error":%q}`, err.Error())
	}
	if readAt != nil {
		s.pub.Publish(a.PeerID, realtime.Event{Type: "read", Data: ReadReceipt{ConversationID: matchID, ReaderID: userID, ReadAt: *readAt}})
	}
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, matchID string) error {
	if _, err := s.repo.Access(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return errNotFound
		}
		return err
	}
	return s.repo.Clear(ctx, userID, matchID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if _, err := s.repo.Unmatch(ctx, userID, matchID); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return errNotFound
		}
		return err
	}
	return nil
}

// HandleMatch is called by discovery right after a mutual like: it notifies both users and pushes
// the new conversation live. It returns the conversation as seen by swiperID.
func (s *Service) HandleMatch(ctx context.Context, matchID, swiperID, otherID string) (Conversation, error) {
	var mine Conversation
	for _, pair := range [][2]string{{swiperID, otherID}, {otherID, swiperID}} {
		recipient, actor := pair[0], pair[1]
		conv, err := s.Conversation(ctx, recipient, matchID)
		if err != nil {
			return Conversation{}, err
		}
		if recipient == swiperID {
			mine = conv
		}
		s.pub.Publish(recipient, realtime.Event{Type: "match", Data: conv})
		if _, err := s.notifier.Notify(ctx, recipient, actor, matchID, notifications.TypeMatch); err != nil {
			log.Printf(`{"event":"notify_error","operation":"chat.match","error":%q}`, err.Error())
		}
	}
	return mine, nil
}

// cleanBody trims and validates a message: 1..2000 runes, no control characters except newlines/tabs.
func cleanBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", platformerrors.BadRequest("empty_message", "message cannot be empty")
	}
	if !utf8.ValidString(body) || utf8.RuneCountInString(body) > maxMessageRunes {
		return "", platformerrors.BadRequest("message_too_long", "message must be at most 2000 characters")
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", platformerrors.BadRequest("invalid_message", "message contains invalid characters")
		}
	}
	return body, nil
}
