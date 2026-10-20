package chat

import "time"

type ChatMessagePreview struct {
	Content   string
	CreatedAt time.Time
}

type ChatSummary struct {
	UserID        string
	UserEmail     string
	UserCreatedAt time.Time
	LastMessage   *ChatMessagePreview
}

type ChatMessage struct {
	ID              string
	SenderUserID    string
	RecipientUserID string
	Content         string
	CreatedAt       time.Time
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type ChatMessagePreviewResponse struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ChatSummaryResponse struct {
	User        UserResponse                `json:"user"`
	LastMessage *ChatMessagePreviewResponse `json:"lastMessage,omitempty"`
}

type ChatsResponse struct {
	Chats []ChatSummaryResponse `json:"chats"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=2000"`
}

type ChatMessageResponse struct {
	ID              string    `json:"id"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type ChatMessagesResponse struct {
	Messages []ChatMessageResponse `json:"messages"`
}
