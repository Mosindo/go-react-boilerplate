package notifications

import "time"

const (
	TypeMatch   = "match"
	TypeMessage = "message"
)

type Notification struct {
	ID        string
	Type      string
	ActorID   *string
	MatchID   *string
	Body      string
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationResponse struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	ActorID   *string    `json:"actorId"`
	MatchID   *string    `json:"matchId"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type ListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int                    `json:"unreadCount"`
	NextCursor    *string                `json:"nextCursor"`
}

func ToResponse(n Notification) NotificationResponse {
	return NotificationResponse{
		ID: n.ID, Type: n.Type, ActorID: n.ActorID, MatchID: n.MatchID,
		Body: n.Body, CreatedAt: n.CreatedAt, ReadAt: n.ReadAt,
	}
}
