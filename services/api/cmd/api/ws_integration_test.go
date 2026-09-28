package main

import (
	"encoding/json"
	"net/http"
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

func dialWS(t *testing.T, e *env, token string) *wsClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(e), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	c := &wsClient{t: t, conn: conn}
	ev := c.next(3 * time.Second)
	if ev.Type != "ready" {
		t.Fatalf("expected ready, got %+v", ev)
	}
	return c
}

func (c *wsClient) next(timeout time.Duration) wsEvent {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	var ev wsEvent
	if err := c.conn.ReadJSON(&ev); err != nil {
		c.t.Fatalf("ws read: %v", err)
	}
	return ev
}

// expect reads until an event of the wanted type arrives (skipping others).
func (c *wsClient) expect(typ string) wsEvent {
	c.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		ev := c.next(time.Until(deadline))
		if ev.Type == typ {
			return ev
		}
	}
	c.t.Fatalf("no %s event", typ)
	return wsEvent{}
}

func (c *wsClient) silent(d time.Duration) {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(d))
	var ev wsEvent
	if err := c.conn.ReadJSON(&ev); err == nil {
		c.t.Fatalf("unexpected event %+v", ev)
	}
}

func closeCode(t *testing.T, conn *websocket.Conn, wait time.Duration) int {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(wait))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				return ce.Code
			}
			t.Fatalf("expected close frame, got %v", err)
		}
	}
}

func TestWebSocketEvents(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Name: "Ann", Gender: "woman", Lat: 10, Lon: 10})
	b := e.person(spec{Name: "Ben", Gender: "man", Lat: 10, Lon: 10})
	outsider := e.person(spec{Name: "Out", Gender: "man", Lat: 10, Lon: 10})

	wa := dialWS(t, e, a.Access)
	wb := dialWS(t, e, b.Access)
	wo := dialWS(t, e, outsider.Access)

	// ping/pong
	if err := wa.conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal(err)
	}
	if ev := wa.expect("pong"); ev.Type != "pong" {
		t.Fatal("no pong")
	}

	// match -> match.new + notification.new for both sides
	e.must(a.swipe(b, "like"), 200)
	e.must(b.swipe(a, "like"), 200)
	var conv, matchID string
	for _, tc := range []struct {
		c     *wsClient
		other *user
	}{{wa, b}, {wb, a}} {
		ev := tc.c.expect("match.new")
		var m map[string]any
		_ = json.Unmarshal(ev.Data, &m)
		if m["user"].(map[string]any)["userId"] != tc.other.ID || m["matchId"] == nil {
			t.Fatalf("match.new payload: %s", ev.Data)
		}
		conv, matchID = m["conversationId"].(string), m["matchId"].(string)
		nev := tc.c.expect("notification.new")
		var n map[string]any
		_ = json.Unmarshal(nev.Data, &n)
		if n["type"] != "match" || n["id"] == nil {
			t.Fatalf("notification.new payload: %s", nev.Data)
		}
	}

	// message.new for both participants, notification.new for the recipient
	sent := e.must(b.call("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hello over ws"}), 201).json()
	ev := wa.expect("message.new")
	var msg map[string]any
	_ = json.Unmarshal(ev.Data, &msg)
	if msg["id"] != sent["id"] || msg["body"] != "hello over ws" || msg["senderId"] != b.ID || msg["conversationId"] != conv {
		t.Fatalf("message.new payload: %s", ev.Data)
	}
	wb.expect("message.new") // echo to the sender's devices
	nev := wa.expect("notification.new")
	var n map[string]any
	_ = json.Unmarshal(nev.Data, &n)
	if n["type"] != "message" {
		t.Fatalf("message notification event: %s", nev.Data)
	}

	// messages.read goes to the other participant
	e.must(a.call("POST", "/conversations/"+conv+"/read", nil), 204)
	rev := wb.expect("messages.read")
	var rd map[string]any
	_ = json.Unmarshal(rev.Data, &rd)
	if rd["conversationId"] != conv || rd["readerId"] != a.ID {
		t.Fatalf("messages.read payload: %s", rev.Data)
	}

	// unmatch -> match.removed to both
	e.must(a.call("DELETE", "/matches/"+matchID, nil), 204)
	for _, c := range []*wsClient{wa, wb} {
		rm := c.expect("match.removed")
		var m map[string]any
		_ = json.Unmarshal(rm.Data, &m)
		if m["matchId"] != matchID || m["conversationId"] != conv {
			t.Fatalf("match.removed payload: %s", rm.Data)
		}
	}
	// a user who is not part of any of this never received anything private
	wo.silent(300 * time.Millisecond)
}

func TestWebSocketAuth(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Gender: "woman", Lat: 10, Lon: 10})

	dial := func(header http.Header) (*websocket.Conn, *http.Response, error) {
		return websocket.DefaultDialer.Dial(wsURL(e), header)
	}

	// invalid token
	conn, _, err := dial(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "garbage"})
	if code := closeCode(t, conn, 3*time.Second); code != 4401 {
		t.Errorf("garbage token close code %d", code)
	}

	// first message must be auth
	conn, _, _ = dial(nil)
	_ = conn.WriteJSON(map[string]string{"type": "ping"})
	if code := closeCode(t, conn, 3*time.Second); code != 4401 {
		t.Errorf("non-auth first message close code %d", code)
	}

	// revoked session
	conn, _, _ = dial(nil)
	e.must(e.call("POST", "/auth/logout", "", map[string]string{"refreshToken": a.Refresh}), 204)
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": a.Access})
	if code := closeCode(t, conn, 3*time.Second); code != 4401 {
		t.Errorf("revoked session close code %d", code)
	}

	// foreign browser origin is refused at the handshake
	_, res, err := dial(http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		t.Fatal("cross-origin handshake must fail")
	}
	if res == nil || res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for bad origin, got %v", res)
	}

	// oversized frame closes the connection (read limit)
	b := e.person(spec{Gender: "man", Lat: 10, Lon: 10})
	wb := dialWS(t, e, b.Access)
	big := make([]byte, 64*1024)
	for i := range big {
		big[i] = 'a'
	}
	_ = wb.conn.WriteMessage(websocket.TextMessage, big)
	if code := closeCode(t, wb.conn, 3*time.Second); code == 0 {
		t.Error("expected close")
	}
}

func TestWebSocketAuthTimeout(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	start := time.Now()
	code := closeCode(t, conn, 8*time.Second)
	elapsed := time.Since(start)
	if code != 4401 {
		t.Errorf("silent client should be closed with 4401, got %d", code)
	}
	if elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Errorf("auth window should be ~5s, closed after %v", elapsed)
	}
}

func TestWebSocketConnectionCap(t *testing.T) {
	e := newEnv(t)
	a := e.person(spec{Gender: "woman", Lat: 10, Lon: 10})
	for i := 0; i < 5; i++ {
		dialWS(t, e, a.Access)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(e), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": a.Access})
	if code := closeCode(t, conn, 3*time.Second); code != 4429 {
		t.Errorf("6th connection should be refused with 4429, got %d", code)
	}
}
