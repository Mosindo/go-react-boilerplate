package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

type swipeResp struct {
	Action           string `json:"action"`
	AlreadyProcessed bool   `json:"alreadyProcessed"`
	Matched          bool   `json:"matched"`
	Match            *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
		User           struct {
			ID        string  `json:"id"`
			FirstName string  `json:"firstName"`
			PhotoURL  *string `json:"photoUrl"`
		} `json:"user"`
	} `json:"match"`
}

func (e *testEnv) swipeOK(u, target testUser, action string) swipeResp {
	e.t.Helper()
	var s swipeResp
	e.expect(e.swipe(u, target, action), 200).decode(e.t, &s)
	return s
}

func TestDiscoveryFilters(t *testing.T) {
	e := newTestEnv(t)
	me := e.onboard(userSpec{Name: "Me", Gender: "man", BirthYear: 1995, Into: []string{"woman"}, MinAge: 25, MaxAge: 40, MaxDist: 50})

	ok := e.onboard(userSpec{Name: "Ok", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	wrongGender := e.onboard(userSpec{Name: "WrongGender", Gender: "man", BirthYear: 1996, Into: []string{"man"}})
	notIntoMe := e.onboard(userSpec{Name: "NotIntoMe", Gender: "woman", BirthYear: 1996, Into: []string{"woman"}})
	tooYoung := e.onboard(userSpec{Name: "TooYoung", Gender: "woman", BirthYear: 2005, Into: []string{"man"}})
	tooOld := e.onboard(userSpec{Name: "TooOld", Gender: "woman", BirthYear: 1970, Into: []string{"man"}})
	theirRange := e.onboard(userSpec{Name: "TheirRange", Gender: "woman", BirthYear: 1996, Into: []string{"man"}, MinAge: 40, MaxAge: 50})
	far := e.onboard(userSpec{Name: "Far", Gender: "woman", BirthYear: 1996, Into: []string{"man"}, Lat: 43.2965, Lng: 5.3698})                              // Marseille ~660 km
	theirDistance := e.onboard(userSpec{Name: "ShortReach", Gender: "woman", BirthYear: 1996, Into: []string{"man"}, Lat: 48.9566, Lng: 2.3522, MaxDist: 5}) // ~11 km away, wants <=5
	noPhoto := e.onboard(userSpec{Name: "NoPhoto", Gender: "woman", BirthYear: 1996, Into: []string{"man"}, NoPhoto: true})
	hidden := e.onboard(userSpec{Name: "Hidden", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	e.expect(e.do("PUT", "/profile", hidden.Token, map[string]any{"firstName": "Hidden", "birthDate": "1996-03-15", "gender": "woman", "isVisible": false}), 200)
	blocked := e.onboard(userSpec{Name: "Blocked", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	blockedBy := e.onboard(userSpec{Name: "BlockedBy", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	passed := e.onboard(userSpec{Name: "Passed", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	liked := e.onboard(userSpec{Name: "Liked", Gender: "woman", BirthYear: 1996, Into: []string{"man"}})
	incomplete := e.register("incomplete")

	e.expect(e.do("POST", "/blocks", me.Token, map[string]string{"userId": blocked.ID}), 204)
	e.expect(e.do("POST", "/blocks", blockedBy.Token, map[string]string{"userId": me.ID}), 204)
	e.swipeOK(me, passed, "pass")
	e.swipeOK(me, liked, "like")

	ids := e.discoverIDs(me)
	if !contains(ids, ok.ID) {
		t.Fatalf("compatible profile should be discovered, got %v", ids)
	}
	for name, u := range map[string]testUser{
		"self": me, "wrongGender": wrongGender, "notIntoMe": notIntoMe, "tooYoung": tooYoung, "tooOld": tooOld,
		"theirAgeRange": theirRange, "far": far, "theirDistance": theirDistance, "noPhoto": noPhoto, "hidden": hidden,
		"blocked": blocked, "blockedBy": blockedBy, "passed": passed, "liked": liked, "incomplete": incomplete,
	} {
		if contains(ids, u.ID) {
			t.Errorf("%s must not be discovered", name)
		}
	}
	if len(ids) != 1 {
		t.Errorf("expected exactly one candidate, got %d: %v", len(ids), ids)
	}

	// Incomplete accounts cannot discover or swipe.
	e.expect(e.do("GET", "/discover", incomplete.Token, nil), 200)
	if got := e.discoverIDs(incomplete); len(got) != 0 {
		t.Fatalf("incomplete profile must get no candidates, got %v", got)
	}
	e.expect(e.swipe(incomplete, ok, "like"), 403)

	e.expect(e.do("GET", "/discover?limit=0", me.Token, nil), 400)
	e.expect(e.do("GET", "/discover?limit=1000", me.Token, nil), 400)
	e.expect(e.do("GET", "/discover?limit=abc", me.Token, nil), 400)
}

func TestDiscoveryRevealsOnlyApproximateData(t *testing.T) {
	e := newTestEnv(t)
	me := e.onboard(userSpec{Name: "Me", Gender: "man", Into: []string{"woman"}})
	_ = e.onboard(userSpec{Name: "Near", Gender: "woman", Into: []string{"man"}, Lat: 48.8966, Lng: 2.3922, Interests: []int16{1, 2}})

	r := e.expect(e.do("GET", "/discover", me.Token, nil), 200)
	body := string(r.Body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "48.89", "2.39"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("discover payload leaks %q: %s", forbidden, body)
		}
	}
	var d discoverResp
	r.decode(t, &d)
	if len(d.Profiles) != 1 || d.Profiles[0].DistanceKm == nil {
		t.Fatalf("expected one candidate with an approximate distance: %s", body)
	}
	if got := *d.Profiles[0].DistanceKm; got < 4 || got > 10 {
		t.Fatalf("expected ~5 km, got %d", got)
	}
	if d.Profiles[0].Age < 18 {
		t.Fatal("age should be derived")
	}

	// A user who hides their distance exposes no distance at all.
	near2 := e.onboard(userSpec{Name: "Private", Gender: "woman", Into: []string{"man"}})
	e.expect(e.do("PUT", "/profile", near2.Token, map[string]any{"firstName": "Private", "birthDate": "1995-03-15", "gender": "woman", "showDistance": false}), 200)
	e.swipeOK(me, testUser{ID: d.Profiles[0].ID}, "pass")
	r = e.expect(e.do("GET", "/discover", me.Token, nil), 200)
	r.decode(t, &d)
	if len(d.Profiles) != 1 || d.Profiles[0].DistanceKm != nil {
		t.Fatalf("distance must be hidden when showDistance=false: %s", r.Body)
	}
}

func TestLikeMatchFlowAndDuplicates(t *testing.T) {
	e := newTestEnv(t)
	a := e.onboard(userSpec{Name: "Alice", Gender: "woman", Into: []string{"man"}})
	b := e.onboard(userSpec{Name: "Bob", Gender: "man", Into: []string{"woman"}})

	// Invalid input.
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"userId": b.ID, "action": "superlike"}), 400)
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"userId": "nope", "action": "like"}), 400)
	e.expect(e.swipe(a, a, "like"), 400)
	e.expect(e.swipe(a, testUser{ID: "11111111-1111-4111-8111-111111111111"}, "like"), 404)

	// A likes B: no match yet.
	first := e.swipeOK(a, b, "like")
	if first.Matched || first.AlreadyProcessed {
		t.Fatalf("unexpected first swipe result: %+v", first)
	}
	// Repeating is idempotent; changing your mind is refused.
	again := e.swipeOK(a, b, "like")
	if !again.AlreadyProcessed || again.Matched {
		t.Fatalf("repeated like must be idempotent: %+v", again)
	}
	e.expect(e.swipe(a, b, "pass"), 409)

	// B no longer sees A as a stranger but sees A (who liked B) in discovery, and A no longer sees B.
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("already-swiped profile reappeared")
	}
	if !contains(e.discoverIDs(b), a.ID) {
		t.Fatal("a profile that liked me should be discoverable")
	}

	// B likes A: match for both.
	m := e.swipeOK(b, a, "like")
	if !m.Matched || m.Match == nil || m.Match.User.ID != a.ID || m.Match.User.FirstName != "Alice" || m.Match.ConversationID == "" {
		t.Fatalf("expected a match with Alice: %+v", m)
	}
	if m.Match.User.PhotoURL == nil {
		t.Fatal("match card should carry the main photo")
	}
	replay := e.swipeOK(b, a, "like")
	if !replay.AlreadyProcessed || !replay.Matched || replay.Match.ID != m.Match.ID {
		t.Fatalf("replaying a matching like must return the same match: %+v", replay)
	}

	var matches, convs, parts, notifs int
	ctx := t.Context()
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversations`).Scan(&convs)
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversation_participants`).Scan(&parts)
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE type = 'match'`).Scan(&notifs)
	if matches != 1 || convs != 1 || parts != 2 || notifs != 2 {
		t.Fatalf("expected 1 match, 1 conversation, 2 participants, 2 notifications; got %d %d %d %d", matches, convs, parts, notifs)
	}

	// Both see a match notification and the conversation.
	for _, u := range []testUser{a, b} {
		var n struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.expect(e.do("GET", "/notifications", u.Token, nil), 200).decode(t, &n)
		if n.UnreadCount != 1 || len(n.Notifications) != 1 || n.Notifications[0].Type != "match" {
			t.Fatalf("expected one unread match notification, got %+v", n)
		}
	}
	// Matched people are no longer proposed to each other.
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("matched users must not be proposed again")
	}

	// Unmatch removes the conversation but they never reappear.
	e.expect(e.do("DELETE", "/matches/"+m.Match.ID, a.Token, nil), 204)
	e.expect(e.do("DELETE", "/matches/"+m.Match.ID, a.Token, nil), 404)
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversations`).Scan(&convs)
	if convs != 0 {
		t.Fatal("conversation must disappear with its match")
	}
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("unmatched users must not reappear in discovery")
	}
}

func TestMutualLikesAtTheSameTimeCreateOneMatch(t *testing.T) {
	e := newTestEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard(userSpec{Name: fmt.Sprintf("A%d", round), Gender: "woman", Into: []string{"man"}})
		b := e.onboard(userSpec{Name: fmt.Sprintf("B%d", round), Gender: "man", Into: []string{"woman"}})

		var wg sync.WaitGroup
		results := make([]swipeResp, 2)
		for i, pair := range [][2]testUser{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				r := e.swipe(pair[0], pair[1], "like")
				if r.Status != 200 {
					t.Errorf("swipe status %d: %s", r.Status, r.Body)
					return
				}
				r.decode(t, &results[i])
			}()
		}
		wg.Wait()

		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: at least one side must see the match: %+v", round, results)
		}
		var n int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_low_id IN ($1,$2) AND user_high_id IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
	}
}

func TestPassNeverMatches(t *testing.T) {
	e := newTestEnv(t)
	a := e.onboard(userSpec{Name: "A", Gender: "woman", Into: []string{"man"}})
	b := e.onboard(userSpec{Name: "B", Gender: "man", Into: []string{"woman"}})
	e.swipeOK(a, b, "pass")
	r := e.swipeOK(b, a, "like")
	if r.Matched {
		t.Fatal("a pass followed by a like must not match")
	}
}

func TestUnmatchFromConversationAndPermissions(t *testing.T) {
	e := newTestEnv(t)
	a, b, conv := e.matchedPair("Alice", "Bob")
	c := e.onboard(userSpec{Name: "Carol", Gender: "woman", Into: []string{"man"}})

	e.expect(e.do("DELETE", "/conversations/"+conv+"/match", c.Token, nil), 404) // outsiders cannot end it
	e.expect(e.do("DELETE", "/conversations/nope/match", a.Token, nil), 404)
	e.expect(e.do("DELETE", "/conversations/"+conv+"/match", b.Token, nil), 204)

	var list convList
	e.expect(e.do("GET", "/conversations", a.Token, nil), 200).decode(t, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("the match must be gone for both people")
	}
	e.expect(e.do("DELETE", "/conversations/"+conv+"/match", a.Token, nil), 404)
	if contains(e.discoverIDs(a), b.ID) {
		t.Fatal("unmatched people must not be proposed again")
	}
}
