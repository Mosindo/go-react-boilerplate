package realtime

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWebSocketAuthAndDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := NewHub()
	r := gin.New()
	r.GET("/ws", hub.Handler(func(token string) (string, string, time.Time, error) {
		if token != "good" {
			return "", "", time.Time{}, errors.New("bad token")
		}
		return "user-1", "session-1", time.Now().Add(time.Minute), nil
	}))
	server := httptest.NewServer(r)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("expected connection with bad token to be closed")
	}
	bad.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": "good"}); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	var ready Event
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&ready); err != nil || ready.Type != "ready" {
		t.Fatalf("expected ready, got %+v err=%v", ready, err)
	}

	hub.Publish("user-1", Event{Type: "message.new", Data: map[string]string{"id": "m1"}})
	hub.Publish("user-2", Event{Type: "message.new", Data: map[string]string{"id": "not-for-me"}})
	var got Event
	if err := conn.ReadJSON(&got); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if got.Type != "message.new" {
		t.Fatalf("unexpected event %+v", got)
	}
	data, _ := got.Data.(map[string]any)
	if data["id"] != "m1" {
		t.Fatalf("received someone else's event: %+v", got)
	}

	// Revoking the session closes the socket.
	hub.DisconnectSession("session-1")
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected socket to be closed after session revocation")
	}
}
