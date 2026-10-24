package safety

import (
	"context"
	"strings"
	"unicode"

	"example.com/api/internal/platform/realtime"
)

// ValidationError is a client-fixable input problem.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfTarget
	}
	convID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if convID != "" {
		ev := realtime.Event{Type: "match.removed", Data: map[string]string{"conversationId": convID}}
		s.pub.Publish(blockedID, ev)
		s.pub.Publish(blockerID, ev)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocked(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocked(ctx, blockerID, limit, offset)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelfTarget
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return ValidationError{Message: "invalid reason"}
	}
	details := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, strings.TrimSpace(req.Details))
	if err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details); err != nil {
		return err
	}
	if req.Block {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}
