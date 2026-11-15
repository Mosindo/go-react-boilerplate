package safety

import (
	"context"
	"slices"
	"strings"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/textx"
)

var (
	ErrSelfAction    = apperr.BadRequest("self_action", "you cannot do this on your own profile")
	ErrUserNotFound  = apperr.NotFound("user_not_found", "user not found")
	ErrBlockNotFound = apperr.NotFound("block_not_found", "this user is not blocked")
	ErrInvalidReason = apperr.BadRequest("invalid_reason", "invalid report reason")
	ErrDetailsLength = apperr.BadRequest("details_too_long", "details must be at most 1000 characters")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return ErrSelfAction
	}
	matchID, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		return err
	}
	s.announceRemoval(userID, targetID, matchID)
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	return s.repo.Unblock(ctx, userID, strings.ToLower(targetID))
}

func (s *Service) ListBlocks(ctx context.Context, userID string) (BlocksResponse, error) {
	blocks, err := s.repo.ListBlocks(ctx, userID, 200)
	if err != nil {
		return BlocksResponse{}, err
	}
	return BlocksResponse{Blocks: blocks}, nil
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) (ReportResponse, error) {
	targetID := strings.ToLower(req.UserID)
	if userID == targetID {
		return ReportResponse{}, ErrSelfAction
	}
	if !slices.Contains(ReportReasons, req.Reason) {
		return ReportResponse{}, ErrInvalidReason
	}
	details := textx.Clean(req.Details, true)
	if textx.Len(details) > 1000 {
		return ReportResponse{}, ErrDetailsLength
	}
	resp, matchID, err := s.repo.Report(ctx, userID, targetID, req.Reason, details)
	if err != nil {
		return ReportResponse{}, err
	}
	s.announceRemoval(userID, targetID, matchID)
	return resp, nil
}

// announceRemoval tells both clients that the match disappeared, without
// revealing to the blocked user that a block happened.
func (s *Service) announceRemoval(userID, targetID, matchID string) {
	if matchID == "" || s.publisher == nil {
		return
	}
	event := realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(targetID, event)
}
