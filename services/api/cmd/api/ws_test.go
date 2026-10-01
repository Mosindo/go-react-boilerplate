package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestServer(e *testEnv) *httptest.Server { return httptest.NewServer(e.router) }

func (e *testEnv) wsTicket(t *testing.T, u testUser) string {
	t.Helper()
	resp := e.expect(t, e.do(t, http.MethodPost, "/ws/ticket", nil, u.Token), http.StatusOK, "ws ticket")
	var out struct {
		Ticket string `json:"ticket"`
	}
	resp.decode(t, &out)
	return out.Ticket
}

func wsURL(base, ticket string) string {
	return "ws" + strings.TrimPrefix(base, "http") + "/ws?ticket=" + url.QueryEscape(ticket)
}

func dialStatus(t *testing.T, base, ticket string) int {
	t.Helper()
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(base, ticket), nil)
	if err == nil {
		conn.Close()
		return http.StatusSwitchingProtocols
	}
	if resp == nil {
		t.Fatalf("dial failed without response: %v", err)
	}
	return resp.StatusCode
}

func dialWS(t *testing.T, base, ticket string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(base, ticket), nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	return conn
}

type wsEvent struct {
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// readEventRaw waits for the next event of the given type and returns its raw JSON.
func readEventRaw(t *testing.T, conn *websocket.Conn, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %q event: %v", want, err)
		}
		var head struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(payload, &head) == nil && head.Type == want {
			return string(payload)
		}
	}
}

func readEvent(t *testing.T, conn *websocket.Conn, want string) wsEvent {
	t.Helper()
	var ev wsEvent
	raw := readEventRaw(t, conn, want)
	// Data may hold non-string values for some events; only string maps are asserted on.
	_ = json.Unmarshal([]byte(raw), &ev)
	return ev
}
