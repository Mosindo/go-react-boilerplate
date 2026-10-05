package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestBlockHidesEverythingBothWays(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	m := env.match(a, b)
	conv := "/conversations/" + m.ConversationID
	env.do(http.MethodPost, conv+"/messages", b.Token, map[string]string{"content": "salut"}).expect(t, http.StatusCreated)

	env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": a.ID}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": "22222222-2222-2222-2222-222222222222"}).expect(t, http.StatusNotFound)
	env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": b.ID}).expect(t, http.StatusNoContent)
	env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": b.ID}).expect(t, http.StatusNoContent) // idempotent

	for name, s := range map[string]session{"blocker": a, "blocked": b} {
		other := b
		if name == "blocked" {
			other = a
		}
		var list struct{ Matches []matchInfo }
		env.do(http.MethodGet, "/matches", s.Token, nil).json(t, &list)
		if len(list.Matches) != 0 {
			t.Errorf("%s: blocked pair must disappear from matches", name)
		}
		var convs conversationList
		env.do(http.MethodGet, "/conversations", s.Token, nil).json(t, &convs)
		if len(convs.Conversations) != 0 {
			t.Errorf("%s: blocked pair must disappear from conversations", name)
		}
		env.do(http.MethodGet, conv+"/messages", s.Token, nil).expect(t, http.StatusNotFound)
		env.do(http.MethodPost, conv+"/messages", s.Token, map[string]string{"content": "hello?"}).expect(t, http.StatusNotFound)
		env.do(http.MethodGet, "/profiles/"+other.ID, s.Token, nil).expect(t, http.StatusNotFound)
		if hasCard(env.discover(s), other.ID) {
			t.Errorf("%s: blocked pair must not be proposed", name)
		}
	}

	// A blocked user cannot like their way back in.
	c := env.register("carl")
	env.onboard(c, profileSpec{Name: "Carl", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng})
	env.do(http.MethodPost, "/blocks", c.Token, map[string]string{"userId": a.ID}).expect(t, http.StatusNoContent)
	env.swipe(a, c.ID, "like").expect(t, http.StatusNotFound)
	env.swipe(c, a.ID, "like").expect(t, http.StatusNotFound)

	// List and unblock.
	var blocks struct {
		Blocks []struct {
			User card `json:"user"`
		} `json:"blocks"`
	}
	env.do(http.MethodGet, "/blocks", a.Token, nil).expect(t, http.StatusOK).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].User.ID != b.ID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	env.do(http.MethodGet, "/blocks", b.Token, nil).json(t, &blocks)
	if len(blocks.Blocks) != 0 {
		t.Fatal("the blocked person must not learn they were blocked")
	}
	env.do(http.MethodDelete, "/blocks/"+b.ID, a.Token, nil).expect(t, http.StatusNoContent)
	env.do(http.MethodGet, conv+"/messages", a.Token, nil).expect(t, http.StatusOK)
	env.do(http.MethodGet, "/profiles/"+b.ID, a.Token, nil).expect(t, http.StatusOK)
}

func TestReports(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	report := func(body map[string]any) resp { return env.do(http.MethodPost, "/reports", a.Token, body) }

	report(map[string]any{"userId": a.ID, "reason": "spam"}).expect(t, http.StatusBadRequest)
	report(map[string]any{"userId": b.ID, "reason": "because"}).expect(t, http.StatusBadRequest)
	report(map[string]any{"userId": b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}).expect(t, http.StatusBadRequest)
	report(map[string]any{"userId": "33333333-3333-3333-3333-333333333333", "reason": "spam"}).expect(t, http.StatusNotFound)
	env.do(http.MethodPost, "/reports", "", map[string]any{"userId": b.ID, "reason": "spam"}).expect(t, http.StatusUnauthorized)

	report(map[string]any{"userId": b.ID, "reason": "harassment", "details": "  insultes  "}).expect(t, http.StatusCreated)
	report(map[string]any{"userId": b.ID, "reason": "harassment", "details": "again"}).expect(t, http.StatusCreated) // deduplicated, still 201

	var n int
	var status, details string
	_ = env.pool.QueryRow(t.Context(), `SELECT count(*), min(status), min(details) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n, &status, &details)
	if n != 1 || status != "open" || details != "insultes" {
		t.Fatalf("expected one open report with trimmed details, got n=%d status=%s details=%q", n, status, details)
	}

	// Report + block in one step.
	m := env.match(a, b)
	report(map[string]any{"userId": b.ID, "reason": "scam", "alsoBlock": true}).expect(t, http.StatusCreated)
	env.do(http.MethodGet, "/conversations/"+m.ConversationID+"/messages", b.Token, nil).expect(t, http.StatusNotFound)

	// Reports survive the deletion of the reported account (moderation history).
	env.do(http.MethodDelete, "/me", b.Token, map[string]string{"password": testPassword}).expect(t, http.StatusNoContent)
	_ = env.pool.QueryRow(t.Context(), `SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, a.ID).Scan(&n)
	if n != 2 {
		t.Fatalf("reports must be kept with a null reported user, got %d", n)
	}
}
