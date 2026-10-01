package app_test

import (
	"sync"
	"testing"
)

func TestDiscoveryEligibility(t *testing.T) {
	e := newEnv(t)
	me := e.newUser(spec{Name: "Me", Gender: "man", InterestedIn: []string{"woman"}, Age: 30, MinAge: 25, MaxAge: 35, MaxKm: 50})
	ok := e.newUser(spec{Name: "Ok", Gender: "woman", InterestedIn: []string{"man"}, Age: 28, Lat: nearLat, Lng: nearLng})
	wrongGender := e.newUser(spec{Name: "Man", Gender: "man", InterestedIn: []string{"man"}})
	notInterested := e.newUser(spec{Name: "Nope", Gender: "woman", InterestedIn: []string{"woman"}})
	tooYoung := e.newUser(spec{Name: "Young", Gender: "woman", InterestedIn: []string{"man"}, Age: 22})
	candidateRange := e.newUser(spec{Name: "Range", Gender: "woman", InterestedIn: []string{"man"}, Age: 28, MinAge: 35, MaxAge: 45})
	far := e.newUser(spec{Name: "Far", Gender: "woman", InterestedIn: []string{"man"}, Lat: lyonLat, Lng: lyonLng})
	farByCandidate := e.newUser(spec{Name: "Shy", Gender: "woman", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng, MaxKm: 1})
	noPhoto := e.newUser(spec{Name: "NoPic", Gender: "woman", InterestedIn: []string{"man"}, NoPhoto: true})
	hidden := e.newUser(spec{Name: "Hidden", Gender: "woman", InterestedIn: []string{"man"}})
	e.want(e.do("PUT", "/me/profile", hidden.Token, map[string]any{"firstName": "Hidden", "birthDate": "1995-01-01", "gender": "woman", "isVisible": false}), 200)
	blocker := e.newUser(spec{Name: "Blocker", Gender: "woman", InterestedIn: []string{"man"}})
	e.want(e.do("POST", "/blocks", blocker.Token, map[string]any{"userId": me.ID}), 204)

	ids := e.feedIDs(me)
	if !contains(ids, ok.ID) {
		t.Fatalf("eligible profile missing from feed: %v", ids)
	}
	for name, u := range map[string]user{"self": me, "gender": wrongGender, "their-orientation": notInterested, "my-age-range": tooYoung,
		"their-age-range": candidateRange, "far": far, "their-distance": farByCandidate, "no-photo": noPhoto, "hidden": hidden, "blocked-me": blocker} {
		if contains(ids, u.ID) {
			t.Errorf("%s must not be in the feed", name)
		}
	}
	if len(ids) != 1 {
		t.Fatalf("expected only the eligible profile, got %v", ids)
	}

	// Public cards never expose coordinates, email or birth date; distance is bucketed.
	r := e.want(e.do("GET", "/discover", me.Token, nil), 200)
	body := string(r.Body)
	for _, forbidden := range []string{"latitude", "longitude", "email", "birthDate", "48.9", "2.4"} {
		if contains([]string{forbidden}, "") || stringsContains(body, `"`+forbidden+`"`) {
			t.Errorf("feed leaks %q: %s", forbidden, body)
		}
	}
	card := r.json()["profiles"].([]any)[0].(map[string]any)
	if km := int(card["distanceKm"].(float64)); km%5 != 0 || km < 5 {
		t.Fatalf("distance must be bucketed by 5km: %v", km)
	}

	// Incomplete viewers cannot browse.
	e.want(e.do("GET", "/discover", noPhoto.Token, nil), 409)
	e.want(e.do("GET", "/discover", "", nil), 401)
}

func stringsContains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestSwipeAndMatch(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{Name: "A", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.newUser(spec{Name: "B", Gender: "man", InterestedIn: []string{"woman"}})
	c := e.newUser(spec{Name: "C", Gender: "man", InterestedIn: []string{"woman"}})
	d := e.newUser(spec{Name: "D", Gender: "woman", InterestedIn: []string{"woman"}})

	if r := e.swipe(a, a, "like"); r.Status != 400 {
		t.Fatalf("self swipe: %d", r.Status)
	}
	e.want(e.do("POST", "/discover/swipes", a.Token, map[string]string{"userId": b.ID, "action": "super"}), 400)
	e.want(e.do("POST", "/discover/swipes", a.Token, map[string]string{"userId": "nope", "action": "like"}), 404)
	// Server-side eligibility: a user outside A's preferences cannot be swiped even if the client knows the id.
	e.want(e.swipe(a, d, "like"), 404)

	r := e.want(e.swipe(a, b, "like"), 200).json()
	if r["matched"] != false {
		t.Fatalf("one-sided like must not match: %v", r)
	}
	// Repeats are refused, whatever the action.
	e.want(e.swipe(a, b, "like"), 409)
	e.want(e.swipe(a, b, "pass"), 409)
	if n := count(e, `SELECT count(*) FROM swipes WHERE from_user_id=$1 AND to_user_id=$2`, a.ID, b.ID); n != 1 {
		t.Fatalf("duplicate swipe rows: %d", n)
	}
	if contains(e.feedIDs(a), b.ID) {
		t.Fatal("swiped profile must not reappear")
	}
	// B sees A first (A liked B) and can view A's profile.
	if ids := e.feedIDs(b); len(ids) == 0 || ids[0] != a.ID {
		t.Fatalf("a profile that liked me ranks first: %v", ids)
	}
	e.want(e.do("GET", "/profiles/"+a.ID, b.Token, nil), 200)
	e.want(e.do("GET", "/profiles/"+d.ID, a.Token, nil), 404)

	m := e.want(e.swipe(b, a, "like"), 200).json()
	if m["matched"] != true || m["matchId"] == "" || m["conversationId"] == "" {
		t.Fatalf("expected match: %v", m)
	}
	if m["match"].(map[string]any)["userId"] != a.ID {
		t.Fatalf("match must carry the other user's card: %v", m)
	}
	if n := count(e, `SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("matches: %d", n)
	}
	for _, u := range []user{a, b} {
		list := e.want(e.do("GET", "/matches", u.Token, nil), 200).json()["matches"].([]any)
		if len(list) != 1 {
			t.Fatalf("each side sees the match: %v", list)
		}
	}

	// A pass never creates a match; undoing it brings the profile back; matched swipes can't be undone.
	e.want(e.swipe(c, a, "pass"), 200)
	e.want(e.swipe(a, c, "like"), 200)
	if n := count(e, `SELECT count(*) FROM matches`); n != 1 {
		t.Fatalf("pass must not match: %d", n)
	}
	e.want(e.do("DELETE", "/discover/swipes/"+a.ID, c.Token, nil), 204)
	e.want(e.do("DELETE", "/discover/swipes/"+a.ID, c.Token, nil), 404)
	if !contains(e.feedIDs(c), a.ID) {
		t.Fatal("undone pass must reappear")
	}
	e.want(e.do("DELETE", "/discover/swipes/"+b.ID, a.Token, nil), 409)

	// Unmatch removes the match and conversation for both, but the profiles do not reappear.
	mid := m["matchId"].(string)
	e.want(e.do("DELETE", "/matches/"+mid, c.Token, nil), 404)
	e.want(e.do("DELETE", "/matches/"+mid, a.Token, nil), 204)
	if n := count(e, `SELECT count(*) FROM conversations`); n != 0 {
		t.Fatalf("conversation must be removed with the match: %d", n)
	}
	if contains(e.feedIDs(a), b.ID) || contains(e.feedIDs(b), a.ID) {
		t.Fatal("unmatched users must not reappear")
	}
}

// Two users liking each other at the same instant must produce exactly one match, and at least
// one of the two requests must see it.
func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 15; round++ {
		a := e.newUser(spec{Name: "RaceA", Gender: "woman", InterestedIn: []string{"man"}})
		b := e.newUser(spec{Name: "RaceB", Gender: "man", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]map[string]any, 2)
		start := make(chan struct{})
		for i, p := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to user) {
				defer wg.Done()
				<-start
				results[i] = e.swipe(from, to, "like").json()
			}(i, p[0], p[1])
		}
		close(start)
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r["matched"] == true {
				matched++
			}
		}
		if matched < 1 {
			t.Fatalf("round %d: nobody saw the match: %v", round, results)
		}
		if n := count(e, `SELECT count(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID); n != 1 {
			t.Fatalf("round %d: %d matches", round, n)
		}
		if n := count(e, `SELECT count(*) FROM conversation_participants cp JOIN conversations c ON c.id = cp.conversation_id JOIN matches m ON m.id = c.match_id WHERE m.user_a_id = LEAST($1::uuid,$2::uuid)`, a.ID, b.ID); n != 2 {
			t.Fatalf("round %d: %d participants", round, n)
		}
	}
}

func TestSimultaneousDuplicateSwipesStoreOne(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{Name: "DupA", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.newUser(spec{Name: "DupB", Gender: "man", InterestedIn: []string{"woman"}})
	var wg sync.WaitGroup
	statuses := make([]int, 8)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = e.swipe(a, b, "like").Status
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, s := range statuses {
		if s == 200 {
			ok++
		} else if s != 409 {
			t.Fatalf("unexpected status %d", s)
		}
	}
	if ok != 1 {
		t.Fatalf("exactly one swipe must win, got %d (%v)", ok, statuses)
	}
}
