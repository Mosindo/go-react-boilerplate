package notifications

import "time"

type Notification struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
}

type ListResponse struct {
	Items       []Notification `json:"items"`
	UnreadCount int            `json:"unreadCount"`
}
