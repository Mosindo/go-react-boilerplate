package safety

import (
	"context"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
)

type Disconnector interface {
	Disconnect(userID string)
}

type Service struct {
	repo Repository
	hub  Disconnector
}

func NewService(repo Repository, hub Disconnector) *Service { return &Service{repo: repo, hub: hub} }

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return httpx.Invalid("you cannot block yourself")
	}
	return s.repo.Block(ctx, blockerID, blockedID)
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID, limit, offset)
}

func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (Report, error) {
	if reporterID == req.UserID {
		return Report{}, httpx.Invalid("you cannot report yourself")
	}
	if _, ok := validReasons[req.Reason]; !ok {
		return Report{}, httpx.Invalid("invalid report reason")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > maxDetailsRunes {
		return Report{}, httpx.Invalid("details must be at most 1000 characters")
	}
	rep, err := s.repo.CreateReport(ctx, reporterID, req.UserID, req.Reason, details)
	if err != nil {
		return Report{}, err
	}
	if req.Block == nil || *req.Block {
		if err := s.repo.Block(ctx, reporterID, req.UserID); err != nil {
			return Report{}, err
		}
	}
	return rep, nil
}

func (s *Service) ListReports(ctx context.Context, status string, limit, offset int) ([]Report, error) {
	if status != "" && status != "open" && status != "reviewed" && status != "dismissed" {
		return nil, httpx.Invalid("invalid status filter")
	}
	return s.repo.ListReports(ctx, status, limit, offset)
}

func (s *Service) ResolveReport(ctx context.Context, reportID string, req ResolveReportRequest) (Report, error) {
	rep, err := s.repo.ResolveReport(ctx, reportID, req.Status, req.SuspendUser)
	if err != nil {
		return Report{}, err
	}
	if req.SuspendUser {
		s.hub.Disconnect(rep.ReportedUserID)
	}
	return rep, nil
}
