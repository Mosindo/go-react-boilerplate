package notifications

import "time"

const (
	TypeNewMatch   = "new_match"
	TypeNewMessage = "new_message"
)

type Notification struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Body      string
	Data      map[string]string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

// New describes a notification to create. With Dedupe set, nothing is created
// while the user still has an unread notification of the same type for the
// same data["matchId"] (prevents one push per chat message).
type New struct {
	UserID string
	Type   string
	Title  string
	Body   string
	Data   map[string]string
	Dedupe bool
}

type NotificationResponse struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data"`
	IsRead    bool              `json:"isRead"`
	CreatedAt time.Time         `json:"createdAt"`
	ReadAt    *time.Time        `json:"readAt,omitempty"`
}

type NotificationsResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int                    `json:"unreadCount"`
}

func ToResponse(n Notification) NotificationResponse {
	data := n.Data
	if data == nil {
		data = map[string]string{}
	}
	return NotificationResponse{
		ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body, Data: data,
		IsRead: n.IsRead, CreatedAt: n.CreatedAt, ReadAt: n.ReadAt,
	}
}
