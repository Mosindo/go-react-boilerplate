package safety

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrSelfAction    = apperr.Validation("self_action", "you cannot do this on your own account")
	ErrUserNotFound  = apperr.New(http.StatusNotFound, "user_not_found", "user not found")
	ErrInvalidReason = apperr.Validation("invalid_reason", "invalid report reason")
	ErrDetailsLength = apperr.Validation("invalid_details", "details must be at most 1000 characters")
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelfAction
	}
	matchID, err := s.repo.Block(ctx, userID, targetID)
	if errors.Is(err, errRepoUserNotFound) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if matchID != "" {
		event := realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": matchID}}
		s.pub.Publish(userID, event)
		s.pub.Publish(targetID, event)
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, targetID)
}

func (s *Service) ListBlocks(ctx context.Context, userID string, limit, offset int) ([]BlockedUser, error) {
	return s.repo.ListBlocks(ctx, userID, limit, offset)
}

// Report stores a moderation report and, unless disabled, blocks the user.
func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (ReportResponse, error) {
	if userID == req.UserID {
		return ReportResponse{}, ErrSelfAction
	}
	if !slices.Contains(ReportReasons, req.Reason) {
		return ReportResponse{}, ErrInvalidReason
	}
	details := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, req.Details))
	if utf8.RuneCountInString(details) > 1000 {
		return ReportResponse{}, ErrDetailsLength
	}
	id, err := s.repo.CreateReport(ctx, userID, req.UserID, req.Reason, details)
	if errors.Is(err, errRepoUserNotFound) {
		return ReportResponse{}, ErrUserNotFound
	}
	if err != nil {
		return ReportResponse{}, err
	}
	resp := ReportResponse{ID: id}
	if req.Block == nil || *req.Block {
		if err := s.Block(ctx, userID, req.UserID); err != nil {
			return ReportResponse{}, err
		}
		resp.Blocked = true
	}
	return resp, nil
}
