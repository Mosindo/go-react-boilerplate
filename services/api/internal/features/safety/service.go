package safety

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrInvalidTarget  = errors.New("invalid user")
	ErrInvalidReason  = errors.New("reason must be one of: spam, fake, harassment, inappropriate, underage, other")
	ErrDetailsTooLong = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.Discard{}
	}
	return &Service{repo: repo, events: events}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrInvalidTarget
	}
	ended, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if ended {
		// Tell the other side's clients to drop the conversation, without saying why.
		s.events.Publish(targetID, realtime.Event{Type: "unmatch"})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) Blocked(ctx context.Context, userID string) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, userID)
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if userID == req.UserID {
		return ErrInvalidTarget
	}
	valid := false
	for _, r := range ReportReasons {
		if r == req.Reason {
			valid = true
		}
	}
	if !valid {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, req.Details))
	if utf8.RuneCountInString(details) > 1000 {
		return ErrDetailsTooLong
	}
	if err := s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, userID, req.UserID)
	}
	return nil
}
