package matching

import (
	"context"
	"strings"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

// CardLoader builds public profile cards (implemented by profiles.Service).
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
	CardResponse(card profiles.Card) profiles.CardResponse
	TouchActivity(ctx context.Context, userID string)
}

// Notifier stores and pushes in-app notifications.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
}

type Service struct {
	repo        Repository
	cards       CardLoader
	notifier    Notifier
	publisher   realtime.Publisher
	ranker      Ranker
	includeDemo bool
	now         func() time.Time
}

type Options struct {
	// IncludeDemo exposes seeded demo profiles in discovery (never in production).
	IncludeDemo bool
	Ranker      Ranker
}

func NewService(repo Repository, cards CardLoader, notifier Notifier, publisher realtime.Publisher, opts Options) *Service {
	if publisher == nil {
		publisher = realtime.NopPublisher{}
	}
	ranker := opts.Ranker
	if ranker == nil {
		ranker = ScoreRanker{}
	}
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher, ranker: ranker, includeDemo: opts.IncludeDemo, now: time.Now}
}

// Discover returns the next profiles to review.
func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]profiles.CardResponse, error) {
	ready, err := s.repo.ViewerReady(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrProfileIncomplete
	}
	s.cards.TouchActivity(ctx, viewerID)

	pool, err := s.repo.CandidatePool(ctx, viewerID, s.includeDemo, CandidatePoolSize)
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(viewerID, pool, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	cards, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.CardResponse, 0, len(cards))
	for _, card := range cards {
		out = append(out, s.cards.CardResponse(card))
	}
	return out, nil
}

// Swipe validates and records a decision; on a new match both people get
// a notification and a real-time event.
func (s *Service) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResponse, error) {
	// Canonical lowercase ids: ordering, pair locks and equality checks all
	// depend on it.
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	viewerID = strings.ToLower(viewerID)
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, httpx.Invalid("invalid_action", "Action invalide (like ou pass).")
	}
	if !httpx.IsUUID(targetID) {
		return SwipeResponse{}, httpx.Invalid("invalid_id", "Identifiant invalide.")
	}
	if targetID == viewerID {
		return SwipeResponse{}, httpx.Invalid("invalid_target", "Tu ne peux pas te liker toi-même.")
	}
	ready, err := s.repo.ViewerReady(ctx, viewerID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ready {
		return SwipeResponse{}, ErrProfileIncomplete
	}

	result, err := s.repo.Swipe(ctx, viewerID, targetID, action, s.includeDemo)
	if err != nil {
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Action: result.Action, AlreadySwiped: result.AlreadySwiped, Matched: result.Matched}
	if !result.Matched {
		return resp, nil
	}

	summary, err := s.summary(ctx, viewerID, result.MatchID)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Match = &summary

	if !result.AlreadySwiped {
		s.announceMatch(ctx, viewerID, targetID, result.MatchID, summary)
	}
	return resp, nil
}

func (s *Service) announceMatch(ctx context.Context, viewerID, targetID, matchID string, viewerSummary MatchSummaryResponse) {
	s.publisher.Publish(viewerID, "match.new", viewerSummary)
	if other, err := s.summary(ctx, targetID, matchID); err == nil {
		s.publisher.Publish(targetID, "match.new", other)
		s.notifier.Notify(ctx, targetID, notifications.KindMatch, "Nouveau match !",
			"Toi et "+other.User.FirstName+" vous plaisez mutuellement. Lance la conversation.",
			map[string]string{"matchId": matchID, "userId": viewerID})
	}
	s.notifier.Notify(ctx, viewerID, notifications.KindMatch, "Nouveau match !",
		"Toi et "+viewerSummary.User.FirstName+" vous plaisez mutuellement. Lance la conversation.",
		map[string]string{"matchId": matchID, "userId": targetID})
}

func (s *Service) summary(ctx context.Context, userID, matchID string) (MatchSummaryResponse, error) {
	m, err := s.repo.GetMatch(ctx, userID, matchID)
	if err != nil {
		return MatchSummaryResponse{}, err
	}
	list, err := s.toResponses(ctx, userID, []MatchSummary{m})
	if err != nil {
		return MatchSummaryResponse{}, err
	}
	if len(list) == 0 {
		return MatchSummaryResponse{}, ErrMatchNotFound
	}
	return list[0], nil
}

func (s *Service) toResponses(ctx context.Context, viewerID string, list []MatchSummary) ([]MatchSummaryResponse, error) {
	ids := make([]string, 0, len(list))
	for _, m := range list {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	byUser := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byUser[c.UserID] = c
	}
	out := make([]MatchSummaryResponse, 0, len(list))
	for _, m := range list {
		card, ok := byUser[m.OtherUserID]
		if !ok {
			continue
		}
		item := MatchSummaryResponse{
			ID:          m.ID,
			MatchedAt:   m.MatchedAt,
			User:        s.cards.CardResponse(card),
			UnreadCount: m.UnreadCount,
		}
		if m.LastMessage != nil {
			item.LastMessage = &LastMessageResponse{
				ID:           m.LastMessage.ID,
				SenderUserID: m.LastMessage.SenderUserID,
				Content:      m.LastMessage.Content,
				CreatedAt:    m.LastMessage.CreatedAt,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// ListMatches returns the inbox page and the cursor of the next one.
func (s *Service) ListMatches(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) ([]MatchSummaryResponse, string, error) {
	var beforeAt *time.Time
	beforeID := ""
	if cursor != nil {
		beforeAt = &cursor.At
		beforeID = cursor.ID
	}
	list, err := s.repo.ListMatches(ctx, userID, beforeAt, beforeID, limit)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(list) == limit {
		last := list[len(list)-1]
		next = httpx.EncodeCursor(last.ActivityAt, last.ID)
	}
	out, err := s.toResponses(ctx, userID, list)
	return out, next, err
}

func (s *Service) GetMatch(ctx context.Context, userID, matchID string) (MatchSummaryResponse, error) {
	return s.summary(ctx, userID, matchID)
}

// Unmatch ends the match for both members.
func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.DeleteMatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	payload := map[string]string{"matchId": matchID}
	s.publisher.Publish(other, "match.removed", payload)
	s.publisher.Publish(userID, "match.removed", payload)
	return nil
}
