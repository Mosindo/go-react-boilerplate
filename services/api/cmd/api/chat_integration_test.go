package main

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type msgResp struct {
	ID             int64      `json:"id"`
	ConversationID string     `json:"conversationId"`
	SenderID       string     `json:"senderId"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReadAt         *time.Time `json:"readAt"`
}

type convList struct {
	Conversations []struct {
		ID          string `json:"id"`
		MatchID     string `json:"matchId"`
		UnreadCount int    `json:"unreadCount"`
		User        struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
		} `json:"user"`
		LastMessage *msgResp `json:"lastMessage"`
	} `json:"conversations"`
	NextBefore *time.Time `json:"nextBefore"`
}

// matchedPair returns two users who matched and their conversation id.
func (e *testEnv) matchedPair(an, bn string) (a, b testUser, conv string) {
	e.t.Helper()
	a = e.onboard(userSpec{Name: an, Gender: "woman", Into: []string{"man"}})
	b = e.onboard(userSpec{Name: bn, Gender: "man", Into: []string{"woman"}})
	e.swipeOK(a, b, "like")
	m := e.swipeOK(b, a, "like")
	if !m.Matched {
		e.t.Fatal("expected match")
	}
	return a, b, m.Match.ConversationID
}

func TestChatSendReadAndPermissions(t *testing.T) {
	e := newTestEnv(t)
	a, b, conv := e.matchedPair("Alice", "Bob")
	c := e.onboard(userSpec{Name: "Carol", Gender: "woman", Into: []string{"man"}})

	// Empty conversation is listed for both (new match).
	var list convList
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].User.FirstName != "Bob" || list.Conversations[0].LastMessage != nil {
		t.Fatalf("unexpected conversation list: %+v", list)
	}

	// Validation.
	send := func(u testUser, id, body string) response {
		return e.do("POST", "/conversations/"+id+"/messages", u.Token, map[string]string{"body": body})
	}
	e.expect(send(a, conv, "   "), 400)
	e.expect(send(a, conv, strings.Repeat("é", 2001)), 400)
	e.expect(send(a, conv, "bad\x00char"), 400)
	e.expect(send(a, "not-a-uuid", "hi"), 404)

	// Outsider can neither read, write, mark read nor hide.
	e.expect(send(c, conv, "let me in"), 404)
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", c.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+conv+"/read", c.Token, nil), 404)
	e.expect(e.do("DELETE", "/conversations/"+conv, c.Token, nil), 404)

	var m1, m2 msgResp
	e.expect(send(a, conv, "  Hello Bob  "), 201).decode(t, &m1)
	e.expect(send(a, conv, "Are you there?"), 201).decode(t, &m2)
	if m1.Body != "Hello Bob" || m1.ReadAt != nil || m1.SenderID != a.ID || m2.ID <= m1.ID {
		t.Fatalf("unexpected message: %+v / %+v", m1, m2)
	}

	// Unread counts and last message for the recipient only.
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).decode(t, &list)
	if list.Conversations[0].UnreadCount != 2 || list.Conversations[0].LastMessage.Body != "Are you there?" {
		t.Fatalf("recipient should see 2 unread: %+v", list.Conversations[0])
	}
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).decode(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatalf("sender has nothing unread: %+v", list.Conversations[0])
	}

	// Messages notification is coalesced per conversation: one unread entry.
	var n struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/notifications", b.Token, nil), 200).decode(t, &n)
	if n.UnreadCount != 2 { // match + one message notification
		t.Fatalf("expected match+message notifications, got %+v", n)
	}

	// History: oldest first, paginated by cursor.
	var page struct {
		Messages   []msgResp `json:"messages"`
		NextBefore *int64    `json:"nextBefore"`
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=1", b.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Are you there?" || page.NextBefore == nil {
		t.Fatalf("unexpected first page: %+v", page)
	}
	e.expect(e.do("GET", fmt.Sprintf("/conversations/%s/messages?limit=1&before=%d", conv, *page.NextBefore), b.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Hello Bob" || page.NextBefore != nil {
		t.Fatalf("unexpected second page: %+v", page)
	}

	// Mark read: sender sees readAt, counts and notifications reset.
	var rr struct {
		ReadCount int `json:"readCount"`
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 200).decode(t, &rr)
	if rr.ReadCount != 2 {
		t.Fatalf("expected 2 messages marked read, got %d", rr.ReadCount)
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 200).decode(t, &rr)
	if rr.ReadCount != 0 {
		t.Fatal("second read must be a no-op")
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).decode(t, &page)
	if page.Messages[0].ReadAt == nil || page.Messages[1].ReadAt == nil {
		t.Fatalf("sender should see read receipts: %+v", page.Messages)
	}
	e.expect(e.do("GET", "/notifications", b.Token, nil), 200).decode(t, &n)
	if n.UnreadCount != 1 { // match notification remains; message one is cleared
		t.Fatalf("expected only the match notification unread, got %d", n.UnreadCount)
	}
	e.expect(e.do("POST", "/notifications/"+n.Notifications[0].ID+"/read", b.Token, nil), 204)
	e.expect(e.do("POST", "/notifications/"+n.Notifications[0].ID+"/read", c.Token, nil), 404) // someone else's
	e.expect(e.do("POST", "/notifications/read-all", b.Token, nil), 204)
	e.expect(e.do("GET", "/notifications", b.Token, nil), 200).decode(t, &n)
	if n.UnreadCount != 0 {
		t.Fatalf("read-all should clear unread, got %d", n.UnreadCount)
	}
}

func TestChatHideClearsLocalHistoryOnly(t *testing.T) {
	e := newTestEnv(t)
	a, b, conv := e.matchedPair("Alice", "Bob")
	send := func(u testUser, body string) {
		e.expect(e.do("POST", "/conversations/"+conv+"/messages", u.Token, map[string]string{"body": body}), 201)
	}
	send(a, "one")
	send(b, "two")
	time.Sleep(5 * time.Millisecond)

	e.expect(e.do("DELETE", "/conversations/"+conv, a.Token, nil), 204)
	var list convList
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("hidden conversation must not be listed")
	}
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("the other participant keeps the conversation")
	}

	time.Sleep(5 * time.Millisecond)
	send(b, "three")
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 {
		t.Fatalf("a new message resurfaces the conversation: %+v", list)
	}
	var page struct {
		Messages []msgResp `json:"messages"`
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "three" {
		t.Fatalf("locally cleared history must stay hidden: %+v", page.Messages)
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 3 {
		t.Fatalf("the other side keeps full history: %+v", page.Messages)
	}
}

func TestBlockAndReport(t *testing.T) {
	e := newTestEnv(t)
	a, b, conv := e.matchedPair("Alice", "Bob")
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "hi"}), 201)

	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": a.ID}), 400)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": "bad"}), 400)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": "11111111-1111-4111-8111-111111111111"}), 404)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), 204)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), 204) // idempotent

	// Match, conversation and messages are gone for both; the blocked user can neither write nor see.
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", b.Token, map[string]string{"body": "hello?"}), 404)
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 404)
	e.expect(e.swipe(b, a, "like"), 404)
	var list convList
	e.expect(e.do("GET", "/conversations", b.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("blocked user still sees the conversation")
	}
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("blocked users must not see each other in discovery")
	}

	var blocks struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	e.expect(e.do("GET", "/blocks", a.Token, nil), 200).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID || blocks.Blocks[0].FirstName != "Bob" {
		t.Fatalf("unexpected block list: %+v", blocks)
	}
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), 204)
	e.expect(e.do("GET", "/blocks", a.Token, nil), 200).decode(t, &blocks)
	if len(blocks.Blocks) != 0 {
		t.Fatal("unblock failed")
	}

	// Reports.
	report := func(u testUser, target string, reason string) response {
		return e.do("POST", "/reports", u.Token, map[string]any{"userId": target, "reason": reason, "details": "  rude  "})
	}
	e.expect(report(a, b.ID, "bogus"), 400)
	e.expect(report(a, a.ID, "spam"), 400)
	e.expect(report(a, "11111111-1111-4111-8111-111111111111", "spam"), 404)
	e.expect(report(a, b.ID, "harassment"), 201)
	e.expect(report(a, b.ID, "harassment"), 201) // duplicate open report is absorbed
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "other", "details": strings.Repeat("x", 1001)}), 400)
	var n int
	var details string
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*), MIN(details) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n, &details)
	if n != 1 || details != "rude" {
		t.Fatalf("expected one trimmed report, got %d %q", n, details)
	}
	// Reporting with block=true also blocks.
	e.expect(e.do("POST", "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "spam", "block": true}), 201)
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, b.ID, a.ID).Scan(&n)
	if n != 1 {
		t.Fatal("report with block=true must block")
	}
}

func TestRealtimeDeliversMessagesMatchesAndReadReceipts(t *testing.T) {
	e := newTestEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a := e.onboard(userSpec{Name: "Alice", Gender: "woman", Into: []string{"man"}})
	b := e.onboard(userSpec{Name: "Bob", Gender: "man", Into: []string{"woman"}})

	dial := func(token string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
			t.Fatal(err)
		}
		return conn
	}
	next := func(conn *websocket.Conn) map[string]any {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read event: %v", err)
		}
		return ev
	}
	expectType := func(conn *websocket.Conn, typ string) map[string]any {
		t.Helper()
		for i := 0; i < 5; i++ {
			if ev := next(conn); ev["type"] == typ {
				return ev
			}
		}
		t.Fatalf("event %q never arrived", typ)
		return nil
	}

	// Wrong token is refused.
	bad := dial("garbage")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("connection with an invalid token must be closed")
	}

	ca, cb := dial(a.Token), dial(b.Token)
	defer ca.Close()
	defer cb.Close()
	expectType(ca, "ready")
	expectType(cb, "ready")

	e.swipeOK(a, b, "like")
	m := e.swipeOK(b, a, "like")
	expectType(ca, "match")
	expectType(cb, "match")

	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/messages", a.Token, map[string]string{"body": "live!"}), 201)
	ev := expectType(cb, "message")
	if data := ev["data"].(map[string]any); data["body"] != "live!" || data["senderId"] != a.ID {
		t.Fatalf("unexpected message event: %v", ev)
	}

	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/read", b.Token, nil), 200)
	if ev := expectType(ca, "read"); ev["data"].(map[string]any)["readerId"] != b.ID {
		t.Fatalf("unexpected read event: %v", ev)
	}
}
