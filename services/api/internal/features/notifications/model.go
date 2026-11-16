package notifications

import "time"

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	ActorID   *string    `json:"actorId"`
	MatchID   *string    `json:"matchId"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	IsRead    bool       `json:"isRead"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	HasMore       bool           `json:"hasMore"`
}
