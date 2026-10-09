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
	conn   *websocket.Conn
	events chan map[string]any
}

func (e *env) connect(srv *httptest.Server, u *user) *wsClient {
	e.t.Helper()
	ticket := e.expect(e.do("POST", "/ws/ticket", nil, u.Token), http.StatusOK).m(e.t)["ticket"].(string)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket=" + ticket
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		e.t.Fatalf("dial: %v", err)
	}
	c := &wsClient{conn: conn, events: make(chan map[string]any, 32)}
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				close(c.events)
				return
			}
			var ev map[string]any
			_ = json.Unmarshal(data, &ev)
			c.events <- ev
		}
	}()
	e.t.Cleanup(func() { _ = conn.Close() })
	return c
}

func (c *wsClient) next(t *testing.T, eventType string) map[string]any {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-c.events:
			if !ok {
				t.Fatalf("connection closed while waiting for %s", eventType)
			}
			if ev["type"] == eventType {
				return ev["data"].(map[string]any)
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", eventType)
		}
	}
}

func TestRealtimeMatchMessageAndReadEvents(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a, b, _ := e.pair()
	wa, wb := e.connect(srv, a), e.connect(srv, b)

	e.swipe(a, b.ID, "like")
	res := e.swipe(b, a.ID, "like")
	conv := res["match"].(map[string]any)["conversationId"].(string)

	for name, w := range map[string]*wsClient{"a": wa, "b": wb} {
		m := w.next(t, "match.new")
		if m["conversationId"] != conv || m["user"].(map[string]any)["firstName"] == "" {
			t.Fatalf("%s: bad match.new %v", name, m)
		}
		w.next(t, "notification.new")
	}

	e.send(a, conv, "temps réel !")
	msg := wb.next(t, "message.new")
	if msg["body"] != "temps réel !" || msg["senderId"] != a.ID {
		t.Fatalf("bad message.new %v", msg)
	}
	wa.next(t, "message.new") // echo to the sender's other devices
	wb.next(t, "notification.new")

	e.expect(e.do("POST", "/conversations/"+conv+"/read", nil, b.Token), http.StatusNoContent)
	read := wa.next(t, "message.read")
	if read["conversationId"] != conv || read["readerId"] != b.ID {
		t.Fatalf("bad message.read %v", read)
	}

	// Unmatching tells both sides.
	matches := e.expect(e.do("GET", "/matches", nil, a.Token), http.StatusOK).m(t)["matches"].([]any)
	e.expect(e.do("DELETE", "/matches/"+matches[0].(map[string]any)["id"].(string), nil, a.Token), http.StatusNoContent)
	wa.next(t, "match.removed")
	wb.next(t, "match.removed")
}

func TestRealtimeTicketsAreSingleUseAndAuthenticated(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	u := e.register("")

	e.expect(e.do("POST", "/ws/ticket", nil, ""), http.StatusUnauthorized)
	ticket := e.expect(e.do("POST", "/ws/ticket", nil, u.Token), http.StatusOK).m(t)["ticket"].(string)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket=" + ticket

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("first use must succeed: %v", err)
	}
	_ = conn.Close()
	if _, resp, err := websocket.DefaultDialer.Dial(url, nil); err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("a ticket must work only once")
	}
	if _, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("missing ticket must be refused")
	}
	// An access token is not accepted as a ticket.
	if _, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+u.Token, nil); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("JWT must not be usable as ws ticket")
	}
}

func TestRealtimeOnlyTargetsParticipants(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a, _, conv := e.matchPair()
	outsider := e.register("")
	wo := e.connect(srv, outsider)
	e.send(a, conv, "private")
	select {
	case ev := <-wo.events:
		t.Fatalf("outsider received %v", ev)
	case <-time.After(300 * time.Millisecond):
	}
}
