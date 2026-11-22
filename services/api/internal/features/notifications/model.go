package notifications

import "time"

type Notification struct {
	ID        string            `json:"id"`
	UserID    string            `json:"userId"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    *time.Time        `json:"readAt,omitempty"`
}

type NotificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
}

type CreateNotificationRequest struct {
	Type  string `json:"type" binding:"required,max=64"`
	Title string `json:"title" binding:"required,max=160"`
	Body  string `json:"body" binding:"required,max=1000"`
}

type ReadAllResponse struct {
	Updated int `json:"updated"`
}
