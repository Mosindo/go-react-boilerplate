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

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.ListWithPagination(ctx, "", defaultUsersLimit, 0)
}

func (s *Service) ListWithPagination(ctx context.Context, organizationID string, limit, offset int) ([]User, error) {
	normalizedLimit := normalizeUsersLimit(limit)
	normalizedOffset := normalizeOffset(offset)
	users, err := s.repo.ListUsers(ctx, organizationID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]User, 0, len(users))
	for _, user := range users {
		payload = append(payload, User{
			ID:             user.ID,
			Email:          user.Email,
			OrganizationID: user.OrganizationID,
			CreatedAt:      user.CreatedAt,
		})
	}
	return payload, nil
}

func (s *Service) GetByID(ctx context.Context, organizationID, userID string) (User, error) {
	user, err := s.repo.GetUserByID(ctx, organizationID, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}
	return User{ID: user.ID, Email: user.Email, OrganizationID: user.OrganizationID, CreatedAt: user.CreatedAt}, nil
}

func normalizeUsersLimit(limit int) int {
	if limit <= 0 {
		return defaultUsersLimit
	}
	if limit > maxUsersLimit {
		return maxUsersLimit
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
