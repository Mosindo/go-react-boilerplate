package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID string `json:"id"`
	} `json:"photos"`
}

func (e *env) discover(p person) []card {
	e.t.Helper()
	var out struct {
		Profiles []card `json:"profiles"`
	}
	e.expect(e.do("GET", "/discover?limit=20", p.Token, nil), http.StatusOK).json(e.t, &out)
	return out.Profiles
}

func hasCard(cards []card, id string) bool {
	for _, c := range cards {
		if c.UserID == id {
			return true
		}
	}
	return false
}

func (e *env) swipe(p person, target person, action string) resp {
	return e.do("POST", "/swipes", p.Token, map[string]string{"targetId": target.ID, "action": action})
}

// TestCriticalFlow covers: register -> profile -> discover -> like -> match -> conversation.
func TestCriticalFlow(t *testing.T) {
	e := newEnv(t)
	alice, bob, carol := e.register("alice"), e.register("bob"), e.register("carol")

	// before onboarding nothing is discoverable and the profile is incomplete
	e.expect(e.do("GET", "/me/profile", alice.Token, nil), http.StatusNotFound)
	me := e.expect(e.do("GET", "/me", alice.Token, nil), http.StatusOK)
	if strings.Contains(string(me.Body), `"profileComplete":true`) {
		t.Fatal("new account must not be complete")
	}
	if got := e.discover(alice); len(got) != 0 {
		t.Fatalf("incomplete profile must see nobody, got %d", len(got))
	}

	e.onboard(alice, profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 29, Lat: fptr(48.8566), Lon: fptr(2.3522), MaxDistance: iptr(50)})
	e.onboard(bob, profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Age: 31, Lat: fptr(48.8606), Lon: fptr(2.3376), MaxDistance: iptr(50)})
	// carol is a woman interested in women: not a mutual fit with Alice
	e.onboard(carol, profileSpec{Name: "Carol", Gender: "woman", InterestedIn: []string{"woman"}, Age: 28})

	me = e.expect(e.do("GET", "/me", alice.Token, nil), http.StatusOK)
	if !strings.Contains(string(me.Body), `"profileComplete":true`) {
		t.Fatalf("expected complete profile: %s", me.Body)
	}

	// discovery respects mutual preferences, hides self and never leaks exact location / birth date / email
	feed := e.expect(e.do("GET", "/discover", alice.Token, nil), http.StatusOK)
	var parsed struct {
		Profiles []card `json:"profiles"`
	}
	feed.json(t, &parsed)
	if !hasCard(parsed.Profiles, bob.ID) || hasCard(parsed.Profiles, alice.ID) || hasCard(parsed.Profiles, carol.ID) {
		t.Fatalf("unexpected feed: %s", feed.Body)
	}
	for _, leak := range []string{"latitude", "longitude", "birthDate", "email", "48.86"} {
		if contains(feed.Body, leak) {
			t.Fatalf("discovery leaked %q: %s", leak, feed.Body)
		}
	}
	if d := parsed.Profiles[0].DistanceKm; d == nil || *d%5 != 0 || *d < 5 {
		t.Fatalf("distance must be a coarse 5km bucket, got %v", d)
	}

	// swiping on someone the feed would not show is refused server-side
	e.expect(e.swipe(alice, carol, "like"), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": alice.ID, "action": "like"}), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "superlike"}), http.StatusBadRequest)

	// one-sided like: no match, no conversation yet
	first := e.expect(e.swipe(alice, bob, "like"), http.StatusOK)
	if first.str(t, "action") != "like" || strings.Contains(string(first.Body), `"matched":true`) {
		t.Fatalf("unexpected first swipe: %s", first.Body)
	}
	// already treated profiles never come back
	if hasCard(e.discover(alice), bob.ID) {
		t.Fatal("swiped profile reappeared in feed")
	}
	// repeating the same decision is idempotent, changing it is refused
	e.expect(e.swipe(alice, bob, "like"), http.StatusOK)
	e.expect(e.swipe(alice, bob, "pass"), http.StatusConflict)
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			MatchID     string `json:"matchId"`
			UnreadCount int    `json:"unreadCount"`
			User        card   `json:"user"`
		} `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", alice.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("no conversation before a mutual like")
	}

	// bob sees alice first (she already liked him) and likes back -> match
	if got := e.discover(bob); len(got) == 0 || got[0].UserID != alice.ID {
		t.Fatalf("expected alice first in bob's feed: %+v", got)
	}

	// realtime: alice listens for the match and the messages
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	aliceWS := dialWS(t, srv.URL, alice.Token)
	defer aliceWS.Close()

	second := e.expect(e.swipe(bob, alice, "like"), http.StatusOK)
	if !strings.Contains(string(second.Body), `"matched":true`) {
		t.Fatalf("expected a match: %s", second.Body)
	}
	convID := second.str(t, "match", "conversationId")
	if convID == "" {
		t.Fatal("match must carry a conversation id")
	}
	if ev := aliceWS.next(t, "match.new"); !strings.Contains(string(ev), bob.ID) {
		t.Fatalf("alice should be told about the match with bob: %s", ev)
	}
	// replaying the winning like stays a single match
	again := e.expect(e.swipe(bob, alice, "like"), http.StatusOK)
	if again.str(t, "match", "id") != second.str(t, "match", "id") {
		t.Fatal("replay must return the same match")
	}
	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low = LEAST($1::uuid,$2::uuid) AND user_high = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}

	// both see the conversation, and both received a match notification
	for _, p := range []person{alice, bob} {
		e.expect(e.do("GET", "/conversations", p.Token, nil), http.StatusOK).json(t, &convs)
		if len(convs.Conversations) != 1 || convs.Conversations[0].ID != convID {
			t.Fatalf("%s should see the new conversation: %+v", p.Email, convs)
		}
		var notes struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
		}
		e.expect(e.do("GET", "/notifications", p.Token, nil), http.StatusOK).json(t, &notes)
		if len(notes.Notifications) != 1 || notes.Notifications[0].Type != "match" {
			t.Fatalf("expected a match notification for %s: %+v", p.Email, notes)
		}
	}

	// chat: send, realtime delivery, unread state, read receipts
	msgPath := "/conversations/" + convID + "/messages"
	e.expect(e.do("POST", msgPath, bob.Token, map[string]string{"content": "   "}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", msgPath, bob.Token, map[string]string{"content": strings.Repeat("x", 2001)}), http.StatusUnprocessableEntity)
	sent := e.expect(e.do("POST", msgPath, bob.Token, map[string]string{"content": "  Salut Alice !  "}), http.StatusCreated)
	if sent.str(t, "content") != "Salut Alice !" || sent.str(t, "senderId") != bob.ID {
		t.Fatalf("unexpected message: %s", sent.Body)
	}
	if ev := aliceWS.next(t, "message.new"); !strings.Contains(string(ev), "Salut Alice") {
		t.Fatalf("alice should receive the message live: %s", ev)
	}
	var unread struct {
		Unread int `json:"unread"`
	}
	e.expect(e.do("GET", "/conversations/unread-count", alice.Token, nil), http.StatusOK).json(t, &unread)
	if unread.Unread != 1 {
		t.Fatalf("expected 1 unread, got %d", unread.Unread)
	}
	e.expect(e.do("GET", "/conversations", alice.Token, nil), http.StatusOK).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 1 {
		t.Fatalf("expected unread count 1 on conversation: %+v", convs)
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/read", alice.Token, nil), http.StatusOK)
	e.expect(e.do("GET", "/conversations/unread-count", alice.Token, nil), http.StatusOK).json(t, &unread)
	if unread.Unread != 0 {
		t.Fatalf("expected 0 unread after read, got %d", unread.Unread)
	}
	var history struct {
		Messages []struct {
			ReadAt *time.Time `json:"readAt"`
		} `json:"messages"`
	}
	e.expect(e.do("GET", msgPath, bob.Token, nil), http.StatusOK).json(t, &history)
	if len(history.Messages) != 1 || history.Messages[0].ReadAt == nil {
		t.Fatalf("bob's message should be flagged read: %+v", history)
	}

	// permissions: an outsider can neither read, write, mark read nor hide it - and learns nothing about it
	e.expect(e.do("GET", msgPath, carol.Token, nil), http.StatusNotFound)
	e.expect(e.do("POST", msgPath, carol.Token, map[string]string{"content": "intrusion"}), http.StatusNotFound)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", carol.Token, nil), http.StatusNotFound)
	e.expect(e.do("DELETE", "/conversations/"+convID, carol.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", alice.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/conversations", "", nil), http.StatusUnauthorized)

	// local deletion hides it for alice only; a new message brings it back without the old history
	e.expect(e.do("DELETE", "/conversations/"+convID, alice.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/conversations", alice.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("conversation should be hidden for alice")
	}
	e.expect(e.do("GET", "/conversations", bob.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("conversation must stay visible for bob")
	}
	time.Sleep(5 * time.Millisecond)
	e.expect(e.do("POST", msgPath, bob.Token, map[string]string{"content": "Tu es là ?"}), http.StatusCreated)
	e.expect(e.do("GET", msgPath, alice.Token, nil), http.StatusOK).json(t, &history)
	if len(history.Messages) != 1 {
		t.Fatalf("alice should only see messages after clearing, got %d", len(history.Messages))
	}

	// unmatch closes the conversation for both
	e.expect(e.do("DELETE", "/matches/"+second.str(t, "match", "id"), alice.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", msgPath, bob.Token, nil), http.StatusNotFound)
	e.expect(e.do("POST", msgPath, alice.Token, map[string]string{"content": "x"}), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+second.str(t, "match", "id"), alice.Token, nil), http.StatusNotFound)
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	a, b := e.register("racea"), e.register("raceb")
	e.onboard(a, profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}, Age: 30})
	e.onboard(b, profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})

	var wg sync.WaitGroup
	results := make([]resp, 2)
	for i, pair := range [][2]person{{a, b}, {b, a}} {
		wg.Add(1)
		go func(i int, from, to person) {
			defer wg.Done()
			results[i] = e.swipe(from, to, "like")
		}(i, pair[0], pair[1])
	}
	wg.Wait()
	matched := 0
	for _, r := range results {
		if r.Status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", r.Status, r.Body)
		}
		if strings.Contains(string(r.Body), `"matched":true`) {
			matched++
		}
	}
	if matched == 0 {
		t.Fatal("at least one side must see the match")
	}
	var matches, convs int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low IN ($1,$2) AND user_high IN ($1,$2)`, a.ID, b.ID).Scan(&matches)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_low IN ($1,$2) AND m.user_high IN ($1,$2)`, a.ID, b.ID).Scan(&convs)
	if matches != 1 || convs != 1 {
		t.Fatalf("expected 1 match and 1 conversation, got %d / %d", matches, convs)
	}
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	viewer := e.register("viewer")
	near := e.register("near")
	far := e.register("far")
	young := e.register("young")
	hidden := e.register("hidden")
	e.onboard(viewer, profileSpec{Name: "Vic", Gender: "woman", InterestedIn: []string{"man"}, Age: 30, Lat: fptr(48.85), Lon: fptr(2.35), MaxDistance: iptr(100)})
	e.onboard(near, profileSpec{Name: "Near", Gender: "man", InterestedIn: []string{"woman"}, Age: 30, Lat: fptr(48.9), Lon: fptr(2.4), MaxDistance: iptr(100)})
	e.onboard(far, profileSpec{Name: "Far", Gender: "man", InterestedIn: []string{"woman"}, Age: 30, Lat: fptr(43.3), Lon: fptr(5.37), MaxDistance: iptr(1000)}) // Marseille
	e.onboard(young, profileSpec{Name: "Young", Gender: "man", InterestedIn: []string{"woman"}, Age: 19, Lat: fptr(48.9), Lon: fptr(2.4)})
	e.onboard(hidden, profileSpec{Name: "Hidden", Gender: "man", InterestedIn: []string{"woman"}, Age: 30, Lat: fptr(48.9), Lon: fptr(2.4)})

	// ageMax 25 for the viewer excludes 30-year-olds but keeps the 19-year-old
	e.expect(e.do("PUT", "/me/preferences", viewer.Token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 25, "maxDistanceKm": 100}), http.StatusOK)
	feed := e.discover(viewer)
	if !hasCard(feed, young.ID) || hasCard(feed, near.ID) || hasCard(feed, far.ID) {
		t.Fatalf("age filter failed: %+v", feed)
	}

	e.expect(e.do("PUT", "/me/preferences", viewer.Token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 100}), http.StatusOK)
	feed = e.discover(viewer)
	if !hasCard(feed, near.ID) || hasCard(feed, far.ID) {
		t.Fatalf("distance filter failed: %+v", feed)
	}

	// anywhere (null) still honours the candidate's own max distance (far accepts 1000km, hidden doesn't care)
	e.expect(e.do("PUT", "/me/preferences", viewer.Token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 60}), http.StatusOK)
	if feed = e.discover(viewer); !hasCard(feed, far.ID) {
		t.Fatalf("anywhere should include far: %+v", feed)
	}

	// Unknown positions never exclude anyone (LEAST ignores NULLs in SQL, which once turned "unknown" into 20000 km).
	noLoc := e.register("noloc")
	e.onboard(noLoc, profileSpec{Name: "Nolo", Gender: "woman", InterestedIn: []string{"man"}, Age: 30, MaxDistance: iptr(10)})
	if !hasCard(e.discover(noLoc), near.ID) || !hasCard(e.discover(noLoc), far.ID) {
		t.Fatal("a viewer without a position must still see candidates despite a max distance")
	}
	noLocMan := e.register("nolocman")
	e.onboard(noLocMan, profileSpec{Name: "Nolan", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})
	if feed = e.discover(viewer); !hasCard(feed, noLocMan.ID) {
		t.Fatal("a candidate without a position must stay visible to viewers with a max distance")
	}

	// discoverable=false removes the profile from feeds and from direct profile/photo access
	e.expect(e.do("PUT", "/me/profile", hidden.Token, map[string]any{
		"firstName": "Hidden", "birthDate": time.Now().AddDate(-30, 0, -3).Format("2006-01-02"),
		"gender": "man", "discoverable": false,
	}), http.StatusOK)
	if hasCard(e.discover(viewer), hidden.ID) {
		t.Fatal("non-discoverable profile must not be listed")
	}
	e.expect(e.do("GET", "/profiles/"+hidden.ID, viewer.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+near.ID, viewer.Token, nil), http.StatusOK)
	e.expect(e.do("GET", "/profiles/"+viewer.ID, viewer.Token, nil), http.StatusOK)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("validate")
	adult := time.Now().AddDate(-25, 0, -1).Format("2006-01-02")
	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	cases := []struct {
		name string
		body map[string]any
	}{
		{"minor", map[string]any{"firstName": "Kid", "birthDate": minor, "gender": "man"}},
		{"bad date", map[string]any{"firstName": "Kid", "birthDate": "31/12/1999", "gender": "man"}},
		{"bad gender", map[string]any{"firstName": "Sam", "birthDate": adult, "gender": "robot"}},
		{"empty name", map[string]any{"firstName": "  ", "birthDate": adult, "gender": "man"}},
		{"digits in name", map[string]any{"firstName": "R2D2", "birthDate": adult, "gender": "man"}},
		{"long bio", map[string]any{"firstName": "Sam", "birthDate": adult, "gender": "man", "bio": strings.Repeat("a", 501)}},
		{"unknown interest", map[string]any{"firstName": "Sam", "birthDate": adult, "gender": "man", "interests": []string{"does-not-exist"}}},
	}
	for _, tc := range cases {
		if r := e.do("PUT", "/me/profile", u.Token, tc.body); r.Status != http.StatusUnprocessableEntity {
			t.Errorf("%s: expected 422, got %d (%s)", tc.name, r.Status, r.Body)
		}
	}
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{"firstName": "  Jean-Marc ", "birthDate": adult, "gender": "man", "interests": []string{"voyage", "voyage"}}), http.StatusOK)
	// birth date is immutable once set
	other := time.Now().AddDate(-40, 0, 0).Format("2006-01-02")
	e.expect(e.do("PUT", "/me/profile", u.Token, map[string]any{"firstName": "Jean-Marc", "birthDate": other, "gender": "man"}), http.StatusUnprocessableEntity)

	for _, bad := range []map[string]any{
		{"interestedIn": []string{}, "ageMin": 18, "ageMax": 30},
		{"interestedIn": []string{"man"}, "ageMin": 17, "ageMax": 30},
		{"interestedIn": []string{"man"}, "ageMin": 40, "ageMax": 30},
		{"interestedIn": []string{"alien"}, "ageMin": 18, "ageMax": 30},
		{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 30, "maxDistanceKm": 0},
	} {
		e.expect(e.do("PUT", "/me/preferences", u.Token, bad), http.StatusUnprocessableEntity)
	}
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 91, "longitude": 0}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222}), http.StatusNoContent)
	var lat float64
	_ = e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat)
	if lat != 48.86 {
		t.Fatalf("stored latitude must be coarsened, got %v", lat)
	}
	e.expect(e.do("DELETE", "/me/location", u.Token, nil), http.StatusNoContent)
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t)
	a, b, c := e.register("blka"), e.register("blkb"), e.register("blkc")
	e.onboard(a, profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}, Age: 30})
	e.onboard(b, profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})
	e.onboard(c, profileSpec{Name: "Cal", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})

	e.expect(e.swipe(a, b, "like"), http.StatusOK)
	m := e.expect(e.swipe(b, a, "like"), http.StatusOK)
	convID := m.str(t, "match", "conversationId")
	photoID := ""
	var prof struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), http.StatusOK).json(t, &prof)
	photoID = prof.Photos[0].ID
	e.expect(e.do("GET", "/photos/"+photoID+"?size=thumb", a.Token, nil), http.StatusOK)

	// blocking: validation, then effects
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": a.ID}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}), http.StatusNotFound)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), http.StatusNoContent)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), http.StatusNoContent) // idempotent

	for _, p := range []person{a, b} {
		e.expect(e.do("GET", "/conversations/"+convID+"/messages", p.Token, nil), http.StatusNotFound)
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", p.Token, map[string]string{"content": "hello?"}), http.StatusNotFound)
	}
	other := map[string]person{a.ID: b, b.ID: a}
	for _, p := range []person{a, b} {
		o := other[p.ID]
		e.expect(e.do("GET", "/profiles/"+o.ID, p.Token, nil), http.StatusNotFound)
		e.expect(e.do("GET", "/photos/"+photoID, a.Token, nil), http.StatusNotFound)
		if hasCard(e.discover(p), o.ID) {
			t.Fatalf("%s must not see blocked counterpart in discovery", p.Email)
		}
	}
	e.expect(e.swipe(b, a, "like"), http.StatusNotFound)

	var blocked struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	e.expect(e.do("GET", "/blocks", a.Token, nil), http.StatusOK).json(t, &blocked)
	if len(blocked.Blocks) != 1 || blocked.Blocks[0].UserID != b.ID || blocked.Blocks[0].FirstName != "Ben" {
		t.Fatalf("unexpected block list: %+v", blocked)
	}
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), http.StatusOK)
	// unblocking does not resurrect the closed conversation
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", a.Token, nil), http.StatusNotFound)

	// reports
	e.expect(e.do("POST", "/reports", a.Token, map[string]string{"userId": c.ID, "reason": "weird"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/reports", a.Token, map[string]string{"userId": a.ID, "reason": "spam"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/reports", a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), http.StatusNotFound)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": c.ID, "reason": "harassment", "details": "insultes\x00 répétées", "block": true}), http.StatusCreated)
	e.expect(e.do("POST", "/reports", a.Token, map[string]any{"userId": c.ID, "reason": "harassment"}), http.StatusCreated) // duplicate open report is absorbed
	var reports int
	var details string
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*), MAX(details) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, c.ID).Scan(&reports, &details)
	if reports != 1 || strings.ContainsRune(details, 0) {
		t.Fatalf("expected one sanitized report, got %d %q", reports, details)
	}
	if hasCard(e.discover(a), c.ID) {
		t.Fatal("report with block=true must hide the reported user")
	}
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a, b := e.register("dela"), e.register("delb")
	e.onboard(a, profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}, Age: 30})
	e.onboard(b, profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})
	e.expect(e.swipe(a, b, "like"), http.StatusOK)
	m := e.expect(e.swipe(b, a, "like"), http.StatusOK)
	e.expect(e.do("POST", "/conversations/"+m.str(t, "match", "conversationId")+"/messages", a.Token, map[string]string{"content": "salut"}), http.StatusCreated)

	files, _ := filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	if len(files) != 4 { // full + thumb for each of the two photos
		t.Fatalf("expected 4 stored files, got %d", len(files))
	}

	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"}), http.StatusForbidden)
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "Passw0rd!x"}), http.StatusNoContent)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Passw0rd!x"}), http.StatusUnauthorized)

	for table, col := range map[string]string{"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "swipes": "actor_id", "notifications": "user_id", "sessions": "user_id", "user_interests": "user_id"} {
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE `+col+` = $1`, a.ID).Scan(&n)
		if n != 0 {
			t.Errorf("%s still holds %d rows of the deleted user", table, n)
		}
	}
	var leftovers int
	_ = e.pool.QueryRow(context.Background(), `SELECT (SELECT COUNT(*) FROM matches WHERE user_low = $1 OR user_high = $1) + (SELECT COUNT(*) FROM messages WHERE sender_user_id = $1)`, a.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("matches/messages of the deleted user remain: %d", leftovers)
	}
	files, _ = filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	if len(files) != 2 {
		t.Fatalf("photo files of the deleted user must be removed, %d left", len(files))
	}
	var convs struct {
		Conversations []any `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", b.Token, nil), http.StatusOK).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("bob must no longer see a conversation with a deleted account")
	}
	_ = os.Remove(e.dir)
}

// --- websocket helpers ---

type wsClient struct {
	conn   *websocket.Conn
	events chan json.RawMessage
}

func dialWS(t *testing.T, serverURL, token string) *wsClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	c := &wsClient{conn: conn, events: make(chan json.RawMessage, 64)}
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				close(c.events)
				return
			}
			c.events <- data
		}
	}()
	c.next(t, "ready")
	return c
}

func (c *wsClient) Close() { _ = c.conn.Close() }

// next waits for the next event of the given type.
func (c *wsClient) next(t *testing.T, typ string) json.RawMessage {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case raw, ok := <-c.events:
			if !ok {
				t.Fatalf("websocket closed while waiting for %q", typ)
			}
			var ev struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(raw, &ev)
			if ev.Type == typ {
				return raw
			}
		case <-deadline:
			t.Fatalf("timed out waiting for websocket event %q", typ)
		}
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "not-a-token"})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected the server to close the connection")
	}
}
