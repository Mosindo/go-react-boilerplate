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
	ErrSelfAction    = errors.New("you cannot do this to yourself")
	ErrInvalidReason = errors.New("invalid report reason")
	ErrDetailsLength = errors.New("details must be at most 1000 characters")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return ErrSelfAction
	}
	matchID, err := s.repo.Block(ctx, blockerID, blockedID)
	if err != nil {
		return err
	}
	// The blocked person sees the same event as an unmatch: the block itself
	// is never disclosed.
	if matchID != "" && s.publisher != nil {
		s.publisher.Publish([]string{blockerID, blockedID}, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) ListBlocks(ctx context.Context, blockerID string) (BlocksResponse, error) {
	blocks, err := s.repo.ListBlocks(ctx, blockerID)
	if err != nil {
		return BlocksResponse{}, err
	}
	return BlocksResponse{Blocks: blocks}, nil
}

func validReason(reason string) bool {
	for _, r := range ReportReasons {
		if r == reason {
			return true
		}
	}
	return false
}

// Report files a report and always blocks the reported person, so the
// reporter is never exposed to them again.
func (s *Service) Report(ctx context.Context, reporterID string, req ReportRequest) (ReportResponse, error) {
	reportedID := strings.ToLower(strings.TrimSpace(req.UserID))
	if reporterID == reportedID {
		return ReportResponse{}, ErrSelfAction
	}
	reason := strings.TrimSpace(req.Reason)
	if !validReason(reason) {
		return ReportResponse{}, ErrInvalidReason
	}
	details := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, req.Details))
	if utf8.RuneCountInString(details) > MaxReportDetails {
		return ReportResponse{}, ErrDetailsLength
	}
	id, err := s.repo.Report(ctx, reporterID, reportedID, reason, details)
	if err != nil {
		return ReportResponse{}, err
	}
	if err := s.Block(ctx, reporterID, reportedID); err != nil {
		return ReportResponse{}, err
	}
	return ReportResponse{ID: id, Blocked: true}, nil
}
