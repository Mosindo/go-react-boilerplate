package app_test

import (
	"sync"
	"testing"
)

func TestDiscoveryRespectsMutualPreferences(t *testing.T) {
	e := newEnv(t)
	maxDist := 20
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}, Age: 30, MinAge: 25, MaxAge: 40})
	bob := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 32, MinAge: 25, MaxAge: 40})
	carl := e.signup("Carl", signupOpts{Gender: "man", InterestedIn: []string{"man"}, Age: 31})                           // not interested in women
	dave := e.signup("Dave", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 45, MinAge: 18, MaxAge: 99}) // outside Alice's age range
	eve := e.signup("Eve", signupOpts{Gender: "woman", InterestedIn: []string{"man"}, Age: 29})                           // wrong gender for Alice
	finn := e.signup("Finn", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 33, MinAge: 18, MaxAge: 28}) // Alice (30) outside his range
	gus := e.signup("Gus", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 34, MaxDistance: &maxDist})    // has a distance cap, Alice has no location
	hidden := e.signup("Hidden", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	e.expect(hidden.do("PUT", "/profile", map[string]any{"firstName": "Hidden", "birthDate": birthDateForAge(33), "gender": "man", "discoverable": false}), 200)
	nophoto := e.signup("Nophoto", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Age: 33, NoPhoto: true})
	_ = []*testUser{carl, dave, eve, finn, gus, nophoto}

	got := alice.feedIDs()
	if len(got) != 1 || got[0] != bob.ID {
		t.Fatalf("Alice should only see Bob, got %v", got)
	}
	if !contains(bob.feedIDs(), alice.ID) {
		t.Fatal("Bob should see Alice")
	}
	if contains(carl.feedIDs(), alice.ID) {
		t.Fatal("Carl is not interested in women")
	}
	// Incomplete profiles can't browse.
	fresh := e.register("fresh")
	if r := fresh.do("GET", "/discovery", nil); r.Status != 200 || string(r.Body) != "{\"items\":[]}" {
		t.Fatalf("incomplete profile should get an empty feed: %d %s", r.Status, r.Body)
	}
	if st, _ := fresh.swipe(bob.ID, "like"); st != 403 {
		t.Fatalf("incomplete profile swipe should be 403, got %d", st)
	}
	e.expect(alice.do("GET", "/discovery?limit=0", nil), 400)
	e.expect(alice.do("GET", "/discovery?limit=21", nil), 400)
	e.expect(alice.do("GET", "/discovery?limit=abc", nil), 400)
}

func TestSwipeAndMatchRules(t *testing.T) {
	e := newEnv(t)
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	carl := e.signup("Carl", signupOpts{Gender: "man", InterestedIn: []string{"man"}})
	cara := e.signup("Cara", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})

	// Illegal targets: self, unknown, ineligible, malformed.
	if st, _ := alice.swipe(alice.ID, "like"); st != 404 {
		t.Fatalf("self swipe: %d", st)
	}
	if st, _ := alice.swipe("00000000-0000-0000-0000-000000000000", "like"); st != 404 {
		t.Fatalf("unknown swipe: %d", st)
	}
	if st, _ := alice.swipe(carl.ID, "like"); st != 404 {
		t.Fatalf("ineligible swipe (orientation): %d", st)
	}
	if st, _ := alice.swipe(cara.ID, "like"); st != 404 {
		t.Fatalf("ineligible swipe (same gender): %d", st)
	}
	e.expect(alice.do("POST", "/discovery/swipes", map[string]string{"userId": "nope", "action": "like"}), 400)
	e.expect(alice.do("POST", "/discovery/swipes", map[string]string{"userId": bob.ID, "action": "superlike"}), 400)

	// A one-sided like is not a match, and likes cannot be repeated.
	st, res := alice.swipe(bob.ID, "like")
	if st != 200 || res.Result != "liked" || res.Match != nil {
		t.Fatalf("expected liked, got %d %+v", st, res)
	}
	if _, res := alice.swipe(bob.ID, "like"); res.Result != "already_swiped" {
		t.Fatalf("repeated like: %+v", res)
	}
	if _, res := alice.swipe(bob.ID, "pass"); res.Result != "already_swiped" {
		t.Fatalf("changing a decision is not allowed: %+v", res)
	}
	if contains(alice.feedIDs(), bob.ID) {
		t.Fatal("already swiped profiles must not reappear")
	}
	var matches struct {
		Items []any `json:"items"`
	}
	e.expect(alice.do("GET", "/matches", nil), 200).decode(t, &matches)
	if len(matches.Items) != 0 {
		t.Fatal("no match expected yet")
	}

	// Bob is shown Alice (she liked him) and likes back: match, visible by both.
	if !contains(bob.feedIDs(), alice.ID) {
		t.Fatal("Bob should see Alice")
	}
	st, res = bob.swipe(alice.ID, "like")
	if st != 200 || res.Result != "matched" || res.Match == nil || res.Match.ConversationID == "" {
		t.Fatalf("expected match, got %d %+v", st, res)
	}
	if _, res := bob.swipe(alice.ID, "like"); res.Result != "already_swiped" {
		t.Fatalf("repeat after match: %+v", res)
	}
	for _, u := range []*testUser{alice, bob} {
		e.expect(u.do("GET", "/matches", nil), 200).decode(t, &matches)
		if len(matches.Items) != 1 {
			t.Fatalf("expected exactly one match, got %d", len(matches.Items))
		}
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches`).Scan(&n)
	if n != 1 {
		t.Fatalf("matches rows: %d", n)
	}
}

func TestPassNeverMatches(t *testing.T) {
	e := newEnv(t)
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	if _, res := alice.swipe(bob.ID, "pass"); res.Result != "passed" {
		t.Fatalf("pass: %+v", res)
	}
	if _, res := bob.swipe(alice.ID, "like"); res.Result != "liked" || res.Match != nil {
		t.Fatalf("like after pass must not match: %+v", res)
	}
	if contains(alice.feedIDs(), bob.ID) {
		t.Fatal("passed profile reappeared")
	}
}

// Two people liking each other at the same instant must produce exactly one match.
func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 8; round++ {
		a := e.signup("Ann", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
		b := e.signup("Ben", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, p := range []struct{ from, to *testUser }{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, results[i] = p.from.swipe(p.to.ID, "like")
			}()
		}
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r.Result == "matched" {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("round %d: expected exactly one side to observe the match, got %+v", round, results)
		}
		var n int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: %d match rows", round, n)
		}
	}
}

func TestRankingPrefersSharedInterestsAndPeopleWhoLikedYou(t *testing.T) {
	e := newEnv(t)
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}, Interests: []string{"travel", "music", "yoga"}})
	none := e.signup("None", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	shared := e.signup("Shared", signupOpts{Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"travel", "music"}})
	admirer := e.signup("Admirer", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	// The admirer can only like Alice after seeing her; she is eligible for him.
	if _, res := admirer.swipe(alice.ID, "like"); res.Result != "liked" {
		t.Fatalf("admirer like: %+v", res)
	}
	got := alice.feedIDs()
	if len(got) != 3 || got[0] != admirer.ID || got[1] != shared.ID || got[2] != none.ID {
		t.Fatalf("unexpected ranking %v (admirer=%s shared=%s none=%s)", got, admirer.ID, shared.ID, none.ID)
	}
	var feed struct {
		Items []struct {
			UserID          string   `json:"userId"`
			SharedInterests []string `json:"sharedInterests"`
		} `json:"items"`
	}
	e.expect(alice.do("GET", "/discovery", nil), 200).decode(t, &feed)
	if len(feed.Items[1].SharedInterests) != 2 {
		t.Fatalf("expected 2 shared interests: %+v", feed.Items[1])
	}
}

func TestProfileVisibility(t *testing.T) {
	e := newEnv(t)
	alice := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	e.expect(alice.do("GET", "/profiles/"+bob.ID, nil), 200)
	e.expect(alice.do("GET", "/profiles/not-a-uuid", nil), 404)
	e.expect(alice.do("GET", "/profiles/00000000-0000-0000-0000-000000000000", nil), 404)

	// Not discoverable -> invisible, unless matched.
	e.expect(bob.do("PUT", "/profile", map[string]any{"firstName": "Bob", "birthDate": birthDateForAge(30), "gender": "man", "discoverable": false}), 200)
	e.expect(alice.do("GET", "/profiles/"+bob.ID, nil), 404)
	e.expect(bob.do("PUT", "/profile", map[string]any{"firstName": "Bob", "birthDate": birthDateForAge(30), "gender": "man", "discoverable": true}), 200)
	matchWith(t, alice, bob)
	e.expect(bob.do("PUT", "/profile", map[string]any{"firstName": "Bob", "birthDate": birthDateForAge(30), "gender": "man", "discoverable": false}), 200)
	e.expect(alice.do("GET", "/profiles/"+bob.ID, nil), 200)

	// Blocking hides both ways.
	e.expect(alice.do("POST", "/blocks", map[string]string{"userId": bob.ID}), 204)
	e.expect(alice.do("GET", "/profiles/"+bob.ID, nil), 404)
	e.expect(bob.do("GET", "/profiles/"+alice.ID, nil), 404)
}
