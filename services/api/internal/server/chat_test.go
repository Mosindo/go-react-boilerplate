package server_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type msgPage struct {
	Messages []struct {
		ID       string    `json:"id"`
		SenderID string    `json:"senderId"`
		Body     string    `json:"body"`
		Created  time.Time `json:"createdAt"`
	} `json:"messages"`
	PeerLastReadAt time.Time `json:"peerLastReadAt"`
	NextCursor     string    `json:"nextCursor"`
}

type convList struct {
	Conversations []struct {
		ID          string `json:"id"`
		UnreadCount int    `json:"unreadCount"`
		LastMessage *struct {
			Body string `json:"body"`
		} `json:"lastMessage"`
		Profile card `json:"profile"`
	} `json:"conversations"`
	NextCursor string `json:"nextCursor"`
}

func TestChatFlowAndPermissions(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)
	outsider := e.onboard("Oscar", man)
	conv := e.match(a, b)

	var list convList
	a.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].LastMessage != nil || list.Conversations[0].Profile.UserID != b.ID {
		t.Fatalf("new match shows as empty conversation: %+v", list)
	}

	// Validation.
	for name, body := range map[string]string{"empty": "   ", "too long": strings.Repeat("é", 2001), "control": "hi\x00there"} {
		if r := a.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": body}); r.Status != 400 || r.code() != "invalid_message" {
			t.Errorf("%s: expected invalid_message, got %d %s", name, r.Status, r.Body)
		}
	}

	a.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "  Salut !  "})
	time.Sleep(5 * time.Millisecond)
	a.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "Ça va ?\nTout va bien"})

	// Unread state from Beth's point of view.
	b.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if list.Conversations[0].UnreadCount != 2 || list.Conversations[0].LastMessage.Body != "Ça va ?\nTout va bien" {
		t.Fatalf("expected 2 unread: %+v", list)
	}
	var page msgPage
	b.expect(200, "GET", "/conversations/"+conv+"/messages", nil).decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "Ça va ?\nTout va bien" || page.Messages[1].Body != "Salut !" {
		t.Fatalf("messages must be newest first and trimmed: %+v", page)
	}

	// Read receipts.
	a.expect(200, "GET", "/conversations/"+conv+"/messages", nil).decode(t, &page)
	if !page.PeerLastReadAt.Before(page.Messages[1].Created) {
		t.Fatal("before Beth reads, her read marker must be older than the messages")
	}
	b.expect(204, "POST", "/conversations/"+conv+"/read", nil)
	a.expect(200, "GET", "/conversations/"+conv+"/messages", nil).decode(t, &page)
	if page.PeerLastReadAt.Before(page.Messages[0].Created) {
		t.Fatal("after reading, Adam must see his messages as read")
	}
	b.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after read")
	}

	// Pagination (cursor).
	for i := 0; i < 5; i++ {
		b.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "m" + string(rune('0'+i))})
	}
	b.expect(200, "GET", "/conversations/"+conv+"/messages?limit=3", nil).decode(t, &page)
	if len(page.Messages) != 3 || page.NextCursor == "" {
		t.Fatalf("expected a page of 3 with cursor: %+v", page)
	}
	var next msgPage
	b.expect(200, "GET", "/conversations/"+conv+"/messages?limit=10&cursor="+page.NextCursor, nil).decode(t, &next)
	if len(next.Messages) != 4 || next.NextCursor != "" {
		t.Fatalf("expected remaining 4 messages: %d", len(next.Messages))
	}
	b.expect(400, "GET", "/conversations/"+conv+"/messages?cursor=garbage", nil)
	b.expect(400, "GET", "/conversations?limit=abc", nil)

	// An outsider can neither read, write, mark read, hide nor see the conversation.
	outsider.expect(404, "GET", "/conversations/"+conv, nil)
	outsider.expect(404, "GET", "/conversations/"+conv+"/messages", nil)
	outsider.expect(404, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "intruder"})
	outsider.expect(404, "POST", "/conversations/"+conv+"/read", nil)
	outsider.expect(404, "DELETE", "/conversations/"+conv, nil)
	outsider.expect(404, "GET", "/conversations/not-a-uuid/messages", nil)
	outsider.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("outsider must not list other people's conversations")
	}
	if r := e.do("GET", "/conversations/"+conv+"/messages", "", nil); r.Status != 401 {
		t.Fatalf("auth required: %d", r.Status)
	}

	// Local deletion hides the conversation for me only, and a new message resurfaces it.
	a.expect(204, "DELETE", "/conversations/"+conv, nil)
	a.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("hidden conversation must not be listed")
	}
	b.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("hiding is local: Beth still sees it")
	}
	b.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "tu es là ?"})
	a.expect(200, "GET", "/conversations", nil).decode(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 {
		t.Fatalf("new message resurfaces the conversation: %+v", list)
	}
}

func readEvent(t *testing.T, c *websocket.Conn) (string, json.RawMessage) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	var ev struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatal(err)
	}
	return ev.Type, ev.Data
}

func (u *user) connectWS(token string) *websocket.Conn {
	u.e.t.Helper()
	url := "ws" + strings.TrimPrefix(u.e.srv.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		u.e.t.Fatal(err)
	}
	u.e.t.Cleanup(func() { c.CloseNow() })
	_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","token":"`+token+`"}`))
	return c
}

func TestRealtimeDelivery(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("Adam", man)
	b := e.onboard("Beth", woman)

	wsA, wsB := a.connectWS(a.Access), b.connectWS(b.Access)
	if typ, _ := readEvent(t, wsA); typ != "ready" {
		t.Fatalf("expected ready, got %s", typ)
	}
	if typ, _ := readEvent(t, wsB); typ != "ready" {
		t.Fatalf("expected ready, got %s", typ)
	}

	conv := e.match(a, b)
	// Adam liked first, Beth completed the match: Adam is notified live.
	typ, data := readEvent(t, wsA)
	if typ != "match.new" || !strings.Contains(string(data), conv) {
		t.Fatalf("expected match.new for Adam, got %s %s", typ, data)
	}

	b.expect(201, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": "live!"})
	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		typ, data := readEvent(t, wsA)
		got[typ] = true
		if typ == "message.new" && !strings.Contains(string(data), "live!") {
			t.Fatalf("payload: %s", data)
		}
	}
	if !got["message.new"] || !got["notification.new"] {
		t.Fatalf("Adam should receive message.new and notification.new, got %v", got)
	}
	// The sender's other devices also get the message; no leakage to third parties.
	for sawEcho := false; !sawEcho; {
		typ, _ := readEvent(t, wsB) // skips the match notification
		sawEcho = typ == "message.new"
	}

	a.expect(204, "POST", "/conversations/"+conv+"/read", nil)
	if typ, _ := readEvent(t, wsB); typ != "message.read" {
		t.Fatalf("Beth should get message.read, got %s", typ)
	}
	if e.hub.ConnCount(a.ID) != 1 || e.hub.ConnCount(b.ID) != 1 {
		t.Fatal("each user has one live connection")
	}
}

func TestRealtimeRejectsBadAuth(t *testing.T) {
	e := newEnv(t)
	a := e.register("Adam")

	c := a.connectWS("not-a-token")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, _, err := c.Read(ctx); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("invalid token must close with policy violation, got %v", err)
	}

	// A first message that is not an auth message is refused too.
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	c2, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.CloseNow()
	_ = c2.Write(ctx, websocket.MessageText, []byte(`{"type":"ping"}`))
	if _, _, err := c2.Read(ctx); websocket.CloseStatus(err) != websocket.StatusPolicyViolation {
		t.Fatalf("unauthenticated ping must be refused, got %v", err)
	}
	if e.hub.ConnCount(a.ID) != 0 {
		t.Fatal("no connection should be registered")
	}
}
