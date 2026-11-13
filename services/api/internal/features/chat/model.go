package chat

import "time"

const (
	maxMessageRunes      = 2000
	defaultMessagesLimit = 30
	maxMessagesLimit     = 100
	defaultChatsLimit    = 30
	maxChatsLimit        = 100
)

type Message struct {
	ID              string    `json:"id"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Peer struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	Age       int    `json:"age"`
	PhotoURL  string `json:"photoUrl"`
	IsDemo    bool   `json:"isDemo"`
}

type Conversation struct {
	MatchID     string    `json:"matchId"`
	MatchedAt   time.Time `json:"matchedAt"`
	User        Peer      `json:"user"`
	LastMessage *Message  `json:"lastMessage"`
	UnreadCount int       `json:"unreadCount"`
}

type ChatsResponse struct {
	Chats []Conversation `json:"chats"`
}

type MessagesPage struct {
	Messages       []Message  `json:"messages"`
	HasMore        bool       `json:"hasMore"`
	PeerLastReadAt *time.Time `json:"peerLastReadAt"`
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,max=8000"`
}

type storedConversation struct {
	MatchID     string
	MatchedAt   time.Time
	PeerID      string
	LastMessage *Message
	UnreadCount int
}
