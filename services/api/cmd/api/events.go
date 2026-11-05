package main

import (
	"context"
	"log"
	"time"

	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/discovery"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/realtime"
)

// eventBridge is the composition point where domain events fan out to
// in-app notifications and live WebSocket pushes. Features only depend on
// small Events interfaces, never on each other.
type eventBridge struct {
	hub   *realtime.Hub
	notif *notifications.Service
}

var (
	_ discovery.Events = (*eventBridge)(nil)
	_ chat.Events      = (*eventBridge)(nil)
)

func (b *eventBridge) async(fn func(ctx context.Context)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		fn(ctx)
	}()
}

func (b *eventBridge) pushNotification(userID string, n notifications.Notification) {
	b.hub.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
}

func (b *eventBridge) MatchCreated(info discovery.MatchInfo) {
	b.async(func(ctx context.Context) {
		for _, side := range []struct{ user, other, otherName, otherID string }{
			{info.UserA, info.UserB, info.NameB, info.UserB},
			{info.UserB, info.UserA, info.NameA, info.UserA},
		} {
			n, err := b.notif.NotifyMatch(ctx, side.user, side.otherName, info.ConversationID)
			if err != nil {
				log.Printf(`{"event":"notify_match_failed","error":%q}`, err.Error())
			} else {
				b.pushNotification(side.user, n)
			}
			b.hub.Publish(side.user, realtime.Event{Type: "match.new", Data: map[string]string{
				"matchId": info.MatchID, "conversationId": info.ConversationID,
				"peerId": side.otherID, "peerName": side.otherName,
			}})
		}
	})
}

func (b *eventBridge) MatchEnded(matchID, conversationID, byUserID, otherUserID string) {
	ev := realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": matchID, "conversationId": conversationID}}
	b.hub.Publish(otherUserID, ev)
	b.hub.Publish(byUserID, ev)
}

func (b *eventBridge) MessageCreated(msg chat.Message, senderName, recipientID string) {
	b.hub.Publish(msg.SenderID, realtime.Event{Type: "message.new", Data: msg})
	b.hub.Publish(recipientID, realtime.Event{Type: "message.new", Data: msg})
	b.async(func(ctx context.Context) {
		n, err := b.notif.NotifyMessage(ctx, recipientID, senderName, msg.ConversationID, msg.Body)
		if err != nil {
			log.Printf(`{"event":"notify_message_failed","error":%q}`, err.Error())
			return
		}
		b.pushNotification(recipientID, n)
	})
}

func (b *eventBridge) MessagesRead(conversationID, readerID, otherID string, upTo int64) {
	data := map[string]any{"conversationId": conversationID, "readerId": readerID, "upToMessageId": upTo}
	b.hub.Publish(otherID, realtime.Event{Type: "message.read", Data: data})
	b.hub.Publish(readerID, realtime.Event{Type: "message.read", Data: data})
	b.async(func(ctx context.Context) {
		if err := b.notif.MarkConversationRead(ctx, readerID, conversationID); err != nil {
			log.Printf(`{"event":"notify_read_failed","error":%q}`, err.Error())
		}
	})
}
