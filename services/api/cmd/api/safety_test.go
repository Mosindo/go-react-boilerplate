package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestBlockingHidesBothSidesAndDissolvesMatch(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchPair()
	e.send(a, conv, "hello")

	e.expect(e.do("POST", "/blocks", map[string]any{"userId": a.ID}, a.Token), http.StatusBadRequest) // self
	e.expect(e.do("POST", "/blocks", map[string]any{"userId": "nope"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/blocks", map[string]any{"userId": b.ID}, a.Token), http.StatusNoContent)
	e.expect(e.do("POST", "/blocks", map[string]any{"userId": b.ID}, a.Token), http.StatusNoContent) // idempotent

	for _, pair := range [][2]*user{{a, b}, {b, a}} {
		viewer, other := pair[0], pair[1]
		e.expect(e.do("GET", "/profiles/"+other.ID, nil, viewer.Token), http.StatusNotFound)
		if ids := e.discoverIDs(viewer); contains(ids, other.ID) {
			t.Fatal("blocked user still discoverable")
		}
		if l := e.expect(e.do("GET", "/matches", nil, viewer.Token), http.StatusOK).m(t)["matches"].([]any); len(l) != 0 {
			t.Fatalf("match survived block: %v", l)
		}
		if len(e.conversations(viewer)) != 0 {
			t.Fatal("conversation survived block")
		}
		e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]any{"body": "x"}, viewer.Token), http.StatusNotFound)
		e.expect(e.do("POST", "/swipes", map[string]any{"userId": other.ID, "action": "like"}, viewer.Token), http.StatusNotFound)
	}

	// Only the blocker sees the block list; the blocked person is not told.
	list := e.expect(e.do("GET", "/blocks", nil, a.Token), http.StatusOK).m(t)["blocks"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["user"].(map[string]any)["id"] != b.ID {
		t.Fatalf("block list: %v", list)
	}
	if l := e.expect(e.do("GET", "/blocks", nil, b.Token), http.StatusOK).m(t)["blocks"].([]any); len(l) != 0 {
		t.Fatalf("blocked user can see who blocked them: %v", l)
	}

	// Unblocking restores visibility in discovery only for people never swiped on, never revives matches.
	e.expect(e.do("DELETE", "/blocks/"+b.ID, nil, a.Token), http.StatusNoContent)
	e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusOK)
	if l := e.expect(e.do("GET", "/matches", nil, a.Token), http.StatusOK).m(t)["matches"].([]any); len(l) != 0 {
		t.Fatal("unblocking must not revive the match")
	}
}

func TestBlockedUserCannotSeeLikeInInbox(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	e.swipe(b, a.ID, "like")
	e.expect(e.do("POST", "/blocks", map[string]any{"userId": b.ID}, a.Token), http.StatusNoContent)
	if l := e.expect(e.do("GET", "/likes/received", nil, a.Token), http.StatusOK).m(t)["profiles"].([]any); len(l) != 0 {
		t.Fatalf("a blocked user's like is still shown: %v", l)
	}
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()

	r := e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "  insultes  "}, a.Token), http.StatusCreated).m(t)
	if r["id"] == "" {
		t.Fatal("report id missing")
	}
	var details, status string
	_ = e.pool.QueryRow(t.Context(), `SELECT details, status FROM reports WHERE id = $1`, r["id"]).Scan(&details, &status)
	if details != "insultes" || status != "open" {
		t.Fatalf("stored report: %q %q", details, status)
	}

	e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "nonsense"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "spam"}, ""), http.StatusUnauthorized)

	// A report survives the reported account's deletion (evidence for moderators).
	e.expect(e.do("DELETE", "/me", map[string]any{"password": "correct horse battery"}, b.Token), http.StatusNoContent)
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE id = $1`, r["id"]).Scan(&n)
	if n != 1 {
		t.Fatal("report was deleted together with the reported account")
	}
}
