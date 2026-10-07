package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/media"
)

var (
	errIncomplete   = platformerrors.New(403, platformerrors.CodeProfileIncomplete, "complete your profile (with at least one photo) first")
	errAlreadySwipe = platformerrors.New(409, platformerrors.CodeAlreadySwiped, "you already reacted to this profile")
	errNotFound     = platformerrors.NotFound("profile not found")
)

// ProfileReader is satisfied by *profiles.Service.
type ProfileReader interface {
	Photos(ctx context.Context, userIDs []string) (map[string][]media.Photo, error)
	InterestsByUsers(ctx context.Context, userIDs []string) (map[string][]profiles.Interest, error)
	Summaries(ctx context.Context, userIDs []string) (map[string]profiles.UserSummary, error)
}

// Notifier persists a notification and pushes it in real time.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

// Publisher pushes realtime events to connected clients.
type Publisher interface {
	Publish(userID, eventType string, data any)
}

type Service struct {
	repo     *Repository
	profiles ProfileReader
	notifier Notifier
	events   Publisher
}

func NewService(repo *Repository, reader ProfileReader, notifier Notifier, events Publisher) *Service {
	return &Service{repo: repo, profiles: reader, notifier: notifier, events: events}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	ok, err := s.repo.HasCompleteProfile(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errIncomplete
	}
	return nil
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]PublicProfile, error) {
	if err := s.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	cands, err := s.repo.NextCandidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, userID, cands)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	if err := s.requireComplete(ctx, viewerID); err != nil {
		return PublicProfile{}, err
	}
	cand, err := s.repo.MatchedTarget(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if cand == nil {
		if cand, err = s.repo.EligibleTarget(ctx, viewerID, targetID); err != nil {
			return PublicProfile{}, err
		}
	}
	if cand == nil {
		return PublicProfile{}, errNotFound
	}
	list, err := s.hydrate(ctx, viewerID, []Candidate{*cand})
	if err != nil {
		return PublicProfile{}, err
	}
	return list[0], nil
}

// hydrate batch-loads photos and interests for the whole page (constant number of queries).
func (s *Service) hydrate(ctx context.Context, viewerID string, cands []Candidate) ([]PublicProfile, error) {
	if len(cands) == 0 {
		return []PublicProfile{}, nil
	}
	ids := make([]string, 0, len(cands)+1)
	for _, c := range cands {
		ids = append(ids, c.UserID)
	}
	photos, err := s.profiles.Photos(ctx, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.profiles.InterestsByUsers(ctx, append(ids, viewerID))
	if err != nil {
		return nil, err
	}
	mine := make(map[string]struct{}, len(interests[viewerID]))
	for _, in := range interests[viewerID] {
		mine[in.Slug] = struct{}{}
	}

	out := make([]PublicProfile, 0, len(cands))
	for _, c := range cands {
		p := PublicProfile{
			UserID:          c.UserID,
			FirstName:       c.FirstName,
			Age:             c.Age,
			Gender:          c.Gender,
			Bio:             c.Bio,
			City:            c.City,
			Interests:       interests[c.UserID],
			SharedInterests: []string{},
			Photos:          photos[c.UserID],
		}
		if p.Interests == nil {
			p.Interests = []profiles.Interest{}
		}
		if p.Photos == nil {
			p.Photos = []media.Photo{}
		}
		for _, in := range p.Interests {
			if _, ok := mine[in.Slug]; ok {
				p.SharedInterests = append(p.SharedInterests, in.Slug)
			}
		}
		if c.ShowDistance && c.DistanceKm != nil {
			d := approximateDistance(*c.DistanceKm)
			p.DistanceKm = &d
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, viewerID string, req SwipeRequest) (SwipeResponse, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "like" && action != "pass" {
		return SwipeResponse{}, platformerrors.Validation("action must be like or pass")
	}
	targetID := strings.ToLower(strings.TrimSpace(req.TargetUserID))
	if targetID == viewerID {
		return SwipeResponse{}, platformerrors.Validation("you cannot react to yourself")
	}
	if err := s.requireComplete(ctx, viewerID); err != nil {
		return SwipeResponse{}, err
	}
	eligible, err := s.repo.EligibleTarget(ctx, viewerID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if eligible == nil {
		return SwipeResponse{}, errNotFound
	}

	row, err := s.repo.RecordSwipe(ctx, viewerID, targetID, action)
	switch {
	case errors.Is(err, ErrRepoAlreadySwiped):
		return SwipeResponse{}, errAlreadySwipe
	case errors.Is(err, ErrRepoNotEligible):
		return SwipeResponse{}, errNotFound
	case err != nil:
		return SwipeResponse{}, err
	}
	if row == nil {
		return SwipeResponse{Matched: false}, nil
	}

	summaries, err := s.profiles.Summaries(ctx, []string{viewerID, targetID})
	if err != nil {
		return SwipeResponse{}, err
	}
	viewerMatch := Match{ID: row.ID, User: summaries[targetID], ConversationID: row.ConversationID, CreatedAt: row.CreatedAt}
	targetMatch := Match{ID: row.ID, User: summaries[viewerID], ConversationID: row.ConversationID, CreatedAt: row.CreatedAt}

	// Side effects are best-effort: the match itself is already durable.
	s.events.Publish(targetID, "match", targetMatch)
	s.events.Publish(viewerID, "match", viewerMatch)
	s.notifyMatch(ctx, viewerID, summaries[targetID], row)
	s.notifyMatch(ctx, targetID, summaries[viewerID], row)

	return SwipeResponse{Matched: true, Match: &viewerMatch}, nil
}

func (s *Service) notifyMatch(ctx context.Context, userID string, other profiles.UserSummary, row *MatchRow) {
	_ = s.notifier.Notify(ctx, userID, "match", "It's a match!",
		fmt.Sprintf("You and %s liked each other. Say hello!", other.FirstName),
		map[string]string{"matchId": row.ID, "conversationId": row.ConversationID, "userId": other.UserID})
}

func (s *Service) Matches(ctx context.Context, userID string, limit, offset int) ([]Match, error) {
	rows, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	summaries, err := s.profiles.Summaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Match, 0, len(rows))
	for _, r := range rows {
		out = append(out, Match{ID: r.ID, User: summaries[r.OtherUserID], ConversationID: r.ConversationID, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, conversationID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return platformerrors.NotFound("match not found")
		}
		return err
	}
	s.events.Publish(other, "unmatch", map[string]string{"matchId": matchID, "conversationId": conversationID})
	return nil
}
