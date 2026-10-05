package main

import (
	"net/http"
	"sync"
	"testing"
)

// pair builds a woman/man pair in Paris who fit each other's preferences.
func (e *testEnv) pair() (session, session) {
	e.t.Helper()
	a, b := e.register("alice"), e.register("bob")
	e.onboard(a, profileSpec{Name: "Alice", Gender: "woman", Interested: []string{"man"}, Lat: parisLat, Lng: parisLng, BirthYear: 1994})
	e.onboard(b, profileSpec{Name: "Bob", Gender: "man", Interested: []string{"woman"}, Lat: 48.87, Lng: 2.33, BirthYear: 1991})
	return a, b
}

func TestDiscoveryFiltersAreMutual(t *testing.T) {
	env := newEnv(t)
	me := env.register("me")
	env.onboard(me, profileSpec{Name: "Mia", Gender: "woman", Interested: []string{"man"}, Lat: parisLat, Lng: parisLng, BirthYear: 1994, MinAge: 28, MaxAge: 40, MaxDist: 50})

	mk := func(label string, spec profileSpec) session {
		s := env.register(label)
		env.onboard(s, spec)
		return s
	}
	good := mk("good", profileSpec{Name: "Good", Gender: "man", Interested: []string{"woman"}, Lat: 48.90, Lng: 2.30, BirthYear: 1992})
	farAway := mk("far", profileSpec{Name: "Far", Gender: "man", Interested: []string{"woman"}, Lat: lyonLat, Lng: lyonLng, BirthYear: 1992})
	tooYoung := mk("young", profileSpec{Name: "Young", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng, BirthYear: 2004})
	wrongGender := mk("woman", profileSpec{Name: "Woman", Gender: "woman", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng, BirthYear: 1992})
	notInterested := mk("notinterested", profileSpec{Name: "Nope", Gender: "man", Interested: []string{"man"}, Lat: parisLat, Lng: parisLng, BirthYear: 1992})
	ageMismatch := mk("agemismatch", profileSpec{Name: "Picky", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng, BirthYear: 1992, MinAge: 18, MaxAge: 25})
	shortRange := mk("shortrange", profileSpec{Name: "Close", Gender: "man", Interested: []string{"woman"}, Lat: 48.60, Lng: 2.35, BirthYear: 1992, MaxDist: 5})
	noPhoto := mk("nophoto", profileSpec{Name: "Ghost", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng, BirthYear: 1992, NoPhoto: true})
	hidden := mk("hidden", profileSpec{Name: "Hidden", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng, BirthYear: 1992})
	env.do(http.MethodPut, "/profile/privacy", hidden.Token, map[string]any{"isDiscoverable": false, "showDistance": true}).expect(t, http.StatusOK)

	feed := env.discover(me)
	if !hasCard(feed, good.ID) {
		t.Fatalf("compatible profile must be proposed")
	}
	for name, s := range map[string]session{
		"too far": farAway, "too young for my range": tooYoung, "gender I do not look for": wrongGender,
		"not interested in me": notInterested, "I am outside their age range": ageMismatch,
		"I am outside their distance": shortRange, "no photo": noPhoto, "paused profile": hidden,
	} {
		if hasCard(feed, s.ID) {
			t.Errorf("%s must not be proposed", name)
		}
	}
	if hasCard(feed, me.ID) {
		t.Error("my own profile must not be proposed")
	}
	for _, c := range feed {
		if c.Age < 28 || c.Age > 40 {
			t.Errorf("age filter ignored: %+v", c)
		}
		if c.DistanceKm == nil || *c.DistanceKm > 50 {
			t.Errorf("distance filter ignored or distance missing: %+v", c)
		}
	}
}

func TestLikesMatchAndNoDuplicates(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()

	if !hasCard(env.discover(a), b.ID) || !hasCard(env.discover(b), a.ID) {
		t.Fatal("both users should see each other")
	}

	var first swipeBody
	env.swipe(a, b.ID, "like").expect(t, http.StatusOK).json(t, &first)
	if first.Matched {
		t.Fatal("a one-sided like must not create a match")
	}
	// Repeated or contradictory swipes on the same profile are refused.
	env.swipe(a, b.ID, "like").expect(t, http.StatusConflict)
	env.swipe(a, b.ID, "pass").expect(t, http.StatusConflict)
	// Already-treated profiles never come back.
	if hasCard(env.discover(a), b.ID) {
		t.Fatal("a liked profile must leave the feed")
	}
	// The person who liked me is boosted to the top of my feed.
	if feed := env.discover(b); len(feed) == 0 || feed[0].ID != a.ID {
		t.Fatalf("admirer should come first, got %+v", feed)
	}

	var second swipeBody
	env.swipe(b, a.ID, "like").expect(t, http.StatusOK).json(t, &second)
	if !second.Matched || second.Match == nil || second.Match.ConversationID == "" {
		t.Fatalf("reciprocal like must match: %+v", second)
	}

	for _, s := range []session{a, b} {
		var list struct {
			Matches []struct {
				ID   string `json:"id"`
				User card   `json:"user"`
			} `json:"matches"`
		}
		env.do(http.MethodGet, "/matches", s.Token, nil).expect(t, http.StatusOK).json(t, &list)
		if len(list.Matches) != 1 || list.Matches[0].ID != second.Match.ID {
			t.Fatalf("each user must see the match once: %+v", list)
		}
	}
	var count int
	_ = env.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}
	// Once matched nobody reappears in the feed.
	if hasCard(env.discover(a), b.ID) || hasCard(env.discover(b), a.ID) {
		t.Fatal("matched users must not be proposed again")
	}
}

func TestPassNeverMatches(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	env.swipe(a, b.ID, "pass").expect(t, http.StatusOK)
	var r swipeBody
	env.swipe(b, a.ID, "like").expect(t, http.StatusOK).json(t, &r)
	if r.Matched {
		t.Fatal("a pass followed by a like must not match")
	}
	if hasCard(env.discover(a), b.ID) {
		t.Fatal("a passed profile must not come back")
	}
	// The pass is final: a later like on the same profile is refused.
	env.swipe(a, b.ID, "like").expect(t, http.StatusConflict)
}

func TestSwipeValidation(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	stranger := env.register("stranger")
	incomplete := env.register("incomplete")

	env.swipe(a, a.ID, "like").expect(t, http.StatusBadRequest)
	env.swipe(a, "not-a-uuid", "like").expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, "/swipes", a.Token, map[string]string{"targetUserId": b.ID, "action": "superlike"}).expect(t, http.StatusBadRequest)
	env.swipe(a, "11111111-1111-1111-1111-111111111111", "like").expect(t, http.StatusNotFound)
	// A user without a profile cannot be targeted, nor swipe.
	env.swipe(a, stranger.ID, "like").expect(t, http.StatusNotFound)
	env.swipe(incomplete, b.ID, "like").expect(t, http.StatusConflict)
	env.do(http.MethodPost, "/swipes", "", map[string]string{"targetUserId": b.ID, "action": "like"}).expect(t, http.StatusUnauthorized)

	// A profile that does not fit my preferences cannot be liked by crafting the request.
	far := env.register("far")
	env.onboard(far, profileSpec{Name: "Far", Gender: "man", Interested: []string{"woman"}, Lat: lyonLat, Lng: lyonLng})
	env.swipe(a, far.ID, "like").expect(t, http.StatusNotFound)
}

func TestConcurrentReciprocalLikesCreateOneMatch(t *testing.T) {
	env := newEnv(t)
	for round := 0; round < 5; round++ {
		a, b := env.pair()
		var wg sync.WaitGroup
		results := make([]swipeBody, 2)
		for i, pair := range [][2]session{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				env.swipe(pair[0], pair[1].ID, "like").expect(t, http.StatusOK).json(t, &results[i])
			}()
		}
		wg.Wait()
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: simultaneous reciprocal likes produced no match", round)
		}
		var n int
		_ = env.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
		var conv int
		_ = env.pool.QueryRow(t.Context(), `SELECT count(*) FROM conversations WHERE match_id IN (SELECT id FROM matches WHERE user_a IN ($1,$2))`, a.ID, b.ID).Scan(&conv)
		if conv != 1 {
			t.Fatalf("round %d: expected one conversation, got %d", round, conv)
		}
	}
}

func TestUnmatch(t *testing.T) {
	env := newEnv(t)
	a, b := env.pair()
	m := env.match(a, b)
	other := env.register("outsider")

	env.do(http.MethodDelete, "/matches/"+m.ID, other.Token, nil).expect(t, http.StatusNotFound)
	env.do(http.MethodDelete, "/matches/"+m.ID, a.Token, nil).expect(t, http.StatusNoContent)
	env.do(http.MethodDelete, "/matches/"+m.ID, a.Token, nil).expect(t, http.StatusNotFound)

	// The conversation is gone for both, and they do not reappear to each other.
	env.do(http.MethodGet, "/conversations/"+m.ConversationID+"/messages", b.Token, nil).expect(t, http.StatusNotFound)
	if hasCard(env.discover(a), b.ID) || hasCard(env.discover(b), a.ID) {
		t.Fatal("unmatched users must not be proposed again")
	}
}

type matchInfo struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
}

// match makes a and b like each other and returns the resulting match.
func (e *testEnv) match(a, b session) matchInfo {
	e.t.Helper()
	e.swipe(a, b.ID, "like").expect(e.t, http.StatusOK)
	var r swipeBody
	e.swipe(b, a.ID, "like").expect(e.t, http.StatusOK).json(e.t, &r)
	if !r.Matched {
		e.t.Fatal("expected a match")
	}
	return matchInfo{ID: r.Match.ID, ConversationID: r.Match.ConversationID}
}
