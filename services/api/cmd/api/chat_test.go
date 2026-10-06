package main

import (
	"net/http"
	"strings"
	"testing"
)

type chatFixture struct {
	a, b, outsider user
	matchID        string
}

func newChatFixture(t *testing.T, e *env) chatFixture {
	t.Helper()
	a := e.newPerson(persona{Name: "Ada", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.newPerson(persona{Name: "Bruno", Gender: "man", InterestedIn: []string{"woman"}})
	outsider := e.newPerson(persona{Name: "Oscar", Gender: "man", InterestedIn: []string{"woman"}})
	return chatFixture{a: a, b: b, outsider: outsider, matchID: e.makeMatch(a, b)}
}

func (e *env) send(u user, matchID, body string, want int) response {
	e.t.Helper()
	return e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/messages", u.Token, map[string]string{"body": body}), want)
}

func TestChatRequiresMatchAndMembership(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)

	// An outsider cannot read, write, mark read, hide or unmatch someone else's conversation.
	f.sendAs(e, f.a, "secret plans")
	path := "/matches/" + f.matchID
	e.expect(e.do(http.MethodGet, path+"/messages", f.outsider.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, path+"/messages", f.outsider.Token, map[string]string{"body": "hi"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, path+"/read", f.outsider.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, path+"/hide", f.outsider.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodDelete, path, f.outsider.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, path+"/messages", "", nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/matches/not-a-uuid/messages", f.a.Token, nil), http.StatusNotFound)

	// The outsider's list does not leak the conversation either.
	list := e.expect(e.do(http.MethodGet, "/matches", f.outsider.Token, nil), http.StatusOK).json(t)
	if len(list["matches"].([]any)) != 0 {
		t.Fatalf("outsider must see no matches: %v", list)
	}

	// Two users who merely liked one-sidedly have no conversation.
	c := e.newPerson(persona{Name: "Cleo", Gender: "woman", InterestedIn: []string{"man"}})
	e.swipe(c, f.outsider, "like", http.StatusCreated)
	if n := countRows(t, e.pool, `SELECT count(*) FROM conversations`); n != 1 {
		t.Fatalf("only the matched pair may have a conversation, got %d", n)
	}
}

func (f chatFixture) sendAs(e *env, from user, body string) {
	e.t.Helper()
	e.send(from, f.matchID, body, http.StatusCreated)
}

func TestChatMessagesReadStateAndPagination(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)

	for i := 0; i < 5; i++ {
		f.sendAs(e, f.a, "message "+string(rune('A'+i)))
	}

	// Unread counts are per user.
	listFor := func(u user) map[string]any {
		return e.expect(e.do(http.MethodGet, "/matches", u.Token, nil), http.StatusOK).json(t)
	}
	if got := listFor(f.b)["unreadTotal"].(float64); got != 5 {
		t.Fatalf("B unread: got %v want 5", got)
	}
	if got := listFor(f.a)["unreadTotal"].(float64); got != 0 {
		t.Fatalf("A unread: got %v want 0 (own messages never count)", got)
	}
	e.expect(e.do(http.MethodPost, "/matches/"+f.matchID+"/read", f.b.Token, nil), http.StatusNoContent)
	if got := listFor(f.b)["unreadTotal"].(float64); got != 0 {
		t.Fatalf("after read B unread: got %v want 0", got)
	}
	f.sendAs(e, f.b, "reply")
	if got := listFor(f.a)["unreadTotal"].(float64); got != 1 {
		t.Fatalf("A unread after reply: got %v want 1", got)
	}

	// Keyset pagination, oldest-first inside a page.
	page1 := e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages?limit=4", f.a.Token, nil), http.StatusOK).json(t)
	msgs1 := page1["messages"].([]any)
	if len(msgs1) != 4 || page1["hasMore"] != true || msgs1[3].(map[string]any)["body"] != "reply" {
		t.Fatalf("unexpected first page: %v", page1)
	}
	oldest := msgs1[0].(map[string]any)["id"].(string)
	page2 := e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages?limit=4&before="+oldest, f.a.Token, nil), http.StatusOK).json(t)
	msgs2 := page2["messages"].([]any)
	if len(msgs2) != 2 || page2["hasMore"] != false || msgs2[0].(map[string]any)["body"] != "message A" {
		t.Fatalf("unexpected second page: %v", page2)
	}
	e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages?before=garbage", f.a.Token, nil), http.StatusBadRequest)
}

func TestChatMessageValidation(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)

	e.send(f.a, f.matchID, "   ", http.StatusUnprocessableEntity)
	e.send(f.a, f.matchID, "\x00\x07", http.StatusUnprocessableEntity)
	e.send(f.a, f.matchID, strings.Repeat("é", 2001), http.StatusUnprocessableEntity)
	e.expect(e.do(http.MethodPost, "/matches/"+f.matchID+"/messages", f.a.Token, map[string]string{}), http.StatusBadRequest)

	// Stored text is trimmed and stripped of control characters; markup is data, not code.
	out := e.send(f.a, f.matchID, "  <b>hi</b>\x07 there  ", http.StatusCreated).json(t)
	if out["body"] != "<b>hi</b> there" {
		t.Fatalf("unexpected normalized body %q", out["body"])
	}
}

func TestChatHideUnmatchAndNewMessageResurfaces(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)
	f.sendAs(e, f.a, "one")

	count := func(u user) int {
		return len(e.expect(e.do(http.MethodGet, "/matches", u.Token, nil), http.StatusOK).json(t)["matches"].([]any))
	}

	// "Local delete": hidden for B only; A still sees it.
	e.expect(e.do(http.MethodPost, "/matches/"+f.matchID+"/hide", f.b.Token, nil), http.StatusNoContent)
	if count(f.b) != 0 || count(f.a) != 1 {
		t.Fatalf("hide must be local: b=%d a=%d", count(f.b), count(f.a))
	}
	// A new incoming message brings it back for B.
	f.sendAs(e, f.a, "two")
	if count(f.b) != 1 {
		t.Fatal("a new message must resurface a hidden conversation")
	}
	// A conversation is only hidden for the one who hid it: nothing was deleted.
	if n := countRows(t, e.pool, `SELECT count(*) FROM messages`); n != 2 {
		t.Fatalf("hiding must not delete messages, got %d", n)
	}

	// Unmatch ends it for both and closes the thread.
	e.expect(e.do(http.MethodDelete, "/matches/"+f.matchID, f.b.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages", f.a.Token, nil), http.StatusNotFound)
	e.send(f.a, f.matchID, "still there?", http.StatusNotFound)
	if count(f.a) != 0 || count(f.b) != 0 {
		t.Fatal("an unmatched conversation must disappear for both")
	}
	// They cannot be re-matched or shown to each other again.
	if contains(e.discoverIDs(f.a), f.b.ID) {
		t.Fatal("an unmatched user was already swiped: it must not come back")
	}
}

func TestNotificationsLifecycle(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)

	// Several messages produce a single unread "new_message" notification.
	for i := 0; i < 3; i++ {
		f.sendAs(e, f.a, "ping")
	}
	n := e.expect(e.do(http.MethodGet, "/notifications", f.b.Token, nil), http.StatusOK).json(t)
	items := n["notifications"].([]any)
	if len(items) != 2 || n["unreadCount"].(float64) != 2 { // new_match + one new_message
		t.Fatalf("expected match + one deduped message notification, got %v", n)
	}
	first := items[0].(map[string]any)
	if first["type"] != "new_message" || first["data"].(map[string]any)["matchId"] != f.matchID {
		t.Fatalf("unexpected newest notification %v", first)
	}

	// Reading the conversation clears its message notification.
	e.expect(e.do(http.MethodPost, "/matches/"+f.matchID+"/read", f.b.Token, nil), http.StatusNoContent)
	n = e.expect(e.do(http.MethodGet, "/notifications?unread=true", f.b.Token, nil), http.StatusOK).json(t)
	if len(n["notifications"].([]any)) != 1 {
		t.Fatalf("only the match notification should remain unread: %v", n)
	}

	// Mark one / all read; nobody can touch someone else's notification.
	matchNotif := n["notifications"].([]any)[0].(map[string]any)["id"].(string)
	e.expect(e.do(http.MethodPost, "/notifications/"+matchNotif+"/read", f.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/notifications/"+matchNotif+"/read", f.b.Token, nil), http.StatusOK)
	e.expect(e.do(http.MethodPost, "/notifications/read-all", f.a.Token, nil), http.StatusNoContent)
	if got := e.expect(e.do(http.MethodGet, "/notifications", f.a.Token, nil), http.StatusOK).json(t)["unreadCount"].(float64); got != 0 {
		t.Fatalf("read-all must clear unread, got %v", got)
	}
	// Clients cannot forge notifications.
	e.expect(e.do(http.MethodPost, "/notifications", f.a.Token, map[string]string{"type": "x", "title": "x", "body": "x"}), http.StatusNotFound)
}
