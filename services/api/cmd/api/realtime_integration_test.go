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

type wsClient struct {
	t    *testing.T
	conn *websocket.Conn
}

type wsEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (e *testEnv) dialWS(srv *httptest.Server, token string) *wsClient {
	e.t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		e.t.Fatalf("dial: %v", err)
	}
	e.t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		e.t.Fatal(err)
	}
	return &wsClient{t: e.t, conn: conn}
}

// next waits for the next event of the given type (skipping others) or fails.
func (c *wsClient) next(eventType string) wsEvent {
	c.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = c.conn.SetReadDeadline(deadline)
		var ev wsEvent
		if err := c.conn.ReadJSON(&ev); err != nil {
			c.t.Fatalf("waiting for %q: %v", eventType, err)
		}
		if ev.Type == eventType {
			return ev
		}
	}
}

func TestRealtimeEvents(t *testing.T) {
	env := newEnv(t)
	srv := httptest.NewServer(env.router)
	defer srv.Close()
	a, b := env.pair()

	wa, wb := env.dialWS(srv, a.Token), env.dialWS(srv, b.Token)
	wa.next("ready")
	wb.next("ready")

	// The person who liked first is told of the match in real time; the one who completes it learns it from
	// the HTTP response (and both get a notification).
	env.swipe(a, b.ID, "like").expect(t, http.StatusOK)
	var r swipeBody
	env.swipe(b, a.ID, "like").expect(t, http.StatusOK).json(t, &r)
	wa.next("match.new")
	wa.next("notification.new")
	wb.next("notification.new")

	// Messages and read receipts arrive in real time, for the recipient (and the sender's other devices).
	conv := "/conversations/" + r.Match.ConversationID
	env.do(http.MethodPost, conv+"/messages", a.Token, map[string]string{"content": "coucou"}).expect(t, http.StatusCreated)
	ev := wb.next("message.new")
	var msg chatMsg
	_ = json.Unmarshal(ev.Data, &msg)
	if msg.Content != "coucou" || msg.SenderUserID != a.ID {
		t.Fatalf("unexpected pushed message: %+v", msg)
	}
	wa.next("message.new")
	wb.next("notification.new")

	env.do(http.MethodPost, conv+"/read", b.Token, nil).expect(t, http.StatusOK)
	wa.next("messages.read")

	// An outsider's socket receives nothing about this conversation.
	c := env.register("spy")
	wc := env.dialWS(srv, c.Token)
	wc.next("ready")
	env.do(http.MethodPost, conv+"/messages", b.Token, map[string]string{"content": "private"}).expect(t, http.StatusCreated)
	wa.next("message.new")
	_ = wc.conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	var leaked wsEvent
	if err := wc.conn.ReadJSON(&leaked); err == nil {
		t.Fatalf("outsider received %q", leaked.Type)
	}

	// Unmatching and blocking tell both clients to drop the conversation.
	env.do(http.MethodDelete, "/matches/"+r.Match.ID, a.Token, nil).expect(t, http.StatusNoContent)
	wa.next("match.removed")
	wb.next("match.removed")
}

func TestRealtimeRejectsBadAuth(t *testing.T) {
	env := newEnv(t)
	srv := httptest.NewServer(env.router)
	defer srv.Close()

	for name, frame := range map[string]any{
		"invalid token": map[string]string{"type": "auth", "token": "garbage"},
		"wrong frame":   map[string]string{"type": "hello"},
	} {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.WriteJSON(frame)
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, _, err = conn.ReadMessage()
		if !websocket.IsCloseError(err, 4401) {
			t.Errorf("%s: expected close code 4401, got %v", name, err)
		}
		_ = conn.Close()
	}

	// A revoked session cannot open a socket either.
	s := env.register("revoked")
	env.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": s.Refresh}).expect(t, http.StatusNoContent)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": s.Token})
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, 4401) {
		t.Errorf("revoked session: expected close code 4401, got %v", err)
	}
}

func TestDeletedAccountSocketIsClosed(t *testing.T) {
	env := newEnv(t)
	srv := httptest.NewServer(env.router)
	defer srv.Close()
	s := env.register("bye")
	w := env.dialWS(srv, s.Token)
	w.next("ready")
	env.do(http.MethodDelete, "/me", s.Token, map[string]string{"password": testPassword}).expect(t, http.StatusNoContent)
	_ = w.conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := w.conn.ReadMessage(); err == nil {
		t.Fatal("socket of a deleted account must be closed")
	}
}
