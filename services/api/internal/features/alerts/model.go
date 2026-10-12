// Package alerts exposes in-app notifications (new matches, new messages).
package alerts

import "time"

type Actor struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	ThumbURL  string `json:"thumbUrl"`
}

type Notification struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Actor     *Actor     `json:"actor"`
	MatchID   *string    `json:"matchId"`
	ReadAt    *time.Time `json:"readAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Page struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
	NextBefore  string         `json:"nextBefore"`
}

type Stored struct {
	ID        string
	Kind      string
	ActorID   *string
	MatchID   *string
	ReadAt    *time.Time
	CreatedAt time.Time
}
