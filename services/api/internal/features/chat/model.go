package chat

import "time"

type Message struct {
	ID              string    `json:"id"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type UserSummary struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
}

type ChatSummary struct {
	User        UserSummary `json:"user"`
	LastMessage Message     `json:"lastMessage"`
	UnreadCount int         `json:"unreadCount"`
}

type ChatsResponse struct {
	Chats []ChatSummary `json:"chats"`
}

type MessagesResponse struct {
	Messages        []Message  `json:"messages"`
	HasMore         bool       `json:"hasMore"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type ReadResponse struct {
	ReadAt time.Time `json:"readAt"`
}

type storedChat struct {
	UserID      string
	FirstName   string
	Birthdate   time.Time
	PhotoID     *string
	LastMessage Message
	UnreadCount int
}

type conversationAccess struct {
	ConversationID  string
	SenderFirstName string
}
