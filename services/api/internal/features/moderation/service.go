package moderation

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"example.com/api/internal/platform/realtime"
)

var ErrInvalidReason = errors.New("reason must be one of spam, fake, harassment, inappropriate, underage, other")

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	if pub == nil {
		pub = realtime.NopPublisher{}
	}
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if err := s.repo.Block(ctx, blockerID, blockedID); err != nil {
		return err
	}
	s.pub.Publish(blockedID, realtime.Event{Type: "match.removed", Data: map[string]string{"userId": blockerID}})
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) List(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID, 200)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	valid := false
	for _, r := range Reasons {
		if r == reason {
			valid = true
		}
	}
	if !valid {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, req.Details))
	if len([]rune(details)) > 1000 {
		details = string([]rune(details)[:1000])
	}
	// Report first: blocking removes the match whose messages are the evidence.
	if err := s.repo.Report(ctx, reporterID, req.UserID, reason, details); err != nil {
		return err
	}
	if req.Block {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
