package matches

import (
	"context"
	"errors"
	"log"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/cursor"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotFound      = errors.New("conversation not found")
	ErrEmptyMessage  = errors.New("message cannot be empty")
	ErrMessageLong   = errors.New("message is too long (2000 characters max)")
	ErrInvalidCursor = errors.New("invalid cursor")
)

// Notifier is implemented by the notifications module.
type Notifier interface {
	NotifyMatch(ctx context.Context, userID, actorID, matchID, actorName string)
	NotifyMessage(ctx context.Context, userID, actorID, matchID, actorName, preview string)
	MarkMatchRead(ctx context.Context, userID, matchID string) error
}

type Service struct {
	repo     Repository
	profiles *profiles.Service
	notifier Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, profilesSvc *profiles.Service, notifier Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profilesSvc, notifier: notifier, pub: pub}
}

type Page struct {
	Matches    []MatchResponse
	NextCursor *string
}

func (s *Service) List(ctx context.Context, userID, rawCursor string, limit int) (Page, error) {
	after, err := cursor.Decode(rawCursor)
	if err != nil {
		return Page{}, ErrInvalidCursor
	}
	rows, err := s.repo.List(ctx, userID, after, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := cursor.Encode(last.LastActivityAt, last.ID)
		page.NextCursor = &next
	}
	page.Matches, err = s.hydrate(ctx, userID, rows)
	return page, err
}

func (s *Service) Get(ctx context.Context, userID, matchID string) (MatchResponse, error) {
	row, err := s.repo.Get(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return MatchResponse{}, ErrNotFound
		}
		return MatchResponse{}, err
	}
	out, err := s.hydrate(ctx, userID, []MatchSummary{row})
	if err != nil {
		return MatchResponse{}, err
	}
	if len(out) == 0 {
		return MatchResponse{}, ErrNotFound
	}
	return out[0], nil
}

// hydrate attaches the other person's public card (one query for the page).
func (s *Service) hydrate(ctx context.Context, userID string, rows []MatchSummary) ([]MatchResponse, error) {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]MatchResponse, 0, len(rows))
	for _, r := range rows {
		card, ok := byID[r.OtherUserID]
		if !ok {
			continue
		}
		item := MatchResponse{
			ID: r.ID, CreatedAt: r.CreatedAt, LastActivityAt: r.LastActivityAt,
			Profile: s.profiles.ToCardResponse(card), UnreadCount: r.UnreadCount,
		}
		if r.LastMessage != nil {
			m := ToMessageResponse(*r.LastMessage)
			item.LastMessage = &m
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, matchID, rawCursor string, limit int) ([]Message, *string, error) {
	if _, err := s.participant(ctx, userID, matchID); err != nil {
		return nil, nil, err
	}
	before, err := cursor.Decode(rawCursor)
	if err != nil {
		return nil, nil, ErrInvalidCursor
	}
	msgs, err := s.repo.ListMessages(ctx, matchID, before, limit+1)
	if err != nil {
		return nil, nil, err
	}
	var next *string
	if len(msgs) > limit {
		msgs = msgs[:limit]
		last := msgs[limit-1]
		c := cursor.Encode(last.CreatedAt, last.ID)
		next = &c
	}
	return msgs, next, nil
}

// NormalizeMessage trims and validates a message body.
func NormalizeMessage(body string) (string, error) {
	body = strings.ReplaceAll(body, "\x00", "")
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", ErrMessageLong
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, senderID, matchID, body string) (Message, error) {
	normalized, err := NormalizeMessage(body)
	if err != nil {
		return Message{}, err
	}
	otherID, err := s.participant(ctx, senderID, matchID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, matchID, senderID, normalized)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: ToMessageResponse(msg)}
	s.pub.Publish(otherID, event)
	s.pub.Publish(senderID, event)

	name, err := s.repo.FirstName(ctx, senderID)
	if err != nil {
		log.Printf(`{"event":"message_notification_name_failed","error":%q}`, err.Error())
		name = "Quelqu'un"
	}
	s.notifier.NotifyMessage(ctx, otherID, senderID, matchID, name, normalized)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, readerID, matchID string) error {
	otherID, err := s.participant(ctx, readerID, matchID)
	if err != nil {
		return err
	}
	n, readAt, err := s.repo.MarkRead(ctx, matchID, readerID)
	if err != nil {
		return err
	}
	if err := s.notifier.MarkMatchRead(ctx, readerID, matchID); err != nil {
		return err
	}
	if n > 0 {
		s.pub.Publish(otherID, realtime.Event{Type: "message.read", Data: map[string]any{
			"matchId": matchID, "readerId": readerID, "readAt": readAt,
		}})
	}
	return nil
}

// Unmatch deletes the match and its conversation for both people. The swipes
// are kept so neither profile is shown again in discovery.
func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	otherID, err := s.repo.Delete(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.PublishRemoved(matchID, userID, otherID)
	return nil
}

func (s *Service) PublishRemoved(matchID string, userIDs ...string) {
	for _, id := range userIDs {
		s.pub.Publish(id, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
}

// MatchCreated is called by discovery after a mutual like.
func (s *Service) MatchCreated(ctx context.Context, matchID, userA, userB string) {
	for _, pair := range [][2]string{{userA, userB}, {userB, userA}} {
		viewer, other := pair[0], pair[1]
		match, err := s.Get(ctx, viewer, matchID)
		if err != nil {
			log.Printf(`{"event":"match_event_failed","error":%q}`, err.Error())
			continue
		}
		s.pub.Publish(viewer, realtime.Event{Type: "match.new", Data: match})
		s.notifier.NotifyMatch(ctx, viewer, other, matchID, match.Profile.FirstName)
	}
}

func (s *Service) participant(ctx context.Context, userID, matchID string) (string, error) {
	other, err := s.repo.OtherParticipant(ctx, userID, matchID)
	if errors.Is(err, ErrRepoNotFound) {
		return "", ErrNotFound
	}
	return other, err
}
