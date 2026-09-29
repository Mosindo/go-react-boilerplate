package matching

import (
	"context"
	"log"
	"strings"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/urlsign"
)

type Service struct {
	repo      *Repository
	signer    *urlsign.Signer
	notifier  notify.Notifier
	publisher realtime.Publisher
}

func NewService(repo *Repository, signer *urlsign.Signer, n notify.Notifier, p realtime.Publisher) *Service {
	if n == nil {
		n = notify.Nop{}
	}
	if p == nil {
		p = realtime.NopPublisher{}
	}
	return &Service{repo: repo, signer: signer, notifier: n, publisher: p}
}

func (s *Service) Swipe(ctx context.Context, me string, req swipeRequest) (SwipeResponse, error) {
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	action := strings.TrimSpace(req.Action)
	if !isUUID(target) || (action != ActionLike && action != ActionPass) {
		return SwipeResponse{}, ErrBadRequest
	}
	if target == me {
		return SwipeResponse{}, ErrSelfSwipe
	}
	out, err := s.repo.Swipe(ctx, me, target, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !out.Matched {
		return SwipeResponse{}, nil
	}

	parties, err := s.repo.Parties(ctx, []string{me, target})
	if err != nil {
		// The match is committed; degrade gracefully rather than failing the request.
		log.Printf("matching: load parties: %v", err)
		return SwipeResponse{Matched: true}, nil
	}
	mine := s.summary(out, parties[target])
	theirs := s.summary(out, parties[me])

	for _, side := range []struct {
		userID, otherID string
		conv            ConversationSummary
	}{{me, target, mine}, {target, me, theirs}} {
		if err := s.notifier.Notify(ctx, notify.New{
			UserID: side.userID,
			Type:   notify.TypeMatch,
			Title:  "New match",
			Body:   "You have a new match",
			Data:   map[string]any{"conversationId": out.ConversationID, "userId": side.otherID},
		}); err != nil {
			log.Printf("matching: notify: %v", err)
		}
		s.publisher.Publish(side.userID, realtime.Event{Type: "match.new", Data: map[string]any{"conversation": side.conv}})
	}
	return SwipeResponse{Matched: true, Conversation: &mine}, nil
}

// summary builds the conversation summary as seen by the user who is NOT other.
func (s *Service) summary(out swipeOutcome, other party) ConversationSummary {
	u := SummaryUser{UserID: other.UserID, FirstName: other.FirstName, Age: other.Age}
	if other.PhotoID != nil {
		u.Photo = &Photo{ID: *other.PhotoID, URL: s.signer.PhotoURL(*other.PhotoID), Position: 0}
	}
	return ConversationSummary{
		ID: out.ConversationID, MatchID: out.MatchID, MatchedAt: out.MatchedAt,
		User: u, LastMessage: nil, UnreadCount: 0, UpdatedAt: out.MatchedAt,
	}
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	matchID = strings.ToLower(strings.TrimSpace(matchID))
	if !isUUID(matchID) {
		return ErrMatchNotFound
	}
	rm, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	ev := realtime.Event{Type: "match.removed", Data: map[string]any{"conversationId": rm.ConversationID}}
	s.publisher.Publish(rm.UserA, ev)
	s.publisher.Publish(rm.UserB, ev)
	return nil
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, ch := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if ch != '-' {
				return false
			}
		case (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'):
		default:
			return false
		}
	}
	return true
}
