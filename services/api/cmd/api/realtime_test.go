package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func readEvent(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ev map[string]any
	if err := conn.ReadJSON(&ev); err != nil {
		t.Fatalf("read ws event: %v", err)
	}
	return ev
}

func TestRealtimeDeliversMessagesMatchesAndNotifications(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a := e.newPerson(persona{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.newPerson(persona{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})

	conn := dialWS(t, srv)
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": b.Token}); err != nil {
		t.Fatal(err)
	}
	if ev := readEvent(t, conn); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	// A likes B, then B likes A: B receives match.new + notification.new live.
	e.swipe(a, b, "like", http.StatusCreated)
	matchID := e.swipe(b, a, "like", http.StatusCreated).MatchID
	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		ev := readEvent(t, conn)
		seen[ev["type"].(string)] = true
	}
	if !seen["match.new"] || !seen["notification.new"] {
		t.Fatalf("expected match.new and notification.new, got %v", seen)
	}

	// A message from A reaches B in real time.
	e.send(a, matchID, "live!", http.StatusCreated)
	found := false
	for i := 0; i < 2 && !found; i++ {
		ev := readEvent(t, conn)
		if ev["type"] == "message.new" {
			found = ev["data"].(map[string]any)["body"] == "live!"
		}
	}
	if !found {
		t.Fatal("message.new not delivered")
	}
}

func TestRealtimeRejectsUnauthenticatedAndRevokedClients(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	u := e.register("ws")

	for name, frame := range map[string]map[string]string{
		"bad token":   {"type": "auth", "token": "garbage"},
		"wrong frame": {"type": "hello"},
	} {
		conn := dialWS(t, srv)
		_ = conn.WriteJSON(frame)
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Errorf("%s: connection must be closed", name)
		}
	}

	// A token whose session was revoked cannot open a socket either.
	e.expect(e.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": u.Refresh}), http.StatusNoContent)
	conn := dialWS(t, srv)
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": u.Token})
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("revoked session must not connect")
	}

	// Silence also gets dropped (auth deadline), checked via a short custom wait would slow
	// the suite, so only the explicit rejections above are asserted.
}
