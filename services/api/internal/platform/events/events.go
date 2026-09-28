// Package events defines the tiny seam between services and the realtime hub.
package events

// Event types pushed to clients over the WebSocket (see docs/API.md).
const (
	MessageNew      = "message.new"
	MessagesRead    = "messages.read"
	MatchNew        = "match.new"
	MatchRemoved    = "match.removed"
	NotificationNew = "notification.new"
)

// Publisher delivers an event to every live connection of a user. Delivery is
// best effort and must never block or fail the caller.
type Publisher interface {
	Publish(userID, eventType string, data any)
}

// Nop discards events (tests, seed tool).
type Nop struct{}

func (Nop) Publish(string, string, any) {}
