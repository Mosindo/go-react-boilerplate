package notifications

import "context"

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (NotificationsResponse, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)
	offset = max(offset, 0)

	items, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return NotificationsResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return NotificationsResponse{}, err
	}
	return NotificationsResponse{Notifications: items, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
