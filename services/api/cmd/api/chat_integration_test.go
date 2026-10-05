package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type chatMsg struct {
	ID             string  `json:"id"`
	ConversationID string  `json:"conversationId"`
	SenderUserID   string  `json:"senderUserId"`
	Content        string  `json:"content"`
	ReadAt         *string `json:"readAt"`
}

type messagesPage struct {
	Messages   []chatMsg `json:"messages"`
	NextCursor string    `json:"nextCursor"`
}

type conversationList struct {
	Conversations []struct {
		ID          string   `json:"id"`
		User        card     `json:"user"`
		LastMessage *chatMsg `json:"lastMessage"`
		UnreadCount int      `json:"unreadCount"`
	} `json:"conversations"`
}

func TestChatSendReadAndUnread(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	m := env.match(a, b)
	path := "/conversations/" + m.ConversationID

	// A fresh match has no message: it lives in /matches, not in /conversations.
	var list conversationList
	env.do(http.MethodGet, "/conversations", a.Token, nil).expect(t, http.StatusOK).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatalf("empty conversation must not be listed: %+v", list)
	}

	var sent chatMsg
	env.do(http.MethodPost, path+"/messages", a.Token, map[string]string{"content": "  Salut, ça va ?  "}).expect(t, http.StatusCreated).json(t, &sent)
	if sent.Content != "Salut, ça va ?" || sent.SenderUserID != a.ID || sent.ReadAt != nil {
		t.Fatalf("unexpected sent message: %+v", sent)
	}
	env.do(http.MethodPost, path+"/messages", a.Token, map[string]string{"content": "Tu es là ?"}).expect(t, http.StatusCreated)

	env.do(http.MethodGet, "/conversations", b.Token, nil).json(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 2 || list.Conversations[0].LastMessage.Content != "Tu es là ?" || list.Conversations[0].User.FirstName != "Alice" {
		t.Fatalf("recipient should see 2 unread: %+v", list)
	}
	env.do(http.MethodGet, "/conversations", a.Token, nil).json(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatalf("sender has nothing unread: %+v", list)
	}

	var page messagesPage
	env.do(http.MethodGet, path+"/messages", b.Token, nil).expect(t, http.StatusOK).json(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Content != "Salut, ça va ?" {
		t.Fatalf("messages must be chronological: %+v", page)
	}
	if page.Messages[0].ReadAt != nil {
		t.Fatal("reading the list must not mark messages as read")
	}

	env.do(http.MethodPost, path+"/read", b.Token, nil).expect(t, http.StatusOK)
	env.do(http.MethodGet, "/conversations", b.Token, nil).json(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatalf("unread must be 0 after read: %+v", list)
	}
	env.do(http.MethodGet, path+"/messages", a.Token, nil).json(t, &page)
	if page.Messages[0].ReadAt == nil || page.Messages[1].ReadAt == nil {
		t.Fatalf("sender must see read receipts: %+v", page)
	}
	// Reading is for the recipient only: the sender marking read does not touch their own sent messages.
	env.do(http.MethodPost, path+"/messages", b.Token, map[string]string{"content": "Oui !"}).expect(t, http.StatusCreated)
	env.do(http.MethodPost, path+"/read", b.Token, nil).expect(t, http.StatusOK)
	env.do(http.MethodGet, path+"/messages", b.Token, nil).json(t, &page)
	if page.Messages[2].ReadAt != nil {
		t.Fatal("a user cannot mark their own message as read")
	}
}

func TestChatMessageValidation(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	path := "/conversations/" + env.match(a, b).ConversationID + "/messages"

	env.do(http.MethodPost, path, a.Token, map[string]string{"content": "   "}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, path, a.Token, map[string]string{"content": "\x00\x01\x02"}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, path, a.Token, map[string]string{"content": strings.Repeat("é", 2001)}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, path, a.Token, map[string]string{"content": strings.Repeat("é", 2000)}).expect(t, http.StatusCreated)
	env.do(http.MethodPost, path, a.Token, map[string]any{"content": 42}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, path, "", map[string]string{"content": "hi"}).expect(t, http.StatusUnauthorized)

	var sent chatMsg
	env.do(http.MethodPost, path, a.Token, map[string]string{"content": "<script>alert(1)</script>"}).expect(t, http.StatusCreated).json(t, &sent)
	if sent.Content != "<script>alert(1)</script>" {
		t.Fatalf("content is stored verbatim (clients render text, never HTML): %q", sent.Content)
	}
}

func TestChatIsPrivate(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	intruder := env.register("intruder")
	env.onboard(intruder, profileSpec{Name: "Ivan", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng})
	m := env.match(a, b)
	path := "/conversations/" + m.ConversationID

	env.do(http.MethodPost, path+"/messages", a.Token, map[string]string{"content": "secret"}).expect(t, http.StatusCreated)

	// Outsiders get a 404 on every conversation operation: reading, writing, marking read, clearing.
	env.do(http.MethodGet, path+"/messages", intruder.Token, nil).expect(t, http.StatusNotFound)
	env.do(http.MethodPost, path+"/messages", intruder.Token, map[string]string{"content": "hi"}).expect(t, http.StatusNotFound)
	env.do(http.MethodPost, path+"/read", intruder.Token, nil).expect(t, http.StatusNotFound)
	env.do(http.MethodDelete, path, intruder.Token, nil).expect(t, http.StatusNotFound)
	env.do(http.MethodGet, path+"/messages", "", nil).expect(t, http.StatusUnauthorized)
	env.do(http.MethodGet, "/conversations/not-a-uuid/messages", a.Token, nil).expect(t, http.StatusBadRequest)

	var list conversationList
	env.do(http.MethodGet, "/conversations", intruder.Token, nil).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("outsider must not list foreign conversations")
	}
	// Even after liking a participant, no conversation exists without a mutual match.
	env.swipe(intruder, a.ID, "like").expect(t, http.StatusOK)
	env.do(http.MethodGet, path+"/messages", intruder.Token, nil).expect(t, http.StatusNotFound)
}

func TestChatPagination(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	path := "/conversations/" + env.match(a, b).ConversationID + "/messages"
	for i := 1; i <= 25; i++ {
		env.do(http.MethodPost, path, a.Token, map[string]string{"content": fmt.Sprintf("m%02d", i)}).expect(t, http.StatusCreated)
	}

	var page messagesPage
	env.do(http.MethodGet, path+"?limit=10", b.Token, nil).expect(t, http.StatusOK).json(t, &page)
	if len(page.Messages) != 10 || page.Messages[0].Content != "m16" || page.Messages[9].Content != "m25" || page.NextCursor == "" {
		t.Fatalf("first page should be the latest 10 messages: %+v", page)
	}
	seen := 10
	for page.NextCursor != "" {
		cursor := page.NextCursor
		page = messagesPage{}
		env.do(http.MethodGet, path+"?limit=10&before="+cursor, b.Token, nil).expect(t, http.StatusOK).json(t, &page)
		seen += len(page.Messages)
	}
	if seen != 25 {
		t.Fatalf("pagination must cover all messages exactly once, saw %d", seen)
	}
	env.do(http.MethodGet, path+"?before=zzz", b.Token, nil).expect(t, http.StatusBadRequest)
}

func TestChatClearIsLocal(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	conv := "/conversations/" + env.match(a, b).ConversationID
	env.do(http.MethodPost, conv+"/messages", a.Token, map[string]string{"content": "old"}).expect(t, http.StatusCreated)

	env.do(http.MethodDelete, conv, b.Token, nil).expect(t, http.StatusNoContent)
	var list conversationList
	env.do(http.MethodGet, "/conversations", b.Token, nil).json(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("cleared conversation must disappear for the user who cleared it")
	}
	env.do(http.MethodGet, "/conversations", a.Token, nil).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("the other participant keeps the conversation")
	}

	// A new message brings the conversation back, without the cleared history.
	env.do(http.MethodPost, conv+"/messages", a.Token, map[string]string{"content": "new"}).expect(t, http.StatusCreated)
	var page messagesPage
	env.do(http.MethodGet, conv+"/messages", b.Token, nil).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Content != "new" {
		t.Fatalf("history before the clear must stay hidden: %+v", page)
	}
	env.do(http.MethodGet, "/conversations", b.Token, nil).json(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("conversation must reappear on a new message")
	}
}

func TestNotificationsFlow(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	m := env.match(a, b)

	type notifs struct {
		Notifications []struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Body   string         `json:"body"`
			IsRead bool           `json:"isRead"`
			Data   map[string]any `json:"data"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	var n notifs
	for _, s := range []session{a, b} {
		env.do(http.MethodGet, "/notifications", s.Token, nil).expect(t, http.StatusOK).json(t, &n)
		if n.UnreadCount != 1 || n.Notifications[0].Type != "match" || n.Notifications[0].Data["matchId"] != m.ID {
			t.Fatalf("both users get a match notification: %+v", n)
		}
	}
	if !strings.Contains(n.Notifications[0].Body, "Alice") && !strings.Contains(n.Notifications[0].Body, "Bob") {
		t.Fatalf("notification should name the match: %q", n.Notifications[0].Body)
	}

	// Messages notify once per burst; marking the conversation read clears it.
	conv := "/conversations/" + m.ConversationID
	env.do(http.MethodPost, conv+"/messages", a.Token, map[string]string{"content": "un"}).expect(t, http.StatusCreated)
	env.do(http.MethodPost, conv+"/messages", a.Token, map[string]string{"content": "deux"}).expect(t, http.StatusCreated)
	env.do(http.MethodGet, "/notifications", b.Token, nil).json(t, &n)
	if n.UnreadCount != 2 || n.Notifications[0].Type != "message" {
		t.Fatalf("expected match + one message notification: %+v", n)
	}
	env.do(http.MethodPost, conv+"/read", b.Token, nil).expect(t, http.StatusOK)
	env.do(http.MethodGet, "/notifications", b.Token, nil).json(t, &n)
	if n.UnreadCount != 1 {
		t.Fatalf("reading the conversation clears its notification: %+v", n)
	}

	// Mark one read; others cannot touch it.
	id := n.Notifications[len(n.Notifications)-1].ID
	env.do(http.MethodPost, "/notifications/"+id+"/read", a.Token, nil).expect(t, http.StatusNotFound)
	env.do(http.MethodPost, "/notifications/"+id+"/read", b.Token, nil).expect(t, http.StatusOK)
	env.do(http.MethodGet, "/notifications", b.Token, nil).json(t, &n)
	if n.UnreadCount != 0 {
		t.Fatalf("expected 0 unread: %+v", n)
	}
	env.do(http.MethodPost, "/notifications/read-all", a.Token, nil).expect(t, http.StatusNoContent)
	// Clients cannot create notifications for themselves or others.
	env.do(http.MethodPost, "/notifications", a.Token, map[string]string{"type": "x", "title": "x", "body": "x"}).expect(t, http.StatusNotFound)
}
