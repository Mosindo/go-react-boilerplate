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

func TestRealtimeDeliversMessagesToMatchesOnly(t *testing.T) {
	env := newTestEnv(t)
	server := httptest.NewServer(env.router)
	defer server.Close()

	ana := env.onboard("rt_ana", "Ana", "woman", []string{"man"}, 48.11, -1.68)
	leo := env.onboard("rt_leo", "Léo", "man", []string{"woman"}, 48.11, -1.68)

	connect := func(token string) *websocket.Conn {
		t.Helper()
		resp := env.do(http.MethodPost, "/realtime/ticket", nil, token)
		env.expect(resp, http.StatusOK, "ticket")
		var ticket struct {
			Ticket string `json:"ticket"`
		}
		resp.decode(t, &ticket)
		url := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime/ws?ticket=" + ticket.Ticket
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		// Reusing a ticket must fail.
		if _, res, err := websocket.DefaultDialer.Dial(url, nil); err == nil || res.StatusCode != http.StatusUnauthorized {
			t.Fatal("tickets must be single use")
		}
		return conn
	}

	readEvent := func(conn *websocket.Conn, want string) map[string]any {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("waiting for %s: %v", want, err)
			}
			var event struct {
				Type string         `json:"type"`
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(raw, &event); err != nil {
				t.Fatal(err)
			}
			if event.Type == want {
				return event.Data
			}
		}
	}

	leoConn := connect(leo.Token)
	defer leoConn.Close()
	readEvent(leoConn, "connection.ready")

	env.expect(env.swipe(leo.Token, ana.UserID, "like"), http.StatusOK, "like")
	env.expect(env.swipe(ana.Token, leo.UserID, "like"), http.StatusCreated, "match")
	if data := readEvent(leoConn, "match.created"); !strings.Contains(toJSON(data), ana.UserID) {
		t.Fatalf("unexpected match event %v", data)
	}

	env.expect(env.do(http.MethodPost, "/chats/"+leo.UserID+"/messages", map[string]string{"content": "Hello en direct"}, ana.Token), http.StatusCreated, "send")
	if data := readEvent(leoConn, "message.created"); data["content"] != "Hello en direct" {
		t.Fatalf("unexpected message event %v", data)
	}
	readEvent(leoConn, "notification.created")

	env.expect(env.do(http.MethodPost, "/chats/"+leo.UserID+"/read", nil, ana.Token), http.StatusOK, "read")
	readEvent(leoConn, "conversation.read")

	env.expect(env.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": leo.RefreshToken}, ""), http.StatusNoContent, "logout")
	resp := env.do(http.MethodPost, "/realtime/ticket", nil, leo.Token)
	env.expect(resp, http.StatusUnauthorized, "ticket after logout")
}

func TestRealtimeRejectsInvalidTicket(t *testing.T) {
	env := newTestEnv(t)
	server := httptest.NewServer(env.router)
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime/ws?ticket=forged"
	if _, res, err := websocket.DefaultDialer.Dial(url, nil); err == nil || res.StatusCode != http.StatusUnauthorized {
		t.Fatal("forged tickets must be rejected")
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
