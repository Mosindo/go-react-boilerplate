package moderation

import (
	"context"
	"errors"
	"log"
	"net/http"

	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
)

var (
	ErrNotModerator   = apperr.Forbidden("moderator role required")
	ErrReportNotFound = apperr.NotFound("report not found or already resolved")
	ErrUserNotFound   = apperr.NotFound("user not found")
	ErrProtected      = apperr.New(http.StatusForbidden, "protected_account", "moderator accounts cannot be suspended here")
	ErrInvalidStatus  = apperr.Validation("status must be open, reviewed or dismissed")
)

type ProfileProvider interface {
	Summaries(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Summary, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileProvider
}

func NewService(repo Repository, profiles ProfileProvider) *Service {
	return &Service{repo: repo, profiles: profiles}
}

func (s *Service) IsModerator(ctx context.Context, userID string) (bool, error) {
	role, err := s.repo.Role(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return false, nil
	}
	return role == RoleModerator, err
}

func (s *Service) ListReports(ctx context.Context, moderatorID, status string, limit, offset int) (ReportsResponse, error) {
	if status == "" {
		status = StatusOpen
	}
	if status != StatusOpen && status != StatusReviewed && status != StatusDismissed {
		return ReportsResponse{}, ErrInvalidStatus
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.repo.ListReports(ctx, status, limit+1, offset)
	if err != nil {
		return ReportsResponse{}, err
	}
	resp := ReportsResponse{}
	if len(rows) > limit {
		rows = rows[:limit]
		next := offset + limit
		resp.NextOffset = &next
	}

	ids := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if row.ReportedID != nil && !seen[*row.ReportedID] {
			seen[*row.ReportedID] = true
			ids = append(ids, *row.ReportedID)
		}
	}
	summaries, err := s.profiles.Summaries(ctx, moderatorID, ids)
	if err != nil {
		return ReportsResponse{}, err
	}
	cards, err := s.profiles.Cards(ctx, moderatorID, ids)
	if err != nil {
		return ReportsResponse{}, err
	}
	cardByID := map[string]profiles.PublicProfile{}
	for _, card := range cards {
		card.DistanceKm = nil // irrelevant for moderation
		cardByID[card.UserID] = card
	}

	resp.Reports = make([]Report, 0, len(rows))
	for _, row := range rows {
		report := Report{
			ID: row.ID, Reason: row.Reason, Details: row.Details, Status: row.Status,
			CreatedAt: row.CreatedAt, ReviewedAt: row.ReviewedAt, OpenReports: row.OpenReports, Suspended: row.Suspended,
		}
		if row.ReportedID != nil {
			if summary, ok := summaries[*row.ReportedID]; ok {
				report.ReportedUser = &summary
			}
			if card, ok := cardByID[*row.ReportedID]; ok {
				report.ReportedProfile = &card
			}
		}
		resp.Reports = append(resp.Reports, report)
	}
	return resp, nil
}

func (s *Service) ResolveReport(ctx context.Context, moderatorID, reportID, status string) error {
	if status != StatusReviewed && status != StatusDismissed {
		return ErrInvalidStatus
	}
	if err := s.repo.ResolveReport(ctx, reportID, moderatorID, status); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrReportNotFound
		}
		return err
	}
	log.Printf(`{"event":"moderation_report_resolved","report_id":%q,"status":%q,"moderator_id":%q}`, reportID, status, moderatorID)
	return nil
}

func (s *Service) SetSuspended(ctx context.Context, moderatorID, userID string, suspended bool) error {
	if moderatorID == userID {
		return ErrProtected
	}
	if err := s.repo.SetSuspended(ctx, userID, moderatorID, suspended); err != nil {
		switch {
		case errors.Is(err, ErrRepositoryNotFound):
			return ErrUserNotFound
		case errors.Is(err, ErrRepositoryProtected):
			return ErrProtected
		}
		return err
	}
	log.Printf(`{"event":"moderation_suspension","user_id":%q,"suspended":%t,"moderator_id":%q}`, userID, suspended, moderatorID)
	return nil
}

// SetRole is only exposed through the admin command line, never over HTTP.
func (s *Service) SetRole(ctx context.Context, email, role string) error {
	if role != RoleMember && role != RoleModerator {
		return apperr.Validation("role must be member or moderator")
	}
	if err := s.repo.SetRoleByEmail(ctx, email, role); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}
