package users

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListContacts(ctx context.Context, userID string, limit, offset int) ([]User, error) {
	return s.repo.ListContacts(ctx, userID, limit, offset)
}

func (s *Service) GetVisible(ctx context.Context, viewerID, userID string) (User, error) {
	return s.repo.GetVisible(ctx, viewerID, userID)
}
