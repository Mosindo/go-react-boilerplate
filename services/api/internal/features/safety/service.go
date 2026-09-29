package safety

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      *Repository
	publisher realtime.Publisher
}

func NewService(repo *Repository, p realtime.Publisher) *Service {
	if p == nil {
		p = realtime.NopPublisher{}
	}
	return &Service{repo: repo, publisher: p}
}

// SanitizeDetails strips control characters (newline and tab are kept), trims the text and
// enforces the 1000 rune limit. ok is false when the cleaned text is too long.
func SanitizeDetails(s string) (string, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	return s, utf8.RuneCountInString(s) <= maxDetailsLen
}

func (s *Service) publishRemoved(rm *removedMatch) {
	if rm == nil {
		return
	}
	ev := realtime.Event{Type: "match.removed", Data: map[string]any{"conversationId": rm.ConversationID}}
	s.publisher.Publish(rm.UserA, ev)
	s.publisher.Publish(rm.UserB, ev)
}

func parseTarget(me, raw string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(raw))
	if !isUUID(id) {
		return "", ErrBadRequest
	}
	if id == me {
		return "", ErrSelf
	}
	return id, nil
}

func (s *Service) Block(ctx context.Context, me, raw string) error {
	target, err := parseTarget(me, raw)
	if err != nil {
		return err
	}
	rm, err := s.repo.Block(ctx, me, target)
	if err != nil {
		return err
	}
	s.publishRemoved(rm)
	return nil
}

func (s *Service) Unblock(ctx context.Context, me, raw string) error {
	target, err := parseTarget(me, raw)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, me, target)
}

func (s *Service) List(ctx context.Context, me string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, me)
}

func (s *Service) Report(ctx context.Context, me string, req reportRequest) (string, error) {
	target, err := parseTarget(me, req.UserID)
	if err != nil {
		return "", err
	}
	reason := strings.TrimSpace(req.Reason)
	if _, ok := validReasons[reason]; !ok {
		return "", ErrBadRequest
	}
	details, ok := SanitizeDetails(req.Details)
	if !ok {
		return "", ErrBadRequest
	}
	id, rm, err := s.repo.CreateReport(ctx, me, target, reason, details, req.Block)
	if err != nil {
		return "", err
	}
	s.publishRemoved(rm)
	return id, nil
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, ch := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if ch != '-' {
				return false
			}
		case (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'):
		default:
			return false
		}
	}
	return true
}
