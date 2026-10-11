package discovery

import (
	"context"

	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/users"
	"example.com/api/internal/platform/realtime"
)

// ProfileReader is the slice of the users feature discovery depends on.
type ProfileReader interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]users.PublicProfile, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
}

// ConversationReader is the slice of the chat feature discovery depends on.
type ConversationReader interface {
	Summary(ctx context.Context, convID, viewerID string) (chat.Conversation, error)
}

type Service struct {
	repo     *Repository
	profiles ProfileReader
	convs    ConversationReader
	pub      realtime.Publisher
}

func NewService(repo *Repository, profiles ProfileReader, convs ConversationReader, pub realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, convs: convs, pub: pub}
}

func (s *Service) requireReady(ctx context.Context, userID string) error {
	ok, err := s.repo.ViewerReady(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProfileRequired
	}
	return nil
}

func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]users.PublicProfile, error) {
	if err := s.requireReady(ctx, viewerID); err != nil {
		return nil, err
	}
	s.repo.TouchActivity(ctx, viewerID)
	ids, err := s.repo.CandidateIDs(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.profiles.PublicProfiles(ctx, viewerID, ids)
}

func (s *Service) LikesReceived(ctx context.Context, viewerID string, limit int) ([]users.PublicProfile, error) {
	if err := s.requireReady(ctx, viewerID); err != nil {
		return nil, err
	}
	ids, err := s.repo.LikesReceivedIDs(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.profiles.PublicProfiles(ctx, viewerID, ids)
}

func (s *Service) Swipe(ctx context.Context, from, to, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, ErrBadAction
	}
	if from == to {
		return SwipeResponse{}, ErrSelfSwipe
	}
	if err := s.requireReady(ctx, from); err != nil {
		return SwipeResponse{}, err
	}
	visible, err := s.profiles.CanView(ctx, from, to)
	if err != nil {
		return SwipeResponse{}, err
	}
	allowed := false
	if visible {
		if allowed, err = s.repo.CanSwipe(ctx, from, to); err != nil {
			return SwipeResponse{}, err
		}
	}
	if !allowed {
		return SwipeResponse{}, ErrNotFound
	}

	out, err := s.repo.Swipe(ctx, from, to, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Action: out.Action, Matched: out.Matched}
	if out.Matched {
		conv, err := s.convs.Summary(ctx, out.ConversationID, from)
		if err != nil {
			return SwipeResponse{}, err
		}
		resp.Conversation = &conv
		if out.NewMatch {
			if theirs, err := s.convs.Summary(ctx, out.ConversationID, to); err == nil {
				s.pub.Publish(to, "match.new", theirs)
			}
			s.pub.Publish(from, "match.new", conv)
		}
	}
	for _, n := range out.Notifications {
		s.pub.Publish(n.UserID, "notification.new", n.Notification)
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, matchID, userID string) error {
	rm, err := s.repo.DeleteMatch(ctx, matchID, userID)
	if err != nil {
		return err
	}
	if rm == nil {
		return ErrNotFound
	}
	for _, id := range []string{rm.UserA, rm.UserB} {
		s.pub.Publish(id, "conversation.removed", map[string]string{"conversationId": rm.ConversationID})
	}
	return nil
}
