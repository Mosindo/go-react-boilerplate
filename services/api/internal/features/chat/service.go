package chat

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo     Repository
	signer   *media.Signer
	pub      realtime.Publisher
	notifier notifications.Notifier
}

func NewService(repo Repository, signer *media.Signer, pub realtime.Publisher, n notifications.Notifier) *Service {
	return &Service{repo: repo, signer: signer, pub: pub, notifier: n}
}

var errNotFound = platformerrors.New(http.StatusNotFound, "not_found", "conversation not found")

// CleanMessage trims the body, removes control characters and enforces the size limit.
func CleanMessage(body string) (string, error) {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || (r >= 0x20 && r != 0x7f) {
			return r
		}
		return -1
	}, body))
	if cleaned == "" {
		return "", platformerrors.New(http.StatusBadRequest, "validation", "message is empty")
	}
	if utf8.RuneCountInString(cleaned) > MaxMessageLength {
		return "", platformerrors.New(http.StatusBadRequest, "validation", "message is too long (max 2000 characters)")
	}
	return cleaned, nil
}

// member loads the caller's access to a conversation. Non-members get a 404 so
// the existence of other people's conversations is never revealed.
func (s *Service) member(ctx context.Context, conversationID, userID string) (access, error) {
	a, err := s.repo.Access(ctx, conversationID, userID)
	if errors.Is(err, ErrNotFound) {
		return access{}, errNotFound
	}
	return a, err
}

func (s *Service) Conversations(ctx context.Context, userID, cursor string, limit int) (ConversationsResponse, error) {
	var before *time.Time
	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return ConversationsResponse{}, platformerrors.New(http.StatusBadRequest, "validation", "invalid cursor")
		}
		before = &t
	}
	rows, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: []Conversation{}}
	if len(rows) > limit {
		rows = rows[:limit]
		resp.NextCursor = rows[len(rows)-1].SortAt.UTC().Format(time.RFC3339Nano)
	}
	for _, r := range rows {
		c := r.Conversation
		if r.PhotoID != nil {
			u := s.signer.URL(*r.PhotoID)
			c.With.PhotoURL = &u
		}
		resp.Conversations = append(resp.Conversations, c)
	}
	return resp, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, beforeRaw string, limit int) (MessagesResponse, error) {
	a, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if a.Blocked {
		return MessagesResponse{}, errNotFound
	}
	var before int64
	if beforeRaw != "" {
		before, err = strconv.ParseInt(beforeRaw, 10, 64)
		if err != nil || before < 0 {
			return MessagesResponse{}, platformerrors.New(http.StatusBadRequest, "validation", "invalid before")
		}
	}
	msgs, err := s.repo.ListMessages(ctx, conversationID, a.HiddenAt, before, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		resp.NextBefore = msgs[len(msgs)-1].ID
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	resp.Messages = msgs
	peer, err := s.repo.PeerLastRead(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp.PeerLastReadID = strconv.FormatInt(peer, 10)
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	cleaned, err := CleanMessage(body)
	if err != nil {
		return Message{}, err
	}
	a, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	if a.Blocked {
		return Message{}, platformerrors.New(http.StatusForbidden, "blocked", "you cannot message this person")
	}
	m, err := s.repo.AddMessage(ctx, conversationID, userID, cleaned)
	if err != nil {
		return Message{}, err
	}
	ev := realtime.Event{Type: "message.new", Data: m}
	s.pub.Publish(userID, ev)
	s.pub.Publish(a.OtherUserID, ev)
	if !s.pub.IsOnline(a.OtherUserID) {
		s.notifier.NotifyOnce(ctx, a.OtherUserID, notifications.TypeNewMessage, "Nouveau message",
			"Vous avez reçu un nouveau message.", map[string]string{"conversationId": conversationID}, "conversationId")
	}
	return m, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	a, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	last, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !a.Blocked {
		s.pub.Publish(a.OtherUserID, realtime.Event{Type: "conversation.read", Data: map[string]string{
			"conversationId": conversationID, "readerId": userID, "lastReadId": strconv.FormatInt(last, 10),
		}})
	}
	return nil
}

// Hide removes the conversation from the caller's list (local deletion).
// It comes back only when a new message arrives.
func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.member(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}

// Unmatch ends the match for both people and deletes the conversation.
func (s *Service) Unmatch(ctx context.Context, userID, conversationID string) error {
	a, err := s.member(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if err := s.repo.Unmatch(ctx, conversationID); err != nil {
		return err
	}
	ev := realtime.Event{Type: "conversation.removed", Data: map[string]string{"conversationId": conversationID}}
	s.pub.Publish(userID, ev)
	s.pub.Publish(a.OtherUserID, ev)
	return nil
}
