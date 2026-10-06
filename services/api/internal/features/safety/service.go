package safety

import (
	"context"
	"strings"
	"unicode"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	hub  *realtime.Hub
}

func NewService(repo Repository, hub *realtime.Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	if err := s.requireUser(ctx, blockedID); err != nil {
		return err
	}
	ended, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	for _, matchID := range ended {
		s.hub.Publish(blockedID, realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListBlocks(ctx, blockerID, limit, offset)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) error {
	if reporterID == req.UserID {
		return ErrSelfAction
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return ErrInvalidReason
	}
	if err := s.requireUser(ctx, req.UserID); err != nil {
		return err
	}
	if _, err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, cleanText(req.Details)); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, reporterID, req.UserID)
	}
	return nil
}

func (s *Service) requireUser(ctx context.Context, userID string) error {
	ok, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	return nil
}

func cleanText(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	text := b.String()
	if runes := []rune(text); len(runes) > 1000 {
		text = string(runes[:1000])
	}
	return text
}
