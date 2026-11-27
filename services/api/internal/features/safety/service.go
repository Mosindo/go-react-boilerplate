package safety

import (
	"context"
	"log"
	"slices"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NopPublisher{}
	}
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) validateTarget(ctx context.Context, organizationID, actorID, targetID string) error {
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	if !httpx.IsUUID(targetID) {
		return httpx.Invalid("invalid_id", "Identifiant invalide.")
	}
	if targetID == actorID {
		return httpx.Invalid("invalid_target", "Action impossible sur ton propre compte.")
	}
	ok, err := s.repo.UserInOrganization(ctx, organizationID, targetID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}
	return nil
}

func (s *Service) Block(ctx context.Context, organizationID, blockerID, blockedID string) error {
	blockedID = strings.ToLower(strings.TrimSpace(blockedID))
	if err := s.validateTarget(ctx, organizationID, blockerID, blockedID); err != nil {
		return err
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	if matchID != "" {
		payload := map[string]string{"matchId": matchID}
		s.publisher.Publish(blockedID, "match.removed", payload)
		s.publisher.Publish(blockerID, "match.removed", payload)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, blockerID)
}

// Report files a moderation report (and blocks by default).
func (s *Service) Report(ctx context.Context, organizationID, reporterID string, req ReportRequest) (ReportResponse, error) {
	reportedID := strings.ToLower(strings.TrimSpace(req.UserID))
	if err := s.validateTarget(ctx, organizationID, reporterID, reportedID); err != nil {
		return ReportResponse{}, err
	}
	reason := strings.TrimSpace(req.Reason)
	if !slices.Contains(ReportReasons, reason) {
		return ReportResponse{}, httpx.Invalid("invalid_reason", "Motif de signalement invalide.")
	}
	details := strings.TrimSpace(req.Details)
	if utf8.RuneCountInString(details) > MaxReportDetailsRunes {
		return ReportResponse{}, httpx.Invalid("details_too_long", "Les détails ne peuvent pas dépasser 1000 caractères.")
	}

	// Snapshot the conversation before blocking deletes it.
	if err := s.repo.CreateReport(ctx, reporterID, reportedID, reason, details); err != nil {
		return ReportResponse{}, err
	}
	hidden, err := s.repo.AutoHideIfNeeded(ctx, reportedID, AutoHideReporterThreshold, AutoHideWindow)
	if err != nil {
		log.Printf(`{"event":"auto_hide_failed","error":%q}`, err.Error())
	} else if hidden {
		log.Printf(`{"event":"profile_auto_hidden","reason":"report_threshold"}`)
	}

	block := req.Block == nil || *req.Block
	if block {
		if err := s.Block(ctx, organizationID, reporterID, reportedID); err != nil {
			return ReportResponse{}, err
		}
	}
	return ReportResponse{Status: "received", Blocked: block}, nil
}
