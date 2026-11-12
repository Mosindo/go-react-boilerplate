package users

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error) {
	return s.repo.List(ctx, organizationID, viewerID, limit, offset)
}

func (s *Service) Get(ctx context.Context, organizationID, viewerID, userID string) (User, error) {
	return s.repo.Get(ctx, organizationID, viewerID, userID)
}
