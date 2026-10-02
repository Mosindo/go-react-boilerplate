package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

var (
	ErrSelf          = errors.New("you cannot do this to yourself")
	ErrUnknownUser   = errors.New("user not found")
	ErrInvalidReason = errors.New("unknown report reason")
	ErrDetailsLong   = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func parseTarget(me, raw string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(raw))
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrUnknownUser
	}
	if id == me {
		return "", ErrSelf
	}
	return id, nil
}

func (s *Service) Block(ctx context.Context, me, target string) error {
	id, err := parseTarget(me, target)
	if err != nil {
		return err
	}
	matchID, err := s.repo.Block(ctx, me, id)
	if err != nil {
		if errors.Is(err, ErrRepoUnknownUser) {
			return ErrUnknownUser
		}
		return err
	}
	if matchID != "" {
		s.events.Publish(id, realtime.Event{Type: "unmatched", Payload: map[string]string{"matchId": matchID}})
		s.events.Publish(me, realtime.Event{Type: "unmatched", Payload: map[string]string{"matchId": matchID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, me, target string) error {
	id, err := parseTarget(me, target)
	if err != nil {
		return err
	}
	return s.repo.Unblock(ctx, me, id)
}

func (s *Service) ListBlocked(ctx context.Context, me string) ([]BlockedUser, error) {
	list, err := s.repo.ListBlocked(ctx, me)
	if list == nil {
		list = []BlockedUser{}
	}
	return list, err
}

func (s *Service) Report(ctx context.Context, me string, req ReportRequest) error {
	id, err := parseTarget(me, req.UserID)
	if err != nil {
		return err
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	valid := false
	for _, r := range Reasons {
		valid = valid || r == reason
	}
	if !valid {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return ErrDetailsLong
	}
	if err := s.repo.CreateReport(ctx, me, id, reason, details); err != nil {
		if errors.Is(err, ErrRepoUnknownUser) {
			return ErrUnknownUser
		}
		return err
	}
	if req.Block {
		return s.Block(ctx, me, id)
	}
	return nil
}
