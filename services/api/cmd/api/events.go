package main

import (
	"context"
	"log"
	"strings"
	"unicode/utf8"

	chatfeature "example.com/api/internal/features/chat"
	notificationsfeature "example.com/api/internal/features/notifications"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// eventBridge is the one place that knows both the domain features and the delivery
// channels: it turns "a match formed" / "a message was sent" into persisted
// notifications and live WebSocket events. Failures here are logged, never returned:
// a notification problem must not undo a like or a sent message.
type eventBridge struct {
	notifications *notificationsfeature.Service
	profiles      *profilesfeature.Service
	hub           *realtime.Hub
}

func newEventBridge(n *notificationsfeature.Service, p *profilesfeature.Service, hub *realtime.Hub) *eventBridge {
	return &eventBridge{notifications: n, profiles: p, hub: hub}
}

func (e *eventBridge) firstName(ctx context.Context, viewerID, userID string) string {
	cards, err := e.profiles.Cards(ctx, viewerID, []string{userID})
	if err != nil || len(cards) == 0 {
		return "Quelqu'un"
	}
	return cards[0].FirstName
}

func (e *eventBridge) notify(ctx context.Context, n notificationsfeature.NewNotification) {
	created, err := e.notifications.Create(ctx, n)
	if err != nil {
		log.Printf(`{"event":"notification_failed","type":%q,"error":%q}`, n.Type, err.Error())
		return
	}
	e.hub.Publish(n.UserID, realtime.Event{Type: "notification.new", Data: created})
}

func (e *eventBridge) MatchCreated(ctx context.Context, matchID, userA, userB string) {
	for _, pair := range [][2]string{{userA, userB}, {userB, userA}} {
		recipient, other := pair[0], pair[1]
		name := e.firstName(ctx, recipient, other)
		e.hub.Publish(recipient, realtime.Event{Type: "match.new", Data: map[string]string{"matchId": matchID, "userId": other}})
		e.notify(ctx, notificationsfeature.NewNotification{
			UserID: recipient,
			Type:   notificationsfeature.TypeMatch,
			Title:  "C'est un match !",
			Body:   "Vous et " + name + " vous plaisez mutuellement. Dites bonjour !",
			Data:   map[string]any{"matchId": matchID, "userId": other},
		})
	}
}

func (e *eventBridge) MessageCreated(ctx context.Context, m chatfeature.Message, recipientID string) {
	// The sender's other devices get the message too; only the recipient is notified.
	e.hub.Publish(m.SenderID, realtime.Event{Type: "message.new", Data: m})
	e.hub.Publish(recipientID, realtime.Event{Type: "message.new", Data: m})

	// One unread notification per conversation: repeated messages update it in place.
	e.notify(ctx, notificationsfeature.NewNotification{
		UserID: recipientID,
		Type:   notificationsfeature.TypeMessage,
		Title:  "Nouveau message de " + e.firstName(ctx, recipientID, m.SenderID),
		Body:   preview(m.Body, 80),
		Data:   map[string]any{"matchId": m.MatchID, "userId": m.SenderID},
	})
}

func (e *eventBridge) MessagesRead(ctx context.Context, matchID, readerID, otherID string) {
	e.hub.Publish(otherID, realtime.Event{Type: "message.read", Data: map[string]string{"matchId": matchID, "readerId": readerID}})
	if err := e.notifications.MarkConversationRead(ctx, readerID, matchID); err != nil {
		log.Printf(`{"event":"notification_clear_failed","error":%q}`, err.Error())
	}
}

func preview(body string, max int) string {
	body = strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(body) <= max {
		return body
	}
	r := []rune(body)
	return string(r[:max-1]) + "…"
}
