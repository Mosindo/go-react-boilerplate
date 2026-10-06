package main

import (
	"net/http"
	"strings"
	"testing"
)

// TestCriticalJourney walks the product's critical path through the public API
// only: sign up -> profile -> photos -> discovery -> like -> match ->
// conversation -> notifications.
func TestCriticalJourney(t *testing.T) {
	e := newEnv(t)

	// 1. Two people sign up and complete onboarding.
	alice := e.register("alice")
	bob := e.register("bob")
	e.onboard(alice, persona{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 29, Bio: "Randonnées et café."})
	e.onboard(bob, persona{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Age: 31})

	for _, u := range []user{alice, bob} {
		self := e.expect(e.do(http.MethodGet, "/profile", u.Token, nil), http.StatusOK).json(t)
		if self["onboarding"].(map[string]any)["complete"] != true {
			t.Fatalf("onboarding should be complete: %v", self["onboarding"])
		}
	}

	// 2. Discovery: each sees the other, with photo URLs but no raw location.
	if !contains(e.discoverIDs(alice), bob.ID) || !contains(e.discoverIDs(bob), alice.ID) {
		t.Fatal("compatible users must discover each other")
	}

	// 3. Alice likes Bob: no match yet, and Bob leaves Alice's feed for good.
	first := e.swipe(alice, bob, "like", http.StatusCreated)
	if first.Matched {
		t.Fatal("a one-sided like must not match")
	}
	if contains(e.discoverIDs(alice), bob.ID) {
		t.Fatal("a processed profile must never reappear")
	}

	// 4. Bob sees Alice in "likes received", then likes back: it's a match.
	likes := e.expect(e.do(http.MethodGet, "/likes/received", bob.Token, nil), http.StatusOK).json(t)["profiles"].([]any)
	if len(likes) != 1 || likes[0].(map[string]any)["userId"] != alice.ID {
		t.Fatalf("Bob should see Alice's like: %v", likes)
	}
	second := e.swipe(bob, alice, "like", http.StatusCreated)
	if !second.Matched || second.MatchID == "" {
		t.Fatalf("reciprocal like must match: %+v", second)
	}
	matchID := second.MatchID

	// 5. Both get a match notification.
	for _, u := range []user{alice, bob} {
		n := e.expect(e.do(http.MethodGet, "/notifications", u.Token, nil), http.StatusOK).json(t)
		items := n["notifications"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["type"] != "new_match" || n["unreadCount"].(float64) != 1 {
			t.Fatalf("expected one unread new_match notification: %v", n)
		}
	}

	// 6. Conversation: Alice writes, Bob reads and answers.
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": "Salut Bob !"}), http.StatusCreated)
	list := e.expect(e.do(http.MethodGet, "/matches", bob.Token, nil), http.StatusOK).json(t)
	bobMatch := list["matches"].([]any)[0].(map[string]any)
	if bobMatch["unreadCount"].(float64) != 1 || list["unreadTotal"].(float64) != 1 || bobMatch["lastMessage"].(map[string]any)["body"] != "Salut Bob !" {
		t.Fatalf("Bob should have 1 unread message: %v", list)
	}
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/read", bob.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/matches/"+matchID+"/messages", bob.Token, map[string]string{"body": "Salut Alice, ravi de te rencontrer."}), http.StatusCreated)

	thread := e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages", alice.Token, nil), http.StatusOK).json(t)
	msgs := thread["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["senderId"] != alice.ID || msgs[1].(map[string]any)["senderId"] != bob.ID {
		t.Fatalf("thread must be chronological: %v", msgs)
	}
	if thread["peerLastReadAt"] == nil {
		t.Fatal("Alice should see that Bob has read her message")
	}

	// 7. A photo is served only to people allowed to see its owner.
	photo := bobMatch["user"].(map[string]any)["primaryPhotoUrl"].(string)
	if !strings.HasPrefix(photo, "/photos/") {
		t.Fatalf("match should carry the peer's primary photo, got %q", photo)
	}
	img := e.expect(e.do(http.MethodGet, photo, bob.Token, nil), http.StatusOK)
	if img.Header.Get("Content-Type") != "image/jpeg" || !strings.Contains(img.Header.Get("Cache-Control"), "private") {
		t.Fatalf("unexpected photo headers: %v", img.Header)
	}
}
