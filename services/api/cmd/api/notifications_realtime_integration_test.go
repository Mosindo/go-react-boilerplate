package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func notifications(e *env, u user, query string) map[string]any {
	e.t.Helper()
	return e.expect(e.do(http.MethodGet, "/notifications"+query, nil, u.Token), http.StatusOK).JSON(e.t)
}

func TestNotificationsForMatchesAndMessages(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b) // a liked first, b completed the match

	// Only the person who was waiting (a) is notified of the match.
	na := notifications(e, a, "")
	items := na["notifications"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["type"] != "match" || na["unreadCount"].(float64) != 1 {
		t.Fatalf("unexpected match notification: %v", na)
	}
	data := items[0].(map[string]any)["data"].(map[string]any)
	if data["matchId"] != matchID || data["userId"] != b.ID {
		t.Fatalf("notification data wrong: %v", data)
	}
	if notifications(e, b, "")["unreadCount"].(float64) != 0 {
		t.Fatal("the swiper who completed the match sees the match in-app, no notification")
	}

	// Messages coalesce into one rolling notification per conversation.
	for i := 0; i < 3; i++ {
		e.expect(send(e, b, matchID, "msg"+string(rune('1'+i))), http.StatusCreated)
	}
	nl := notifications(e, a, "")
	if nl["unreadCount"].(float64) != 2 {
		t.Fatalf("expected match + one coalesced message notification, got %v", nl["unreadCount"])
	}
	first := nl["notifications"].([]any)[0].(map[string]any)
	if first["type"] != "message" || first["body"] != "msg3" || !strings.Contains(first["title"].(string), "Berta") {
		t.Fatalf("unexpected message notification %v", first)
	}

	// Opening the conversation clears its message notification.
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", nil, a.Token), http.StatusNoContent)
	if notifications(e, a, "")["unreadCount"].(float64) != 1 {
		t.Fatal("reading the conversation must clear its message notification")
	}

	// Mark one / mark all / ownership.
	id := items[0].(map[string]any)["id"].(string)
	e.expect(e.do(http.MethodPost, "/notifications/"+id+"/read", nil, b.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/notifications/"+id+"/read", nil, a.Token), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/notifications/"+id+"/read", nil, a.Token), http.StatusNoContent)
	if notifications(e, a, "")["unreadCount"].(float64) != 0 {
		t.Fatal("unread should be zero")
	}
	e.expect(send(e, b, matchID, "again"), http.StatusCreated)
	if notifications(e, a, "")["unreadCount"].(float64) != 1 {
		t.Fatal("a new message after reading creates a fresh notification")
	}
	e.expect(e.do(http.MethodPost, "/notifications/read-all", nil, a.Token), http.StatusNoContent)
	if notifications(e, a, "")["unreadCount"].(float64) != 0 {
		t.Fatal("read-all must clear everything")
	}
	e.expect(e.do(http.MethodPost, "/notifications/not-a-uuid/read", nil, a.Token), http.StatusNotFound)
}

func TestNotificationsPagination(t *testing.T) {
	e := newEnv(t)
	me := e.register("notif-many")
	e.onboard(me, "Max", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	for i := 0; i < 3; i++ {
		other := e.register("notif-w")
		e.onboard(other, "Wanda", profileOpts{gender: "woman", interestedIn: []string{"man"}})
		e.match(me, other)
	}
	p1 := notifications(e, me, "?limit=2")
	if len(p1["notifications"].([]any)) != 2 || p1["nextCursor"] == nil {
		t.Fatalf("bad first page %v", p1)
	}
	p2 := notifications(e, me, "?limit=2&cursor="+p1["nextCursor"].(string))
	if len(p2["notifications"].([]any)) != 1 || p2["nextCursor"] != nil {
		t.Fatalf("bad second page %v", p2)
	}
	e.expect(e.do(http.MethodGet, "/notifications?cursor=@@@", nil, me.Token), http.StatusBadRequest)
}

// ---- WebSocket ----

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func authWS(t *testing.T, conn *websocket.Conn, token string) {
	t.Helper()
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	var ev map[string]any
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := conn.ReadJSON(&ev); err != nil || ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v %v", ev, err)
	}
}

func nextEvent(t *testing.T, conn *websocket.Conn, wantType string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %s: %v", wantType, err)
		}
		var ev map[string]any
		_ = json.Unmarshal(raw, &ev)
		if ev["type"] == wantType {
			return ev
		}
	}
	t.Fatalf("timed out waiting for %s", wantType)
	return nil
}

func TestRealtimeEvents(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a, b := e.pair()
	wsA, wsB := dialWS(t, srv), dialWS(t, srv)
	authWS(t, wsA, a.Token)
	authWS(t, wsB, b.Token)

	// Match: both are told, the earlier liker also gets a notification.
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": b.ID, "action": "like"}, a.Token), http.StatusOK)
	r := e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": a.ID, "action": "like"}, b.Token), http.StatusOK).JSON(t)
	matchID := r["match"].(map[string]any)["id"].(string)
	evA := nextEvent(t, wsA, "match.new")
	if evA["data"].(map[string]any)["user"].(map[string]any)["userId"] != b.ID {
		t.Fatalf("a should see b as the match: %v", evA)
	}
	if nextEvent(t, wsB, "match.new")["data"].(map[string]any)["user"].(map[string]any)["userId"] != a.ID {
		t.Fatal("b should see a as the match")
	}
	if nextEvent(t, wsA, "notification.new")["data"].(map[string]any)["type"] != "match" {
		t.Fatal("expected a match notification event")
	}

	// Message delivery and read receipts.
	e.expect(send(e, a, matchID, "temps réel !"), http.StatusCreated)
	msg := nextEvent(t, wsB, "message.new")["data"].(map[string]any)
	if msg["content"] != "temps réel !" || msg["senderId"] != a.ID {
		t.Fatalf("unexpected message event %v", msg)
	}
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", nil, b.Token), http.StatusNoContent)
	rd := nextEvent(t, wsA, "message.read")["data"].(map[string]any)
	if rd["matchId"] != matchID || rd["readerId"] != b.ID {
		t.Fatalf("unexpected read event %v", rd)
	}

	// Unmatch notifies both.
	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, nil, a.Token), http.StatusNoContent)
	if nextEvent(t, wsB, "match.removed")["data"].(map[string]any)["matchId"] != matchID {
		t.Fatal("b must learn about the unmatch")
	}

	// Ping/pong keeps the app-level heartbeat.
	_ = wsB.WriteJSON(map[string]string{"type": "ping"})
	nextEvent(t, wsB, "pong")
}

func TestRealtimeIsolatesUsers(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	a, b := e.pair()
	matchID := e.match(a, b)
	outsider := e.register("ws-outsider")
	wsOut := dialWS(t, srv)
	authWS(t, wsOut, outsider.Token)

	e.expect(send(e, a, matchID, "private"), http.StatusCreated)
	_ = wsOut.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	if _, raw, err := wsOut.ReadMessage(); err == nil {
		t.Fatalf("outsider received %s", raw)
	}
}

func TestRealtimeRejectsBadAuth(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	u := e.register("ws-bad")

	for name, frame := range map[string]any{
		"garbage token": map[string]string{"type": "auth", "token": "nope"},
		"wrong type":    map[string]string{"type": "hello", "token": u.Token},
		"empty token":   map[string]string{"type": "auth", "token": ""},
	} {
		conn := dialWS(t, srv)
		_ = conn.WriteJSON(frame)
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Errorf("%s: connection should have been closed", name)
		}
	}

	// A token whose session was revoked is refused, too.
	e.expect(e.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": u.Refresh}, ""), http.StatusNoContent)
	conn := dialWS(t, srv)
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": u.Token})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Error("revoked session must not open a socket")
	}

	// Browsers from foreign origins are refused by the upgrader.
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	_, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		t.Error("foreign origin must be refused")
	} else if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}
