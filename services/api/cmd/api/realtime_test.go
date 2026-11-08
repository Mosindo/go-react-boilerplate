package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dialWS(t *testing.T, srvURL, token string) *websocket.Conn {
	t.Helper()
	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srvURL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	return c
}

func readEvent(t *testing.T, c *websocket.Conn) (string, json.RawMessage) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ev struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := c.ReadJSON(&ev); err != nil {
		t.Fatalf("read event: %v", err)
	}
	return ev.Type, ev.Data
}

func TestRealtimeEvents(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a := e.newUser(spec{name: "Wa", gender: "woman", interestedIn: []string{"man"}, lat: 1.35, lng: 103.82})
	b := e.newUser(spec{name: "Wb", gender: "man", interestedIn: []string{"woman"}, lat: 1.36, lng: 103.83})

	// Bad token is rejected right after the auth frame.
	bad := dialWS(t, srv.URL, "garbage")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("connection with an invalid token must be closed")
	}

	wsA := dialWS(t, srv.URL, a.Token)
	defer wsA.Close()
	wsB := dialWS(t, srv.URL, b.Token)
	defer wsB.Close()
	for _, c := range []*websocket.Conn{wsA, wsB} {
		if typ, _ := readEvent(t, c); typ != "ready" {
			t.Fatalf("expected ready, got %s", typ)
		}
	}

	e.swipe(a, b, "like")
	_, _, conv := e.swipe(b, a, "like")

	// Both receive match.new (and the matching notification), in any interleaving.
	for name, c := range map[string]*websocket.Conn{"a": wsA, "b": wsB} {
		seen := map[string]bool{}
		for len(seen) < 2 {
			typ, _ := readEvent(t, c)
			seen[typ] = true
		}
		if !seen["match.new"] || !seen["notification.new"] {
			t.Fatalf("%s: expected match.new + notification.new, got %v", name, seen)
		}
	}

	e.call("POST", "/conversations/"+conv+"/messages", map[string]any{"body": "live!"}, a.Token, 201, nil)
	for {
		typ, data := readEvent(t, wsB)
		if typ != "message.new" {
			continue
		}
		var m struct {
			Body     string `json:"body"`
			SenderID string `json:"senderId"`
		}
		_ = json.Unmarshal(data, &m)
		if m.Body != "live!" || m.SenderID != a.ID {
			t.Fatalf("unexpected message event: %s", data)
		}
		break
	}

	e.call("POST", "/conversations/"+conv+"/read", nil, b.Token, 204, nil)
	for {
		if typ, _ := readEvent(t, wsA); typ == "conversation.read" {
			break
		}
	}

	// Account deletion closes live connections.
	e.call("DELETE", "/me", map[string]any{"password": "correct-horse"}, b.Token, 204, nil)
	_ = wsB.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := wsB.ReadMessage(); err != nil {
			break
		}
	}
}
