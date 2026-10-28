package safety

import "time"

type BlockRequest struct {
	UserID string `json:"userId" binding:"required,uuid"`
}

type Block struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReportRequest struct {
	UserID         string `json:"userId" binding:"required,uuid"`
	Reason         string `json:"reason" binding:"required,oneof=spam fake_profile harassment inappropriate_content underage other"`
	Details        string `json:"details" binding:"max=4000"`
	ConversationID string `json:"conversationId" binding:"omitempty,uuid"`
}
