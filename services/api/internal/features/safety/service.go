package safety

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/signing"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrSelf          = errors.New("you cannot do this to yourself")
	ErrInvalidReason = errors.New("invalid report reason")
)

type Service struct {
	repo   Repository
	signer *signing.Signer
	pub    realtime.Publisher
}

func NewService(repo Repository, signer *signing.Signer, pub realtime.Publisher) *Service {
	return &Service{repo: repo, signer: signer, pub: pub}
}

func (s *Service) Block(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return ErrSelf
	}
	ended, err := s.repo.Block(ctx, userID, targetID)
	if err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	for _, id := range ended {
		s.pub.Publish(targetID, realtime.Event{Type: "match_ended", Data: map[string]string{"matchId": id}})
	}
	return nil
}

func (s *Service) Unblock(ctx context.Context, userID, targetID string) error {
	if err := s.repo.Unblock(ctx, userID, targetID); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ListBlocks(ctx context.Context, userID string) (BlocksResponse, error) {
	rows, err := s.repo.ListBlocks(ctx, userID)
	if err != nil {
		return BlocksResponse{}, err
	}
	out := BlocksResponse{Blocks: make([]BlockedUser, 0, len(rows))}
	for _, r := range rows {
		b := r.BlockedUser
		if r.photoID != nil {
			u := s.signer.PhotoURL(*r.photoID)
			b.PhotoURL = &u
		}
		out.Blocks = append(out.Blocks, b)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, userID string, req ReportRequest) error {
	if userID == req.UserID {
		return ErrSelf
	}
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if _, ok := validReasons[reason]; !ok {
		return ErrInvalidReason
	}
	details := strings.TrimSpace(req.Details)
	if !utf8.ValidString(details) {
		return ErrInvalidReason
	}
	if err := s.repo.CreateReport(ctx, userID, req.UserID, reason, details); err != nil {
		if errors.Is(err, ErrRepositoryUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}
