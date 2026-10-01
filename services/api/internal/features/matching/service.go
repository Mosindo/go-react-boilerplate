package matching

import (
	"context"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// ProfileReader attaches public profiles to ids (profiles.Service).
type ProfileReader interface {
	ListPublic(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

// Notifier persists + pushes an in-app notification (notifications.Service).
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

// Publisher pushes a realtime event (realtime.Hub).
type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	notifier Notifier
	pub      Publisher
}

func NewService(repo Repository, profiles ProfileReader, notifier Notifier, pub Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, pub: pub}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	if limit <= 0 {
		limit = DefaultDiscoverLimit
	}
	if limit > MaxDiscoverLimit {
		limit = MaxDiscoverLimit
	}
	_ = s.repo.Touch(ctx, userID) // activity signal for ranking; failure is harmless
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.ordered(ctx, userID, ids)
}

// ordered loads public profiles in one query and restores the ranked order.
func (s *Service) ordered(ctx context.Context, userID string, ids []string) ([]profiles.PublicProfile, error) {
	list, err := s.profiles.ListPublic(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(list))
	for _, p := range list {
		byID[p.UserID] = p
	}
	out := make([]profiles.PublicProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.UserID == userID {
		return SwipeResponse{}, ErrNotEligible
	}
	ok, err := s.repo.Eligible(ctx, userID, req.UserID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ok {
		return SwipeResponse{}, ErrNotEligible
	}

	res, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		return SwipeResponse{}, err
	}
	out := SwipeResponse{Action: res.Action, AlreadySwiped: !res.Inserted}
	if res.MatchID == "" {
		return out, nil
	}
	out.Matched = true
	row := MatchRow{ID: res.MatchID, CreatedAt: res.MatchedAt, OtherUserID: req.UserID, ConversationID: res.ConversationID}
	summary, err := s.summarize(ctx, userID, row)
	if err != nil {
		return SwipeResponse{}, err
	}
	out.Match = &summary
	if res.Inserted {
		s.announceMatch(ctx, userID, req.UserID, row)
	}
	return out, nil
}

func (s *Service) summarize(ctx context.Context, viewerID string, row MatchRow) (MatchSummary, error) {
	list, err := s.profiles.ListPublic(ctx, viewerID, []string{row.OtherUserID})
	if err != nil {
		return MatchSummary{}, err
	}
	summary := MatchSummary{ID: row.ID, CreatedAt: row.CreatedAt, ConversationID: row.ConversationID, HasMessages: row.HasMessages}
	if len(list) > 0 {
		summary.User = list[0]
	}
	return summary, nil
}

func (s *Service) announceMatch(ctx context.Context, userID, otherID string, row MatchRow) {
	for _, pair := range [][2]string{{userID, otherID}, {otherID, userID}} {
		recipient, counterpart := pair[0], pair[1]
		name := "Quelqu'un"
		if list, err := s.profiles.ListPublic(ctx, recipient, []string{counterpart}); err == nil && len(list) > 0 {
			name = list[0].FirstName
		}
		d := map[string]string{"matchId": row.ID, "conversationId": row.ConversationID, "userId": counterpart}
		_ = s.notifier.Notify(ctx, recipient, "match", "C'est un match !", "Vous et "+name+" vous plaisez mutuellement. Dites bonjour !", d)
		s.pub.Publish(recipient, realtime.Event{Type: "match", Data: d})
	}
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchSummary, error) {
	rows, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	list, err := s.profiles.ListPublic(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(list))
	for _, p := range list {
		byID[p.UserID] = p
	}
	out := make([]MatchSummary, 0, len(rows))
	for _, r := range rows {
		p, ok := byID[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, MatchSummary{ID: r.ID, CreatedAt: r.CreatedAt, ConversationID: r.ConversationID, HasMessages: r.HasMessages, User: p})
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	return nil
}
