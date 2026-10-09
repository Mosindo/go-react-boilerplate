package notifications

import "context"

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (NotificationsResponse, error) {
	list, unread, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return NotificationsResponse{}, err
	}
	return NotificationsResponse{Notifications: list, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
