package safety

import (
	"context"
	"errors"
	"net/http"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

type Service struct {
	repo   Repository
	signer *media.Signer
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func validation(msg string) error {
	return platformerrors.New(http.StatusBadRequest, "validation", msg)
}

func translate(err error) error {
	if errors.Is(err, ErrUserNotFound) {
		return platformerrors.New(http.StatusNotFound, "not_found", "user not found")
	}
	return err
}

func (s *Service) Block(ctx context.Context, me, target string) error {
	if me == target {
		return validation("you cannot block yourself")
	}
	return translate(s.repo.Block(ctx, me, target))
}

func (s *Service) Unblock(ctx context.Context, me, target string) error {
	return s.repo.Unblock(ctx, me, target)
}

func (s *Service) Blocked(ctx context.Context, me string) ([]BlockedUser, error) {
	rows, err := s.repo.ListBlocked(ctx, me)
	if err != nil {
		return nil, err
	}
	out := make([]BlockedUser, 0, len(rows))
	for _, r := range rows {
		b := BlockedUser{UserID: r.UserID, FirstName: r.FirstName, BlockedAt: r.BlockedAt}
		if r.PhotoID != nil {
			u := s.signer.URL(*r.PhotoID)
			b.PhotoURL = &u
		}
		out = append(out, b)
	}
	return out, nil
}

func (s *Service) Report(ctx context.Context, me string, req ReportRequest) error {
	if me == req.UserID {
		return validation("you cannot report yourself")
	}
	if !validReasons[req.Reason] {
		return validation("invalid reason")
	}
	details := strings.TrimSpace(req.Details)
	if err := translate(s.repo.CreateReport(ctx, me, req.UserID, req.Reason, details)); err != nil {
		return err
	}
	if req.AlsoBlock {
		return s.Block(ctx, me, req.UserID)
	}
	return nil
}
