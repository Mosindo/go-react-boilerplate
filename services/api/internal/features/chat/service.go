package chat

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/signedurl"
)

var (
	ErrNotFound   = platformerrors.New(http.StatusNotFound, "not_found", "conversation not found")
	ErrBadCursor  = platformerrors.New(http.StatusBadRequest, "invalid_request", "invalid cursor")
	ErrEmptyBody  = platformerrors.New(http.StatusBadRequest, "invalid_request", "message must contain 1 to 2000 characters")
	ErrBadMessage = platformerrors.New(http.StatusBadRequest, "invalid_request", "invalid message id")
)

const MaxMessageLength = 2000

type Service struct {
	repo   *Repository
	signer *signedurl.Signer
	pub    realtime.Publisher
}

func NewService(repo *Repository, signer *signedurl.Signer, pub realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, pub: pub}
}

func (s *Service) finish(rows []convRow) []Conversation {
	now := time.Now()
	out := make([]Conversation, 0, len(rows))
	for _, r := range rows {
		c := r.Conversation
		if r.photoID != nil {
			u := s.signer.PhotoURL(*r.photoID, now)
			c.User.PhotoURL = &u
		}
		out = append(out, c)
	}
	return out
}

// Summary returns one conversation as seen by viewer (hidden or not).
func (s *Service) Summary(ctx context.Context, convID, viewerID string) (Conversation, error) {
	rows, err := s.repo.conversations(ctx, viewerID, convFilter{convID: &convID, limit: 1})
	if err != nil {
		return Conversation{}, err
	}
	if len(rows) == 0 {
		return Conversation{}, ErrNotFound
	}
	return s.finish(rows)[0], nil
}

func (s *Service) List(ctx context.Context, viewerID, rawCursor string, limit int) (ConversationsPage, error) {
	at, id, ok, err := httpx.DecodeCursor(rawCursor)
	if err != nil {
		return ConversationsPage{}, ErrBadCursor
	}
	f := convFilter{limit: limit + 1}
	if ok {
		f.cursor = &cursor{at: at, id: id}
	}
	rows, err := s.repo.conversations(ctx, viewerID, f)
	if err != nil {
		return ConversationsPage{}, err
	}
	page := ConversationsPage{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next := httpx.EncodeCursor(last.LastActivityAt, last.ID)
		page.NextCursor = &next
	}
	page.Conversations = s.finish(rows)
	return page, nil
}

func (s *Service) Hide(ctx context.Context, convID, userID string) error {
	return mapParticipant(s.repo.Hide(ctx, convID, userID))
}

func (s *Service) Messages(ctx context.Context, convID, userID string, before string, limit int) (MessagesPage, error) {
	ok, err := s.repo.IsParticipant(ctx, convID, userID)
	if err != nil {
		return MessagesPage{}, err
	}
	if !ok {
		return MessagesPage{}, ErrNotFound
	}
	var beforePtr *string
	if before != "" {
		if len(before) != 36 {
			return MessagesPage{}, ErrBadMessage
		}
		beforePtr = &before
	}
	msgs, readAt, err := s.repo.Messages(ctx, convID, userID, beforePtr, limit+1)
	if err != nil {
		return MessagesPage{}, err
	}
	page := MessagesPage{OtherLastReadAt: readAt}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		next := msgs[len(msgs)-1].ID
		page.NextCursor = &next
	}
	page.Messages = msgs
	return page, nil
}

// CleanBody trims the text and rejects empty, oversized or control-character content.
func CleanBody(body string) (string, error) {
	body = strings.Map(func(r rune) rune {
		if r == '\r' || (unicode.IsControl(r) && r != '\n' && r != '\t') {
			return -1
		}
		return r
	}, body)
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > MaxMessageLength {
		return "", ErrEmptyBody
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, convID, senderID, body string) (Message, error) {
	clean, err := CleanBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, other, err := s.repo.Insert(ctx, convID, senderID, clean)
	if err != nil {
		return Message{}, mapParticipant(err)
	}
	s.pub.Publish(other, "message.new", msg)
	s.pub.Publish(senderID, "message.new", msg)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, convID, userID string) error {
	other, readAt, err := s.repo.MarkRead(ctx, convID, userID)
	if err != nil {
		return mapParticipant(err)
	}
	s.pub.Publish(other, "message.read", map[string]any{"conversationId": convID, "readerId": userID, "readAt": readAt})
	return nil
}

func mapParticipant(err error) error {
	if errors.Is(err, errNotParticipant) {
		return ErrNotFound
	}
	return err
}
