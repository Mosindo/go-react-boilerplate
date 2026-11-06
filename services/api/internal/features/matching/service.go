package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var ErrInvalidAction = errors.New("action must be like or pass")

// Notifier creates user notifications (implemented by the notifications feature).
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

// CardLoader batch-loads photos and interests for a set of users.
type CardLoader interface {
	LoadCards(ctx context.Context, userIDs []string) (map[string][]profiles.Photo, map[string][]profiles.Interest, error)
}

type Service struct {
	repo     Repository
	cards    CardLoader
	notifier Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, notifier Notifier, pub realtime.Publisher) *Service {
	if pub == nil {
		pub = realtime.NopPublisher{}
	}
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

func clampBatch(n int) int {
	if n <= 0 {
		return DefaultBatch
	}
	if n > MaxBatch {
		return MaxBatch
	}
	return n
}

// Discover returns the next batch of profiles for the user.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	cands, err := s.repo.Candidates(ctx, userID, clampBatch(limit))
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(cands))
	if len(cands) == 0 {
		return out, nil
	}
	ids := make([]string, len(cands))
	for i, c := range cands {
		ids[i] = c.ID
	}
	photos, interests, err := s.cards.LoadCards(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range cands {
		p := profiles.PublicProfile{
			ID: c.ID, FirstName: c.FirstName, Age: c.Age, Gender: c.Gender, Bio: c.Bio, City: c.City,
			Photos: photos[c.ID], Interests: interests[c.ID],
		}
		if p.Photos == nil {
			p.Photos = []profiles.Photo{}
		}
		if p.Interests == nil {
			p.Interests = []profiles.Interest{}
		}
		if c.ShowDistance {
			d := profiles.DistanceBucket(c.DistanceKm)
			p.DistanceKm = &d
		}
		out = append(out, p)
	}
	return out, nil
}

// Swipe applies a like or pass and, on a new mutual like, notifies both people.
func (s *Service) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, ErrInvalidAction
	}
	res, err := s.repo.Swipe(ctx, actorID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Matched: res.Matched, Duplicate: res.Duplicate, MatchID: res.MatchID, ConversationID: res.ConversationID}
	if res.Matched && !res.Duplicate {
		s.announceMatch(ctx, actorID, targetID, res)
	}
	return resp, nil
}

func (s *Service) announceMatch(ctx context.Context, actorID, targetID string, res SwipeResult) {
	// Best effort: the match is already committed, a failed notification must not undo it.
	_ = s.notifier.Notify(ctx, targetID, notifications.KindMatch, "C'est un match !",
		"Toi et "+res.ActorName+" vous plaisez mutuellement. Dites-vous bonjour.",
		map[string]string{"matchId": res.MatchID, "conversationId": res.ConversationID, "userId": actorID})
	_ = s.notifier.Notify(ctx, actorID, notifications.KindMatch, "C'est un match !",
		"Toi et "+res.TargetName+" vous plaisez mutuellement. Dites-vous bonjour.",
		map[string]string{"matchId": res.MatchID, "conversationId": res.ConversationID, "userId": targetID})
	s.pub.Publish(targetID, realtime.Event{Type: "match.new", Data: map[string]string{
		"matchId": res.MatchID, "conversationId": res.ConversationID, "userId": actorID, "firstName": res.ActorName}})
}

func (s *Service) ResetPasses(ctx context.Context, userID string) (int64, error) {
	return s.repo.ResetPasses(ctx, userID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
