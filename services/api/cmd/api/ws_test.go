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

func dialWS(t *testing.T, srv *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.WriteJSON(map[string]string{"type": "auth", "token": token})
	return c
}

func readEvent(t *testing.T, c *websocket.Conn) map[string]any {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read ws: %v", err)
	}
	var ev map[string]any
	_ = json.Unmarshal(data, &ev)
	return ev
}

func TestRealtimeMessageDelivery(t *testing.T) {
	r := newRouter()
	srv := httptest.NewServer(r)
	defer srv.Close()
	a, b, conv := matchPair(t, r)

	wsB := dialWS(t, srv, b.Token)
	defer wsB.Close()
	if ev := readEvent(t, wsB); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "temps réel"}, a.Token)
	ev := readEvent(t, wsB)
	if ev["type"] != "message.new" {
		t.Fatalf("expected message.new, got %v", ev)
	}
	data := ev["data"].(map[string]any)
	if data["body"] != "temps réel" || data["conversationId"] != conv {
		t.Fatalf("payload: %v", data)
	}

	// Read receipts reach the sender in real time.
	wsA := dialWS(t, srv, a.Token)
	defer wsA.Close()
	readEvent(t, wsA) // ready
	do(t, r, http.MethodPost, "/conversations/"+conv+"/read", nil, b.Token)
	if ev := readEvent(t, wsA); ev["type"] != "conversation.read" {
		t.Fatalf("expected conversation.read, got %v", ev)
	}
}

func TestWebSocketRejectsBadAuth(t *testing.T) {
	r := newRouter()
	srv := httptest.NewServer(r)
	defer srv.Close()
	c := dialWS(t, srv, "not-a-token")
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("connection should be closed for an invalid token")
	}
}

func TestWebSocketDoesNotLeakOtherUsersEvents(t *testing.T) {
	r := newRouter()
	srv := httptest.NewServer(r)
	defer srv.Close()
	a, _, conv := matchPair(t, r)
	eve := onboard(t, r, "Eve", "woman", []string{"man"}, 0, 0)

	ws := dialWS(t, srv, eve.Token)
	defer ws.Close()
	readEvent(t, ws)
	do(t, r, http.MethodPost, "/conversations/"+conv+"/messages", map[string]string{"body": "secret"}, a.Token)
	_ = ws.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if _, data, err := ws.ReadMessage(); err == nil {
		t.Fatalf("outsider received %s", data)
	}
}
