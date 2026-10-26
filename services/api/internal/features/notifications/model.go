package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Actor struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	MatchID   *string    `json:"matchId"`
	Actor     *Actor     `json:"actor"`
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    string         `json:"nextCursor,omitempty"`
}

// New describes a notification created by another feature inside its own transaction.
type New struct {
	UserID  string
	Type    string
	ActorID string
	MatchID string
}
