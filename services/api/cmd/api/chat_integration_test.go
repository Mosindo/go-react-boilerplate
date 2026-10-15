package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"net/http/httptest"
)

func (e *env) send(from *person, conv, body string) response {
	return e.do(http.MethodPost, "/chats/"+conv+"/messages", from.Token, map[string]string{"body": body})
}

func TestChatFlowReadStateAndPermissions(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Cleo", Gender: "woman", Seeks: []string{"man"}, Lat: 44.0, Lng: 14.0})
	b := e.signup(profileSpec{Name: "Cyrus", Gender: "man", Seeks: []string{"woman"}, Lat: 44.0, Lng: 14.0})
	stranger := e.signup(profileSpec{Name: "Stan", Gender: "man", Seeks: []string{"woman"}, Lat: 44.0, Lng: 14.0})
	conv := e.match(a, b)

	// A new match shows up as an empty conversation for both people.
	for _, p := range []*person{a, b} {
		list := e.mustStatus(e.do(http.MethodGet, "/chats", p.Token, nil), http.StatusOK, "chats").json(t)["conversations"].([]any)
		if len(list) != 1 || list[0].(map[string]any)["id"] != conv {
			t.Fatalf("%s should see the new conversation: %v", p.Name, list)
		}
		if list[0].(map[string]any)["lastMessage"] != nil {
			t.Fatal("a new match has no message yet")
		}
	}

	// Validation.
	for name, body := range map[string]string{"empty": "", "blank": "   \n ", "control only": "\x00\x01"} {
		if r := e.send(a, conv, body); r.Status != http.StatusBadRequest {
			t.Errorf("%s message must be rejected, got %d %s", name, r.Status, r.Body)
		}
	}
	tooLong := e.send(a, conv, strings.Repeat("é", 2001))
	if tooLong.Status != http.StatusBadRequest || tooLong.code(t) != "message_too_long" {
		t.Fatalf("2001 characters must be rejected, got %d %s", tooLong.Status, tooLong.Body)
	}
	e.mustStatus(e.send(a, conv, strings.Repeat("é", 2000)), http.StatusCreated, "exactly 2000 characters")

	sent := e.mustStatus(e.send(a, conv, "  Bonjour Cyrus !  "), http.StatusCreated, "send").json(t)
	if sent["body"] != "Bonjour Cyrus !" || sent["senderId"] != a.ID {
		t.Fatalf("unexpected message: %v", sent)
	}
	e.mustStatus(e.send(b, conv, "Salut Cleo"), http.StatusCreated, "reply")
	e.mustStatus(e.send(a, conv, "Ça va ?"), http.StatusCreated, "send again")

	// Unread counters and last message.
	chatB := e.mustStatus(e.do(http.MethodGet, "/chats", b.Token, nil), http.StatusOK, "chats B").json(t)["conversations"].([]any)[0].(map[string]any)
	// B's own reply counts as having read everything before it.
	if chatB["unreadCount"].(float64) != 1 {
		t.Fatalf("B should have 1 unread message, got %v", chatB["unreadCount"])
	}
	if chatB["lastMessage"].(map[string]any)["body"] != "Ça va ?" {
		t.Fatalf("unexpected last message: %v", chatB["lastMessage"])
	}
	chatA := e.mustStatus(e.do(http.MethodGet, "/chats", a.Token, nil), http.StatusOK, "chats A").json(t)["conversations"].([]any)[0].(map[string]any)
	if chatA["unreadCount"].(float64) != 0 {
		t.Fatalf("A wrote last, so has nothing unread, got %v", chatA["unreadCount"])
	}
	if chatA["peer"].(map[string]any)["firstName"] != "Cyrus" || chatA["peer"].(map[string]any)["thumbUrl"] == nil {
		t.Fatalf("peer info missing: %v", chatA["peer"])
	}

	// Message history, newest first, with a cursor.
	page := e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages?limit=2", b.Token, nil), http.StatusOK, "history").json(t)
	msgs := page["messages"].([]any)
	if len(msgs) != 2 || page["hasMore"] != true || msgs[0].(map[string]any)["body"] != "Ça va ?" {
		t.Fatalf("unexpected first page: %v", page)
	}
	older := e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages?limit=10&before="+msgs[1].(map[string]any)["id"].(string), b.Token, nil), http.StatusOK, "older").json(t)
	if len(older["messages"].([]any)) != 2 || older["hasMore"] != false {
		t.Fatalf("unexpected older page: %v", older)
	}
	e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages?before=zzz", b.Token, nil), http.StatusBadRequest, "bad cursor")

	// Read receipts.
	before := e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages", a.Token, nil), http.StatusOK, "A history").json(t)
	_ = before
	e.mustStatus(e.do(http.MethodPost, "/chats/"+conv+"/read", b.Token, nil), http.StatusOK, "B reads")
	after := e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages", a.Token, nil), http.StatusOK, "A history").json(t)
	if after["peerLastReadAt"] == nil {
		t.Fatal("the sender must be able to see when the other person read")
	}
	chatB = e.mustStatus(e.do(http.MethodGet, "/chats", b.Token, nil), http.StatusOK, "chats B").json(t)["conversations"].([]any)[0].(map[string]any)
	if chatB["unreadCount"].(float64) != 0 {
		t.Fatalf("unread must be cleared after reading, got %v", chatB["unreadCount"])
	}

	// Strangers cannot read, write, mark or hide.
	e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages", stranger.Token, nil), http.StatusNotFound, "stranger reads")
	e.mustStatus(e.send(stranger, conv, "let me in"), http.StatusNotFound, "stranger writes")
	e.mustStatus(e.do(http.MethodPost, "/chats/"+conv+"/read", stranger.Token, nil), http.StatusNotFound, "stranger marks read")
	e.mustStatus(e.do(http.MethodDelete, "/chats/"+conv, stranger.Token, nil), http.StatusNotFound, "stranger hides")
	e.mustStatus(e.do(http.MethodGet, "/chats/not-a-uuid/messages", a.Token, nil), http.StatusNotFound, "bad id")
	if list := e.mustStatus(e.do(http.MethodGet, "/chats", stranger.Token, nil), http.StatusOK, "stranger chats").json(t)["conversations"].([]any); len(list) != 0 {
		t.Fatalf("a stranger sees no conversations, got %v", list)
	}

	// Local deletion hides it for one side only, until a new message arrives.
	e.mustStatus(e.do(http.MethodDelete, "/chats/"+conv, b.Token, nil), http.StatusNoContent, "hide")
	if list := e.mustStatus(e.do(http.MethodGet, "/chats", b.Token, nil), http.StatusOK, "chats B").json(t)["conversations"].([]any); len(list) != 0 {
		t.Fatal("hidden conversation must disappear from the list")
	}
	if list := e.mustStatus(e.do(http.MethodGet, "/chats", a.Token, nil), http.StatusOK, "chats A").json(t)["conversations"].([]any); len(list) != 1 {
		t.Fatal("hiding is local: the other person keeps the conversation")
	}
	time.Sleep(20 * time.Millisecond)
	e.mustStatus(e.send(a, conv, "Tu es là ?"), http.StatusCreated, "message after hide")
	if list := e.mustStatus(e.do(http.MethodGet, "/chats", b.Token, nil), http.StatusOK, "chats B").json(t)["conversations"].([]any); len(list) != 1 {
		t.Fatal("a new message must bring the conversation back")
	}
}

func TestNotificationsForMatchAndMessages(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Nora", Gender: "woman", Seeks: []string{"man"}, Lat: 46.0, Lng: 16.0})
	b := e.signup(profileSpec{Name: "Noel", Gender: "man", Seeks: []string{"woman"}, Lat: 46.0, Lng: 16.0})
	conv := e.match(a, b)

	e.mustStatus(e.send(a, conv, "first"), http.StatusCreated, "m1")
	e.mustStatus(e.send(a, conv, "second"), http.StatusCreated, "m2")
	e.mustStatus(e.send(a, conv, "third"), http.StatusCreated, "m3")

	list := e.mustStatus(e.do(http.MethodGet, "/notifications", b.Token, nil), http.StatusOK, "notifications").json(t)
	items := list["notifications"].([]any)
	types := map[string]int{}
	for _, raw := range items {
		types[raw.(map[string]any)["type"].(string)]++
	}
	if types["match"] != 1 || types["message"] != 1 {
		t.Fatalf("expected one match and one coalesced message notification, got %v", types)
	}
	if list["unreadCount"].(float64) != 2 {
		t.Fatalf("expected 2 unread, got %v", list["unreadCount"])
	}
	var message map[string]any
	for _, raw := range items {
		if m := raw.(map[string]any); m["type"] == "message" {
			message = m
		}
	}
	if message["body"] != "third" || message["title"] != "Nora" {
		t.Fatalf("the message notification must show the latest message: %v", message)
	}

	// Mark one read, then opening the conversation clears the message one.
	id := items[0].(map[string]any)["id"].(string)
	e.mustStatus(e.do(http.MethodPost, "/notifications/"+id+"/read", b.Token, nil), http.StatusOK, "mark read")
	e.mustStatus(e.do(http.MethodPost, "/notifications/"+id+"/read", a.Token, nil), http.StatusNotFound, "someone else's notification")
	e.mustStatus(e.do(http.MethodPost, "/chats/"+conv+"/read", b.Token, nil), http.StatusOK, "open conversation")
	if got := e.mustStatus(e.do(http.MethodGet, "/notifications", b.Token, nil), http.StatusOK, "notifications").json(t)["unreadCount"].(float64); got != 1 {
		t.Fatalf("only the match notification should remain unread, got %v", got)
	}

	// A new message after reading creates a fresh unread notification.
	e.mustStatus(e.send(a, conv, "again"), http.StatusCreated, "m4")
	if got := e.mustStatus(e.do(http.MethodGet, "/notifications", b.Token, nil), http.StatusOK, "notifications").json(t)["unreadCount"].(float64); got != 2 {
		t.Fatalf("expected 2 unread, got %v", got)
	}
	e.mustStatus(e.do(http.MethodPost, "/notifications/read-all", b.Token, nil), http.StatusNoContent, "read all")
	if got := e.mustStatus(e.do(http.MethodGet, "/notifications", b.Token, nil), http.StatusOK, "notifications").json(t)["unreadCount"].(float64); got != 0 {
		t.Fatalf("read-all failed, got %v", got)
	}

	// The sender does not notify themselves, and clients cannot forge notifications.
	if got := e.mustStatus(e.do(http.MethodGet, "/notifications", a.Token, nil), http.StatusOK, "notifications A").json(t)["unreadCount"].(float64); got != 1 {
		t.Fatalf("A only has the match notification, got %v", got)
	}
	if r := e.do(http.MethodPost, "/notifications", a.Token, map[string]string{"type": "x", "title": "x", "body": "x"}); r.Status != http.StatusNotFound && r.Status != http.StatusMethodNotAllowed {
		t.Fatalf("clients must not be able to create notifications, got %d", r.Status)
	}
	e.mustStatus(e.do(http.MethodGet, "/notifications?limit=0", a.Token, nil), http.StatusBadRequest, "bad limit")
}

func TestRealtimeDeliversMessagesAndMatches(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Rita", Gender: "woman", Seeks: []string{"man"}, Lat: 47.0, Lng: 17.0})
	b := e.signup(profileSpec{Name: "Remy", Gender: "man", Seeks: []string{"woman"}, Lat: 47.0, Lng: 17.0})

	srv := httptest.NewServer(e.router)
	defer srv.Close()

	connect := func(p *person) *websocket.Conn {
		t.Helper()
		ticket := e.mustStatus(e.do(http.MethodPost, "/ws/ticket", p.Token, nil), http.StatusOK, "ticket").json(t)["ticket"].(string)
		u := url.URL{Scheme: "ws", Host: strings.TrimPrefix(srv.URL, "http://"), Path: "/ws", RawQuery: "ticket=" + ticket}
		conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })

		// Tickets are single use.
		if _, resp, err := websocket.DefaultDialer.Dial(u.String(), nil); err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatal("a ticket must not be reusable")
		}
		return conn
	}
	next := func(conn *websocket.Conn, wantType string) map[string]any {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(deadline)
			var event map[string]any
			if err := conn.ReadJSON(&event); err != nil {
				t.Fatalf("waiting for %s: %v", wantType, err)
			}
			if event["type"] == wantType {
				return event
			}
		}
		t.Fatalf("event %s never arrived", wantType)
		return nil
	}

	if r := e.do(http.MethodGet, "/ws?ticket=forged", "", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("forged ticket must be refused, got %d", r.Status)
	}

	wsA, wsB := connect(a), connect(b)

	e.mustStatus(e.swipe(a, b, "like"), http.StatusOK, "like")
	e.mustStatus(e.swipe(b, a, "like"), http.StatusOK, "like back")
	for _, ws := range []*websocket.Conn{wsA, wsB} {
		data := next(ws, "match.new")["data"].(map[string]any)
		if data["conversationId"] == "" {
			t.Fatalf("match event incomplete: %v", data)
		}
	}
	conv := next(wsA, "notification.new")["data"].(map[string]any)["data"].(map[string]any)["conversationId"].(string)

	e.mustStatus(e.send(a, conv, "live message"), http.StatusCreated, "send")
	got := next(wsB, "message.new")["data"].(map[string]any)
	if got["body"] != "live message" || got["conversationId"] != conv {
		t.Fatalf("unexpected realtime message: %v", got)
	}

	e.mustStatus(e.do(http.MethodPost, "/chats/"+conv+"/read", b.Token, nil), http.StatusOK, "read")
	if read := next(wsA, "message.read")["data"].(map[string]any); read["conversationId"] != conv {
		t.Fatalf("unexpected read event: %v", read)
	}

	e.mustStatus(e.do(http.MethodDelete, "/matches/"+a.ID, b.Token, nil), http.StatusNoContent, "unmatch")
	if removed := next(wsA, "match.removed")["data"].(map[string]any); removed["userId"] != b.ID {
		t.Fatalf("unexpected removal event: %v", removed)
	}
}
