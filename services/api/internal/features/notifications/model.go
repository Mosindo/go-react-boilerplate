package notifications

import (
	"time"

	"example.com/api/internal/platform/realtime"
)

const (
	KindMatch   = "match"
	KindMessage = "message"
)

type Notification struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	RefID         string     `json:"refId"`
	ActorID       *string    `json:"actorId"`
	ActorName     string     `json:"actorName"`
	ActorThumbURL *string    `json:"actorThumbUrl"`
	IsRead        bool       `json:"isRead"`
	CreatedAt     time.Time  `json:"createdAt"`
	ReadAt        *time.Time `json:"readAt"`
}

// Publisher pushes realtime events to connected clients.
type Publisher interface {
	Publish(userID string, ev realtime.Event)
}
