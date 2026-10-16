package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// The critical scenario: sign up -> profile -> discover -> like -> match -> conversation.
func TestDatingJourney(t *testing.T) {
	env := newTestEnv(t)

	alice := env.register("alice")
	bob := env.register("bob")
	carol := env.register("carol") // outsider, never matched

	// Before onboarding nobody can use discovery.
	if res := env.do(http.MethodGet, "/discover", alice.Token, nil, alice.addr); res.Status != http.StatusConflict {
		t.Fatalf("discover before onboarding should be 409, got %d", res.Status)
	}
	me := env.do(http.MethodGet, "/me/profile", alice.Token, nil, alice.addr).Map(t)
	if me["isComplete"] != false {
		t.Fatalf("fresh account must be incomplete: %v", me)
	}

	env.onboard(alice, profileSpec{Name: "Alice", BirthDate: "1995-05-05", Gender: "woman", InterestedIn: []string{"man"}, Interests: []string{"music", "travel", "cinema"}})
	env.onboard(bob, profileSpec{Name: "Bob", BirthDate: "1992-03-14", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"music", "cooking"}, Lat: 48.90, Lon: 2.40})
	env.onboard(carol, profileSpec{Name: "Carol", BirthDate: "1990-01-01", Gender: "woman", InterestedIn: []string{"woman"}})

	me = env.do(http.MethodGet, "/me/profile", alice.Token, nil, alice.addr).Map(t)
	if me["isComplete"] != true {
		t.Fatalf("onboarded account must be complete: %v", me)
	}

	// Discovery respects mutual preferences.
	if ids := env.discoverIDs(alice); len(ids) != 1 || ids[0] != bob.ID {
		t.Fatalf("alice should see only bob, got %v", ids)
	}
	if ids := env.discoverIDs(bob); len(ids) != 1 || ids[0] != alice.ID {
		t.Fatalf("bob should see only alice, got %v", ids)
	}
	if ids := env.discoverIDs(carol); len(ids) != 0 {
		t.Fatalf("carol should see nobody, got %v", ids)
	}

	// A card never leaks private fields.
	raw := string(env.do(http.MethodGet, "/discover", alice.Token, nil, alice.addr).Body)
	for _, forbidden := range []string{"email", "birthDate", "latitude", "longitude", bob.Email, "48.9"} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("discover payload leaks %q: %s", forbidden, raw)
		}
	}
	card := env.do(http.MethodGet, "/profiles/"+bob.ID, alice.Token, nil, alice.addr).Map(t)
	if card["firstName"] != "Bob" || card["age"] == nil || card["distanceKm"] == nil {
		t.Fatalf("unexpected profile card: %v", card)
	}

	// Bob has not liked yet; Alice likes first: no match.
	if res := env.swipe(alice, bob, "like"); res["matched"] != false {
		t.Fatalf("one-sided like must not match: %v", res)
	}
	if res := env.swipe(alice, bob, "like"); res["matched"] != false {
		t.Fatalf("repeating a like stays a no-op: %v", res)
	}
	if ids := env.discoverIDs(alice); contains(ids, bob.ID) {
		t.Fatal("a swiped profile must not come back")
	}

	// Bob sees Alice in "likes received" and discovery.
	likes := env.do(http.MethodGet, "/likes/received", bob.Token, nil, bob.addr).Map(t)
	if likes["total"] != float64(1) {
		t.Fatalf("bob should have 1 pending like: %v", likes)
	}

	// Reciprocal like -> match, created once.
	res := env.swipe(bob, alice, "like")
	if res["matched"] != true || res["matchId"] == "" {
		t.Fatalf("reciprocal like must match: %v", res)
	}
	matchID := res["matchId"].(string)
	if again := env.swipe(alice, bob, "like"); again["matchId"] != matchID {
		t.Fatalf("match must be stable: %v vs %s", again, matchID)
	}
	if again := env.swipe(bob, alice, "like"); again["matchId"] != matchID {
		t.Fatalf("match must be stable on repeat: %v vs %s", again, matchID)
	}
	var matchCount int
	if err := env.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches`).Scan(&matchCount); err != nil || matchCount != 1 {
		t.Fatalf("exactly one match row expected, got %d (%v)", matchCount, err)
	}

	// Both get exactly one match notification.
	for _, u := range []*user{alice, bob} {
		n := env.do(http.MethodGet, "/notifications", u.Token, nil, u.addr).Map(t)
		list := n["notifications"].([]any)
		if len(list) != 1 || list[0].(map[string]any)["type"] != "match" || n["unread"] != float64(1) {
			t.Fatalf("%s should have one unread match notification: %v", u.Email, n)
		}
	}

	// Chat: only the two participants can read or write.
	msgPath := "/conversations/" + matchID + "/messages"
	if r := env.do(http.MethodGet, msgPath, carol.Token, nil, carol.addr); r.Status != http.StatusNotFound {
		t.Fatalf("outsider reading messages: got %d, want 404", r.Status)
	}
	if r := env.do(http.MethodPost, msgPath, carol.Token, map[string]string{"body": "hi"}, carol.addr); r.Status != http.StatusNotFound {
		t.Fatalf("outsider sending: got %d, want 404", r.Status)
	}
	if r := env.do(http.MethodPost, msgPath, alice.Token, map[string]string{"body": "   "}, alice.addr); r.Status != http.StatusBadRequest {
		t.Fatalf("blank message: got %d", r.Status)
	}
	if r := env.do(http.MethodPost, msgPath, alice.Token, map[string]string{"body": strings.Repeat("a", 2001)}, alice.addr); r.Status != http.StatusBadRequest {
		t.Fatalf("over-long message: got %d", r.Status)
	}

	// Bob listens on the WebSocket while Alice writes.
	srv := httptest.NewServer(env.router)
	defer srv.Close()
	ticket := env.do(http.MethodPost, "/realtime/ticket", bob.Token, nil, bob.addr).Map(t)["ticket"].(string)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime?ticket=" + url.QueryEscape(ticket)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer conn.Close()
	if _, _, err := websocket.DefaultDialer.Dial(wsURL, nil); err == nil {
		t.Fatal("a ticket must be single-use")
	}

	sent := env.do(http.MethodPost, msgPath, alice.Token, map[string]string{"body": "  Salut Bob !  "}, alice.addr)
	if sent.Status != http.StatusCreated {
		t.Fatalf("send: %d %s", sent.Status, sent.Body)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event struct {
		Type string `json:"type"`
		Data struct {
			Body string `json:"body"`
		} `json:"data"`
	}
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("expected a realtime event: %v", err)
	}
	if event.Type != "message.new" || event.Data.Body != "Salut Bob !" {
		t.Fatalf("unexpected realtime event: %+v", event)
	}

	// Unread state, reading, and read receipts.
	convs := env.do(http.MethodGet, "/conversations", bob.Token, nil, bob.addr).Map(t)["conversations"].([]any)
	if len(convs) != 1 || convs[0].(map[string]any)["unreadCount"] != float64(1) {
		t.Fatalf("bob should see 1 unread: %v", convs)
	}
	if unread := env.do(http.MethodGet, "/conversations/unread-count", bob.Token, nil, bob.addr).Map(t)["unread"]; unread != float64(1) {
		t.Fatalf("unread total: %v", unread)
	}
	msgs := env.do(http.MethodGet, msgPath, bob.Token, nil, bob.addr).Map(t)["messages"].([]any)
	if len(msgs) != 1 || msgs[0].(map[string]any)["readAt"] != nil {
		t.Fatalf("message should start unread: %v", msgs)
	}
	if r := env.do(http.MethodPost, "/conversations/"+matchID+"/read", bob.Token, nil, bob.addr); r.Status != http.StatusNoContent {
		t.Fatalf("mark read: %d", r.Status)
	}
	msgs = env.do(http.MethodGet, msgPath, alice.Token, nil, alice.addr).Map(t)["messages"].([]any)
	if msgs[0].(map[string]any)["readAt"] == nil {
		t.Fatal("alice should see the read receipt")
	}
	if unread := env.do(http.MethodGet, "/conversations/unread-count", bob.Token, nil, bob.addr).Map(t)["unread"]; unread != float64(0) {
		t.Fatalf("unread total after reading: %v", unread)
	}

	// Message pagination: newest page first, older pages via the cursor.
	for i := 0; i < 5; i++ {
		env.do(http.MethodPost, msgPath, bob.Token, map[string]string{"body": "msg " + string(rune('A'+i))}, bob.addr)
	}
	page := env.do(http.MethodGet, msgPath+"?limit=3", alice.Token, nil, alice.addr).Map(t)
	got := page["messages"].([]any)
	if len(got) != 3 || page["hasMore"] != true || got[2].(map[string]any)["body"] != "msg E" || got[0].(map[string]any)["body"] != "msg C" {
		t.Fatalf("unexpected first page: %v", page)
	}
	older := env.do(http.MethodGet, msgPath+"?limit=3&before="+got[0].(map[string]any)["id"].(string), alice.Token, nil, alice.addr).Map(t)
	if o := older["messages"].([]any); len(o) != 3 || o[0].(map[string]any)["body"] != "Salut Bob !" || older["hasMore"] != false {
		t.Fatalf("unexpected older page: %v", older)
	}

	// Local deletion hides history for one side only.
	if r := env.do(http.MethodDelete, "/conversations/"+matchID, alice.Token, nil, alice.addr); r.Status != http.StatusNoContent {
		t.Fatalf("clear conversation: %d", r.Status)
	}
	if m := env.do(http.MethodGet, msgPath, alice.Token, nil, alice.addr).Map(t)["messages"].([]any); len(m) != 0 {
		t.Fatalf("alice's history should be hidden: %v", m)
	}
	if m := env.do(http.MethodGet, msgPath, bob.Token, nil, bob.addr).Map(t)["messages"].([]any); len(m) != 6 {
		t.Fatalf("bob keeps his history: %d", len(m))
	}

	// Unmatch removes the conversation for both and does not allow silent re-matching.
	if r := env.do(http.MethodDelete, "/matches/"+matchID, alice.Token, nil, alice.addr); r.Status != http.StatusNoContent {
		t.Fatalf("unmatch: %d", r.Status)
	}
	if r := env.do(http.MethodGet, msgPath, bob.Token, nil, bob.addr); r.Status != http.StatusNotFound {
		t.Fatalf("conversation should be gone, got %d", r.Status)
	}
}

func TestPassHidesProfileAndLikeCanBeRetractedBeforeMatch(t *testing.T) {
	env := newTestEnv(t)
	a, b := env.register("a"), env.register("b")
	env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}})
	env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})

	env.swipe(a, b, "pass")
	if ids := env.discoverIDs(a); contains(ids, b.ID) {
		t.Fatal("passed profile must not return")
	}
	// Changing a pass into a like before any match is allowed (user error recovery).
	env.swipe(a, b, "like")
	if res := env.swipe(b, a, "like"); res["matched"] != true {
		t.Fatalf("like after change of mind must match: %v", res)
	}
}

func TestSwipeValidation(t *testing.T) {
	env := newTestEnv(t)
	a, b := env.register("a"), env.register("b")
	env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}})
	env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})

	cases := []struct {
		name string
		body map[string]string
		want int
	}{
		{"bad action", map[string]string{"targetId": b.ID, "action": "superlike"}, http.StatusBadRequest},
		{"self", map[string]string{"targetId": a.ID, "action": "like"}, http.StatusBadRequest},
		{"not a uuid", map[string]string{"targetId": "x'; DROP TABLE users;--", "action": "like"}, http.StatusBadRequest},
		{"unknown user", map[string]string{"targetId": "00000000-0000-4000-8000-000000000000", "action": "like"}, http.StatusNotFound},
	}
	for _, tc := range cases {
		if res := env.do(http.MethodPost, "/swipes", a.Token, tc.body, a.addr); res.Status != tc.want {
			t.Errorf("%s: got %d, want %d (%s)", tc.name, res.Status, tc.want, res.Body)
		}
	}

	// Somebody without a profile cannot swipe.
	c := env.register("c")
	if res := env.do(http.MethodPost, "/swipes", c.Token, map[string]string{"targetId": b.ID, "action": "like"}, c.addr); res.Status != http.StatusConflict {
		t.Errorf("swipe without profile: got %d", res.Status)
	}
}

// Two people liking each other at the same instant must still produce exactly one match.
func TestConcurrentMutualLikesProduceOneMatch(t *testing.T) {
	env := newTestEnv(t)
	const pairs = 12
	type pair struct{ a, b *user }
	var all []pair
	for i := 0; i < pairs; i++ {
		a, b := env.register("pa"), env.register("pb")
		env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}})
		env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})
		all = append(all, pair{a, b})
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, p := range all {
		for _, dir := range [][2]*user{{p.a, p.b}, {p.b, p.a}} {
			wg.Add(1)
			go func(actor, target *user) {
				defer wg.Done()
				<-start
				env.do(http.MethodPost, "/swipes", actor.Token, map[string]string{"targetId": target.ID, "action": "like"}, actor.addr)
			}(dir[0], dir[1])
		}
	}
	close(start)
	wg.Wait()

	var matches int
	if err := env.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches`).Scan(&matches); err != nil {
		t.Fatal(err)
	}
	if matches != pairs {
		t.Fatalf("expected exactly %d matches, got %d", pairs, matches)
	}
}
