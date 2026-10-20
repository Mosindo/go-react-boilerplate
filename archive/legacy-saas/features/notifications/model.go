package notifications

import "time"

type Notification struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	IsRead    bool       `json:"isRead"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
}

type NotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
}

type CreateNotificationRequest struct {
	Type  string `json:"type" binding:"required,max=64"`
	Title string `json:"title" binding:"required,max=160"`
	Body  string `json:"body" binding:"required,max=1000"`
}
