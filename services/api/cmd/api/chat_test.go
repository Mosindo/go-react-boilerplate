package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (e *env) send(u *user, conv, body string) map[string]any {
	e.t.Helper()
	return e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]any{"body": body}, u.Token), http.StatusCreated).m(e.t)
}

func (e *env) conversations(u *user) []map[string]any {
	e.t.Helper()
	raw := e.expect(e.do("GET", "/conversations", nil, u.Token), http.StatusOK).m(e.t)["conversations"].([]any)
	out := make([]map[string]any, len(raw))
	for i, c := range raw {
		out[i] = c.(map[string]any)
	}
	return out
}

func TestChatSendReceiveReadState(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchPair()

	// A new match already appears as an (empty) conversation for both.
	for _, u := range []*user{a, b} {
		cs := e.conversations(u)
		if len(cs) != 1 || cs[0]["lastMessage"] != nil || cs[0]["unreadCount"].(float64) != 0 {
			t.Fatalf("fresh conversation: %v", cs)
		}
	}

	m1 := e.send(a, conv, "  Salut Bob !  ")
	if m1["body"] != "Salut Bob !" || m1["senderId"] != a.ID || m1["readAt"] != nil || m1["createdAt"] == "" {
		t.Fatalf("unexpected message %v", m1)
	}
	e.send(a, conv, "Ça va ?")

	// B: unread badge and last message; A: nothing unread.
	cs := e.conversations(b)
	if cs[0]["unreadCount"].(float64) != 2 || cs[0]["lastMessage"].(map[string]any)["body"] != "Ça va ?" {
		t.Fatalf("recipient view: %v", cs[0])
	}
	if e.conversations(a)[0]["unreadCount"].(float64) != 0 {
		t.Fatal("sender must not have unread messages")
	}
	// One coalesced notification for two messages.
	n := e.expect(e.do("GET", "/notifications", nil, b.Token), http.StatusOK).m(t)
	msgNotifs := 0
	for _, x := range n["notifications"].([]any) {
		if x.(map[string]any)["type"] == "message" {
			msgNotifs++
		}
	}
	if msgNotifs != 1 {
		t.Fatalf("expected 1 coalesced message notification, got %d", msgNotifs)
	}

	// Messages are unread for A's view until B reads.
	msgs := e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, a.Token), http.StatusOK).m(t)["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["readAt"] != nil {
		t.Fatalf("messages should be unread: %v", msgs)
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/read", nil, b.Token), http.StatusNoContent)
	msgs = e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, a.Token), http.StatusOK).m(t)["messages"].([]any)
	if msgs[0].(map[string]any)["readAt"] == nil || msgs[1].(map[string]any)["readAt"] == nil {
		t.Fatalf("messages should now be read: %v", msgs)
	}
	if e.conversations(b)[0]["unreadCount"].(float64) != 0 {
		t.Fatal("unread count should be reset")
	}
	// Reading cleared the message notification.
	n = e.expect(e.do("GET", "/notifications", nil, b.Token), http.StatusOK).m(t)
	if n["unreadCount"].(float64) != 1 { // only the match one remains
		t.Fatalf("message notification should be read: %v", n)
	}
	e.expect(e.do("POST", "/notifications/read-all", nil, b.Token), http.StatusNoContent)
	if e.expect(e.do("GET", "/notifications", nil, b.Token), http.StatusOK).m(t)["unreadCount"].(float64) != 0 {
		t.Fatal("read-all failed")
	}
	// A's reply is unread for B and ordering is chronological.
	e.send(b, conv, "Très bien, et toi ?")
	msgs = e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, b.Token), http.StatusOK).m(t)["messages"].([]any)
	if len(msgs) != 3 || msgs[2].(map[string]any)["body"] != "Très bien, et toi ?" || msgs[2].(map[string]any)["readAt"] != nil {
		t.Fatalf("chronology/read state: %v", msgs)
	}
}

func TestChatPermissions(t *testing.T) {
	e := newEnv(t)
	a, _, conv := e.matchPair()
	_, _, otherConv := e.matchPair()
	intruder := e.register("")

	for _, tc := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"DELETE", "/conversations/" + conv},
	} {
		var body any
		if tc.method == "POST" && strings.HasSuffix(tc.path, "messages") {
			body = map[string]any{"body": "intrusion"}
		}
		// Strangers, and members of other conversations, learn nothing: always 404.
		e.expect(e.do(tc.method, tc.path, body, intruder.Token), http.StatusNotFound)
		e.expect(e.do(tc.method, tc.path, body, ""), http.StatusUnauthorized)
	}
	// A real member of another conversation cannot read this one either.
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, a.Token), http.StatusOK)
	other := e.expect(e.do("GET", "/conversations", nil, intruder.Token), http.StatusOK).m(t)["conversations"].([]any)
	if len(other) != 0 {
		t.Fatalf("intruder sees conversations: %v", other)
	}
	_ = otherConv
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", nil, a.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/conversations/00000000-0000-0000-0000-000000000000/messages", nil, a.Token), http.StatusNotFound)
}

func TestChatMessageValidation(t *testing.T) {
	e := newEnv(t)
	a, _, conv := e.matchPair()
	for name, body := range map[string]string{"empty": "", "blank": "  \n\t ", "too long": strings.Repeat("é", 2001), "control only": "\x00\x07"} {
		if r := e.do("POST", "/conversations/"+conv+"/messages", map[string]any{"body": body}, a.Token); r.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, r.Status)
		}
	}
	ok := e.send(a, conv, strings.Repeat("é", 2000))
	if len([]rune(ok["body"].(string))) != 2000 {
		t.Fatal("2000 characters must be accepted")
	}
	// Control characters are stripped, line breaks kept; markup is stored verbatim (clients render text only).
	m := e.send(a, conv, "a\x00b\nc<script>")
	if m["body"] != "ab\nc<script>" {
		t.Fatalf("unexpected sanitisation: %q", m["body"])
	}
}

func TestChatPaginationAndLocalDeletion(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchPair()
	for i := 1; i <= 12; i++ {
		e.send(a, conv, fmt.Sprintf("m%02d", i))
	}
	page1 := e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=5", nil, b.Token), http.StatusOK).m(t)
	msgs := page1["messages"].([]any)
	if len(msgs) != 5 || page1["hasMore"] != true || msgs[0].(map[string]any)["body"] != "m08" || msgs[4].(map[string]any)["body"] != "m12" {
		t.Fatalf("page 1: %v", page1)
	}
	oldest := msgs[0].(map[string]any)["id"].(string)
	page2 := e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=5&before="+oldest, nil, b.Token), http.StatusOK).m(t)
	msgs2 := page2["messages"].([]any)
	if len(msgs2) != 5 || msgs2[0].(map[string]any)["body"] != "m03" || msgs2[4].(map[string]any)["body"] != "m07" {
		t.Fatalf("page 2: %v", page2)
	}
	page3 := e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=5&before="+msgs2[0].(map[string]any)["id"].(string), nil, b.Token), http.StatusOK).m(t)
	if len(page3["messages"].([]any)) != 2 || page3["hasMore"] != false {
		t.Fatalf("page 3: %v", page3)
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=51", nil, b.Token), http.StatusBadRequest)

	// B deletes the conversation locally: it disappears for B only.
	e.expect(e.do("DELETE", "/conversations/"+conv, nil, b.Token), http.StatusNoContent)
	if len(e.conversations(b)) != 0 {
		t.Fatal("deleted conversation is still listed")
	}
	if len(e.conversations(a)) != 1 {
		t.Fatal("deleting locally must not affect the other member")
	}
	if got := e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, b.Token), http.StatusOK).m(t)["messages"].([]any); len(got) != 0 {
		t.Fatalf("history should be cleared for B: %v", got)
	}
	if got := e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, a.Token), http.StatusOK).m(t)["messages"].([]any); len(got) != 12 {
		t.Fatalf("A keeps the history: %d", len(got))
	}
	// A new message brings it back with only the new content.
	e.send(a, conv, "tu es là ?")
	cs := e.conversations(b)
	if len(cs) != 1 || cs[0]["unreadCount"].(float64) != 1 {
		t.Fatalf("conversation should resurface: %v", cs)
	}
	if got := e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, b.Token), http.StatusOK).m(t)["messages"].([]any); len(got) != 1 {
		t.Fatalf("only the new message expected: %v", got)
	}
}
