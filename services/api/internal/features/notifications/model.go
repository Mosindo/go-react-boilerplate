package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Actor struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl,omitempty"`
}

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Actor     *Actor     `json:"actor,omitempty"`
	RefID     *string    `json:"refId,omitempty"` // match id or conversation id
	ReadAt    *time.Time `json:"readAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextBefore    *time.Time     `json:"nextBefore,omitempty"`
}

type stored struct {
	ID        string
	Type      string
	ActorID   *string
	RefID     *string
	ReadAt    *time.Time
	CreatedAt time.Time
}
