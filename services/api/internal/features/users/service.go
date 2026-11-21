package users

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Service struct {
	repo Repository
}

const (
	defaultUsersLimit = 20
	maxUsersLimit     = 100
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListWithPagination(ctx context.Context, organizationID, viewerID string, limit, offset int) ([]User, error) {
	if limit <= 0 {
		limit = defaultUsersLimit
	}
	if limit > maxUsersLimit {
		limit = maxUsersLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListUsers(ctx, organizationID, viewerID, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, organizationID, viewerID, userID string) (User, error) {
	return s.repo.GetUserByID(ctx, organizationID, viewerID, userID)
}
